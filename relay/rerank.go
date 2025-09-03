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

	for i := retryTimes; i > 0; i-- {
		// 冻结通道
		shouldCooldowns(c, channel, apiErr)
		if err := relay.setProvider(relay.getOriginalModel()); err != nil {
			continue
		}

		channel = relay.getProvider().GetChannel()

		// 计算当前实际可用的渠道数量（包括当前失败的渠道）
		groupName := c.GetString("token_group")
		if groupName == "" {
			groupName = c.GetString("group")
		}
		modelName := c.GetString("new_model")

		// 构建包含当前失败渠道的过滤器
		filters := buildChannelFilters(c, modelName)
		// 添加当前失败的渠道到跳过列表中进行计算
		skipChannelIds, _ := utils.GetGinValue[[]int](c, "skip_channel_ids")
		skipChannelIds = append(skipChannelIds, channel.Id)
		tempFilters := append(filters, model.FilterChannelId(skipChannelIds))

		availableChannels := model.ChannelGroup.CountAvailableChannels(groupName, modelName, tempFilters...)

		logger.LogError(c.Request.Context(), fmt.Sprintf("using channel #%d(%s) to retry (remain times %d, available channels %d)", channel.Id, channel.Name, i, availableChannels))

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
