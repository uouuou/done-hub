package relay

import (
	"done-hub/common"
	"done-hub/common/config"
	"done-hub/common/logger"
	"done-hub/common/utils"
	"done-hub/model"
	providersBase "done-hub/providers/base"
	"done-hub/types"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RelayRerank(c *gin.Context) {
	relay := NewRelayRerank(c)

	if err := relay.setRequest(); err != nil {
		common.AbortWithErr(c, http.StatusBadRequest, &types.RerankError{Detail: err.Error()})
		return
	}

	if err := relay.setProvider(relay.getOriginalModel()); err != nil {
		common.AbortWithErr(c, http.StatusServiceUnavailable, &types.RerankError{Detail: err.Error()})
		return
	}

	apiErr, done := RelayHandler(relay)
	if apiErr == nil {
		return
	}

	channel := relay.getProvider().GetChannel()
	go processChannelRelayError(c.Request.Context(), channel.Id, channel.Name, apiErr, channel.Type)

	retryTimes := config.RetryTimes
	if done || !shouldRetry(c, apiErr, channel.Type) {
		logger.LogError(c.Request.Context(), fmt.Sprintf("relay error happen, status code is %d, won't retry in this case", apiErr.StatusCode))
		retryTimes = 0
	}

	// 在重试开始前计算并缓存总渠道数，避免重试过程中动态变化
	groupName := c.GetString("token_group")
	if groupName == "" {
		groupName = c.GetString("group")
	}
	modelName := c.GetString("new_model")
	totalChannelsAtStart := model.ChannelGroup.CountAvailableChannels(groupName, modelName)
	c.Set("total_channels_at_start", totalChannelsAtStart)
	c.Set("attempt_count", 1) // 初始化尝试计数

	for i := retryTimes; i > 0; i-- {
		// 冻结通道
		shouldCooldowns(c, channel, apiErr)
		if err := relay.setProvider(relay.getOriginalModel()); err != nil {
			continue
		}

		channel = relay.getProvider().GetChannel()

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
		filters := buildChannelFilters(c, modelName)
		skipChannelIds, _ := utils.GetGinValue[[]int](c, "skip_channel_ids")
		tempFilters := append(filters, model.FilterChannelId(skipChannelIds))
		remainChannels := model.ChannelGroup.CountAvailableChannels(groupName, modelName, tempFilters...)

		logger.LogError(c.Request.Context(), fmt.Sprintf("using channel #%d(%s) to retry (attempt %d/%d, remain channels %d, total channels %d)", channel.Id, channel.Name, attemptCount, totalChannels, remainChannels, totalChannels))

		apiErr, done = RelayHandler(relay)
		if apiErr == nil {
			return
		}
		go processChannelRelayError(c.Request.Context(), channel.Id, channel.Name, apiErr, channel.Type)
		if done || !shouldRetry(c, apiErr, channel.Type) {
			break
		}
	}

	if apiErr != nil {
		if apiErr.StatusCode == http.StatusTooManyRequests {
			apiErr.OpenAIError.Message = "当前分组上游负载已饱和，请稍后再试"
		}
		relayRerankResponseWithErr(c, apiErr)
	}
}

type relayRerank struct {
	relayBase
	request types.RerankRequest
}

func NewRelayRerank(c *gin.Context) *relayRerank {
	relay := &relayRerank{}
	relay.c = c
	return relay
}

func (r *relayRerank) setRequest() error {
	if err := common.UnmarshalBodyReusable(r.c, &r.request); err != nil {
		return err
	}

	r.setOriginalModel(r.request.Model)

	return nil
}

func (r *relayRerank) getPromptTokens() (int, error) {
	channel := r.provider.GetChannel()
	return common.CountTokenRerankMessages(r.request, r.modelName, channel.PreCost), nil
}

func (r *relayRerank) send() (err *types.OpenAIErrorWithStatusCode, done bool) {
	chatProvider, ok := r.provider.(providersBase.RerankInterface)
	if !ok {
		err = common.StringErrorWrapperLocal("channel not implemented", "channel_error", http.StatusServiceUnavailable)
		done = true
		return
	}

	r.request.Model = r.modelName

	var response *types.RerankResponse
	response, err = chatProvider.CreateRerank(&r.request)
	if err != nil {
		return
	}
	err = responseJsonClient(r.c, response)

	if err != nil {
		done = true
	}

	return
}
