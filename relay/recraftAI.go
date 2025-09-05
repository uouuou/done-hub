package relay

import (
	"done-hub/common"
	"done-hub/common/config"
	"done-hub/common/logger"
	"done-hub/common/utils"
	"done-hub/metrics"
	modelPkg "done-hub/model"
	"done-hub/providers/recraftAI"
	"done-hub/relay/relay_util"
	"done-hub/types"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// buildChannelFiltersForRecraftAI 为 RecraftAI 构建渠道过滤器列表
func buildChannelFiltersForRecraftAI(c *gin.Context, modelName string) []modelPkg.ChannelsFilterFunc {
	var filters []modelPkg.ChannelsFilterFunc

	if skipOnlyChat := c.GetBool("skip_only_chat"); skipOnlyChat {
		filters = append(filters, modelPkg.FilterOnlyChat())
	}

	if skipChannelIds, ok := utils.GetGinValue[[]int](c, "skip_channel_ids"); ok {
		filters = append(filters, modelPkg.FilterChannelId(skipChannelIds))
	}

	if types, exists := c.Get("allow_channel_type"); exists {
		if allowTypes, ok := types.([]int); ok {
			filters = append(filters, modelPkg.FilterChannelTypes(allowTypes))
		}
	}

	return filters
}

func RelayRecraftAI(c *gin.Context) {
	model := Path2RecraftAIModel(c.Request.URL.Path)

	usage := &types.Usage{
		PromptTokens: 1,
	}

	recraftProvider, err := getRecraftProvider(c, model)
	if err != nil {
		common.AbortWithMessage(c, http.StatusServiceUnavailable, err.Error())
		return
	}

	recraftProvider.SetUsage(usage)

	quota := relay_util.NewQuota(c, model, 1)
	if err := quota.PreQuotaConsumption(); err != nil {
		common.AbortWithMessage(c, http.StatusServiceUnavailable, err.Error())
		return
	}

	requestURL := strings.Replace(c.Request.URL.Path, "/recraftAI", "", 1)
	response, apiErr := recraftProvider.CreateRelay(requestURL)
	if apiErr == nil {
		quota.Consume(c, usage, false)

		metrics.RecordProvider(c, 200)
		errWithCode := responseMultipart(c, response)
		logger.LogError(c.Request.Context(), fmt.Sprintf("relay error happen %v, won't retry in this case", errWithCode))
		return
	}

	channel := recraftProvider.GetChannel()
	go processChannelRelayError(c.Request.Context(), channel.Id, channel.Name, apiErr, channel.Type)

	retryTimes := config.RetryTimes
	if !shouldRetry(c, apiErr, channel.Type) {
		logger.LogError(c.Request.Context(), fmt.Sprintf("relay error happen, status code is %d, won't retry in this case", apiErr.StatusCode))
		retryTimes = 0
	}

	// 在重试开始前计算并缓存总渠道数，避免重试过程中动态变化
	groupName := c.GetString("token_group")
	if groupName == "" {
		groupName = c.GetString("group")
	}
	modelName := c.GetString("new_model")
	totalChannelsAtStart := modelPkg.ChannelGroup.CountAvailableChannels(groupName, modelName)
	c.Set("total_channels_at_start", totalChannelsAtStart)
	c.Set("attempt_count", 1) // 初始化尝试计数

	for i := retryTimes; i > 0; i-- {
		shouldCooldowns(c, channel, apiErr)
		if recraftProvider, err = getRecraftProvider(c, model); err != nil {
			continue
		}

		channel = recraftProvider.GetChannel()

		// 计算渠道信息用于日志显示
		groupName := c.GetString("token_group")
		if groupName == "" {
			groupName = c.GetString("group")
		}
		modelName := c.GetString("new_model")

		// 使用请求开始时缓存的总渠道数，保持一致性
		totalChannels := c.GetInt("total_channels_at_start")

		// 更新尝试计数
		attemptCount := c.GetInt("attempt_count")
		c.Set("attempt_count", attemptCount+1)

		// 计算剩余可重试的渠道数（不包括当前渠道，因为当前渠道正在使用）
		filters := buildChannelFiltersForRecraftAI(c, modelName)
		skipChannelIds, _ := utils.GetGinValue[[]int](c, "skip_channel_ids")
		tempFilters := append(filters, modelPkg.FilterChannelId(skipChannelIds))
		remainChannels := modelPkg.ChannelGroup.CountAvailableChannels(groupName, modelName, tempFilters...)

		logger.LogError(c.Request.Context(), fmt.Sprintf("using channel #%d(%s) to retry (attempt %d/%d, remain channels %d, total channels %d)", channel.Id, channel.Name, attemptCount, totalChannels, remainChannels, totalChannels))

		response, apiErr := recraftProvider.CreateRelay(requestURL)
		if apiErr == nil {
			quota.Consume(c, usage, false)

			metrics.RecordProvider(c, 200)
			errWithCode := responseMultipart(c, response)
			logger.LogError(c.Request.Context(), fmt.Sprintf("relay error happen %v, won't retry in this case", errWithCode))
			return
		}

		go processChannelRelayError(c.Request.Context(), channel.Id, channel.Name, apiErr, channel.Type)
		if !shouldRetry(c, apiErr, channel.Type) {
			break
		}
	}

	quota.Undo(c)
	newErrWithCode := FilterOpenAIErr(c, apiErr)
	common.AbortWithErr(c, newErrWithCode.StatusCode, &newErrWithCode.OpenAIError)
}

func Path2RecraftAIModel(path string) string {
	parts := strings.Split(path, "/")
	lastPart := parts[len(parts)-1]

	return "recraft_" + lastPart
}

func getRecraftProvider(c *gin.Context, model string) (*recraftAI.RecraftProvider, error) {
	provider, _, fail := GetProvider(c, model)
	if fail != nil {
		// common.AbortWithMessage(c, http.StatusServiceUnavailable, fail.Error())
		return nil, fail
	}

	recraftProvider, ok := provider.(*recraftAI.RecraftProvider)
	if !ok {
		return nil, errors.New("provider not found")
	}

	return recraftProvider, nil
}
