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

	for i := retryTimes; i > 0; i-- {
		shouldCooldowns(c, channel, apiErr)
		if recraftProvider, err = getRecraftProvider(c, model); err != nil {
			continue
		}

		channel = recraftProvider.GetChannel()

		// 计算当前实际可用的渠道数量（包括当前失败的渠道）
		groupName := c.GetString("token_group")
		if groupName == "" {
			groupName = c.GetString("group")
		}
		modelName := c.GetString("new_model")

		// 构建包含当前失败渠道的过滤器
		filters := buildChannelFiltersForRecraftAI(c, modelName)
		// 添加当前失败的渠道到跳过列表中进行计算
		skipChannelIds, _ := utils.GetGinValue[[]int](c, "skip_channel_ids")
		skipChannelIds = append(skipChannelIds, channel.Id)
		tempFilters := append(filters, modelPkg.FilterChannelId(skipChannelIds))

		availableChannels := modelPkg.ChannelGroup.CountAvailableChannels(groupName, modelName, tempFilters...)

		logger.LogError(c.Request.Context(), fmt.Sprintf("using channel #%d(%s) to retry (remain times %d, available channels %d)", channel.Id, channel.Name, i, availableChannels))

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
