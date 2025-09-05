package relay

import (
	"bytes"
	"done-hub/common"
	"done-hub/common/config"
	"done-hub/common/logger"
	"done-hub/common/utils"
	"done-hub/metrics"
	"done-hub/model"
	"done-hub/relay/relay_util"
	"done-hub/types"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func Relay(c *gin.Context) {
	relay := Path2Relay(c, c.Request.URL.Path)
	if relay == nil {
		common.AbortWithMessage(c, http.StatusNotFound, "Not Found")
		return
	}

	// Apply pre-mapping before setRequest to ensure request body modifications take effect
	if err := applyPreMappingBeforeRequest(c); err != nil {
		openaiErr := common.StringErrorWrapperLocal(err.Error(), "one_hub_error", http.StatusBadRequest)
		relay.HandleJsonError(openaiErr)
		return
	}

	if err := relay.setRequest(); err != nil {
		openaiErr := common.StringErrorWrapperLocal(err.Error(), "one_hub_error", http.StatusBadRequest)
		relay.HandleJsonError(openaiErr)
		return
	}

	c.Set("is_stream", relay.IsStream())
	if err := relay.setProvider(relay.getOriginalModel()); err != nil {
		openaiErr := common.StringErrorWrapperLocal(err.Error(), "one_hub_error", http.StatusServiceUnavailable)
		relay.HandleJsonError(openaiErr)
		return
	}

	heartbeat := relay.SetHeartbeat(relay.IsStream())
	if heartbeat != nil {
		defer heartbeat.Close()
	}

	// 这里在setProvider的多次请求中没有去判定当前上下文是否存在第一次判定过的情况从而将数据写入异常的问题
	BillingOriginalModel := c.GetBool("billing_original_model")
	channel := relay.getProvider().GetChannel()
	// 获取用户设置的一些工具(设置一下原本请求模型和渠道的id防止日志异常)
	if channel.EnableSearch || c.GetBool("enable_search") {
		handleSearch(c, relay.getRequest().(*types.ChatCompletionRequest), true)
		c.Set("billing_original_model", BillingOriginalModel)
		c.Set("channel_id", channel.Id)
		c.Set("channel_type", channel.Type)
	}
	// 处理systemPrompt
	if channel.SystemPrompt != "" {
		systemPrompt(channel.SystemPrompt, relay.getRequest().(*types.ChatCompletionRequest))
	}

	apiErr, done := RelayHandler(relay)
	if apiErr == nil {
		metrics.RecordProvider(c, 200)
		return
	}

	go processChannelRelayError(c.Request.Context(), channel.Id, channel.Name, apiErr, channel.Type)

	retryTimes := config.RetryTimes
	if done || !shouldRetry(c, apiErr, channel.Type) {
		logger.LogError(c.Request.Context(), fmt.Sprintf("relay error happen, status code is %d, won't retry in this case", apiErr.StatusCode))
		retryTimes = 0
	}

	startTime := c.GetTime("requestStartTime")
	timeout := time.Duration(config.RetryTimeOut) * time.Second

	// 在重试开始前计算并缓存总渠道数，避免重试过程中动态变化
	groupName := c.GetString("token_group")
	if groupName == "" {
		groupName = c.GetString("group")
	}
	modelName := c.GetString("new_model")
	totalChannelsAtStart := model.ChannelGroup.CountAvailableChannels(groupName, modelName)

	// 实际重试次数 = min(配置的重试数, 可用渠道数)
	actualRetryTimes := retryTimes
	if totalChannelsAtStart < retryTimes {
		actualRetryTimes = totalChannelsAtStart
	}

	c.Set("total_channels_at_start", totalChannelsAtStart)
	c.Set("actual_retry_times", actualRetryTimes)
	c.Set("attempt_count", 1) // 初始化尝试计数

	// 记录初始失败 - 使用OpenAI风格的结构化日志
	logger.LogError(c.Request.Context(), fmt.Sprintf("retry_start model=%s total_channels=%d config_max_retries=%d actual_max_retries=%d initial_error=\"%s\" status_code=%d",
		modelName, totalChannelsAtStart, retryTimes, actualRetryTimes, apiErr.OpenAIError.Message, apiErr.StatusCode))

	for i := retryTimes; i > 0; i-- {
		// 冻结通道并记录是否应用了冷却
		cooldownApplied := shouldCooldowns(c, channel, apiErr)

		if time.Since(startTime) > timeout {
			logger.LogError(c.Request.Context(), fmt.Sprintf("retry_timeout elapsed_time=%.2fs timeout=%.2fs",
				time.Since(startTime).Seconds(), timeout.Seconds()))
			apiErr = common.StringErrorWrapperLocal("重试超时，上游负载已饱和，请稍后再试", "system_error", http.StatusTooManyRequests)
			break
		}

		if err := relay.setProvider(relay.getOriginalModel()); err != nil {
			logger.LogError(c.Request.Context(), fmt.Sprintf("retry_provider_error error=\"%s\"", err.Error()))
			break
		}

		channel = relay.getProvider().GetChannel()

		// 更新尝试计数
		attemptCount := c.GetInt("attempt_count")
		c.Set("attempt_count", attemptCount+1)

		// 计算剩余渠道数
		filters := buildChannelFilters(c, modelName)
		skipChannelIds, _ := utils.GetGinValue[[]int](c, "skip_channel_ids")
		tempFilters := append(filters, model.FilterChannelId(skipChannelIds))
		remainChannels := model.ChannelGroup.CountAvailableChannels(groupName, modelName, tempFilters...)

		// 获取实际重试次数
		actualRetryTimes := c.GetInt("actual_retry_times")

		// 记录重试尝试 - 按照OpenAI规范的结构化日志
		logger.LogError(c.Request.Context(), fmt.Sprintf("retry_attempt attempt=%d/%d channel_id=%d channel_name=\"%s\" remaining_channels=%d cooldown_applied=%t",
			attemptCount, actualRetryTimes, channel.Id, channel.Name, remainChannels, cooldownApplied))

		apiErr, done = RelayHandler(relay)
		if apiErr == nil {
			// 重试成功
			logger.LogError(c.Request.Context(), fmt.Sprintf("retry_success attempt=%d/%d channel_id=%d final_channel=\"%s\"",
				attemptCount, actualRetryTimes, channel.Id, channel.Name))
			metrics.RecordProvider(c, 200)
			return
		}

		// 记录重试失败
		logger.LogError(c.Request.Context(), fmt.Sprintf("retry_failed attempt=%d/%d channel_id=%d status_code=%d error_type=\"%s\" error=\"%s\"",
			attemptCount, actualRetryTimes, channel.Id, apiErr.StatusCode, apiErr.OpenAIError.Type, apiErr.OpenAIError.Message))

		go processChannelRelayError(c.Request.Context(), channel.Id, channel.Name, apiErr, channel.Type)
		if done || !shouldRetry(c, apiErr, channel.Type) {
			logger.LogError(c.Request.Context(), fmt.Sprintf("retry_stop_condition attempt=%d/%d done=%t should_retry=%t",
				attemptCount, actualRetryTimes, done, shouldRetry(c, apiErr, channel.Type)))
			break
		}
	}

	// 记录最终失败
	finalAttempt := c.GetInt("attempt_count")
	actualRetryTimes = c.GetInt("actual_retry_times")
	logger.LogError(c.Request.Context(), fmt.Sprintf("retry_exhausted total_attempts=%d actual_max_retries=%d config_max_retries=%d final_error=\"%s\" status_code=%d",
		finalAttempt, actualRetryTimes, retryTimes, apiErr.OpenAIError.Message, apiErr.StatusCode))

	if apiErr != nil {
		if heartbeat != nil && heartbeat.IsSafeWriteStream() {
			relay.HandleStreamError(apiErr)
			return
		}

		relay.HandleJsonError(apiErr)
	}
}

func RelayHandler(relay RelayBaseInterface) (err *types.OpenAIErrorWithStatusCode, done bool) {
	promptTokens, tonkeErr := relay.getPromptTokens()
	if tonkeErr != nil {
		err = common.ErrorWrapperLocal(tonkeErr, "token_error", http.StatusBadRequest)
		done = true
		return
	}

	usage := &types.Usage{
		PromptTokens: promptTokens,
	}

	relay.getProvider().SetUsage(usage)

	quota := relay_util.NewQuota(relay.getContext(), relay.getModelName(), promptTokens)
	if err = quota.PreQuotaConsumption(); err != nil {
		done = true
		return
	}

	err, done = relay.send()
	// 最后处理流式中断时计算tokens
	if usage.CompletionTokens == 0 && usage.TextBuilder.Len() > 0 {
		usage.CompletionTokens = common.CountTokenText(usage.TextBuilder.String(), relay.getModelName())
		usage.TotalTokens = usage.PromptTokens + usage.CompletionTokens
	}
	if err != nil {
		quota.Undo(relay.getContext())
		return
	}

	quota.SetFirstResponseTime(relay.GetFirstResponseTime())

	quota.Consume(relay.getContext(), usage, relay.IsStream())

	return
}

func shouldCooldowns(c *gin.Context, channel *model.Channel, apiErr *types.OpenAIErrorWithStatusCode) bool {
	modelName := c.GetString("new_model")
	channelId := channel.Id
	cooldownApplied := false

	// 如果是频率限制，冻结通道
	if apiErr.StatusCode == http.StatusTooManyRequests {
		model.ChannelGroup.SetCooldowns(channelId, modelName)
		cooldownApplied = true
		logger.LogError(c.Request.Context(), fmt.Sprintf("channel_cooldown channel_id=%d model=\"%s\" duration=%ds reason=\"rate_limit\"",
			channelId, modelName, config.RetryCooldownSeconds))
	}

	skipChannelIds, ok := utils.GetGinValue[[]int](c, "skip_channel_ids")
	if !ok {
		skipChannelIds = make([]int, 0)
	}

	skipChannelIds = append(skipChannelIds, channelId)
	c.Set("skip_channel_ids", skipChannelIds)

	return cooldownApplied
}

// applies pre-mapping before setRequest to ensure modifications take effect
func applyPreMappingBeforeRequest(c *gin.Context) error {
	// check if this is a chat completion request that needs pre-mapping
	path := c.Request.URL.Path
	if !(strings.HasPrefix(path, "/v1/chat/completions") || strings.HasPrefix(path, "/v1/completions")) {
		return errors.New("not a chat completion request")
	}

	bodyBytes, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return err
	}
	c.Request.Body.Close()

	// Use defer to ensure request body is always restored
	var finalBodyBytes = bodyBytes // default to original body
	defer func() {
		c.Request.Body = io.NopCloser(bytes.NewBuffer(finalBodyBytes))
	}()

	var requestBody struct {
		Model string `json:"model"`
	}
	if err := json.Unmarshal(bodyBytes, &requestBody); err != nil || requestBody.Model == "" {
		return errors.New("invalid request body")
	}

	provider, _, err := GetProvider(c, requestBody.Model)
	if err != nil {
		return err
	}

	customParams, err := provider.CustomParameterHandler()
	if err != nil || customParams == nil {
		return err
	}

	preAdd, exists := customParams["pre_add"]
	if !exists || preAdd != true {
		return errors.New("pre_add is not true")
	}

	var requestMap map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &requestMap); err != nil {
		return err
	}

	// Apply custom parameter merging
	modifiedRequestMap := mergeCustomParamsForPreMapping(requestMap, customParams)

	// Convert back to JSON - if successful, use modified body; otherwise use original
	if modifiedBodyBytes, err := json.Marshal(modifiedRequestMap); err == nil {
		finalBodyBytes = modifiedBodyBytes
	}
	return nil
}
