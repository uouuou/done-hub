package task

import (
	"done-hub/common/config"
	"done-hub/common/logger"
	"done-hub/common/utils"
	"done-hub/metrics"
	"done-hub/model"
	"done-hub/relay/relay_util"
	"done-hub/relay/task/base"
	"done-hub/types"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

// buildTaskChannelFilters 为任务构建渠道过滤器列表
func buildTaskChannelFilters(c *gin.Context) []model.ChannelsFilterFunc {
	var filters []model.ChannelsFilterFunc

	if skipChannelIds, ok := utils.GetGinValue[[]int](c, "skip_channel_ids"); ok {
		filters = append(filters, model.FilterChannelId(skipChannelIds))
	}

	if types, exists := c.Get("allow_channel_type"); exists {
		if allowTypes, ok := types.([]int); ok {
			filters = append(filters, model.FilterChannelTypes(allowTypes))
		}
	}

	return filters
}

func RelayTaskSubmit(c *gin.Context) {
	var taskErr *base.TaskError
	taskAdaptor, err := GetTaskAdaptor(GetRelayMode(c), c)
	if err != nil {
		taskErr = base.StringTaskError(http.StatusBadRequest, "adaptor_not_found", "adaptor not found", true)
		c.JSON(http.StatusBadRequest, taskErr)
		return
	}

	taskErr = taskAdaptor.Init()
	if taskErr != nil {
		taskAdaptor.HandleError(taskErr)
		return
	}

	taskErr = taskAdaptor.SetProvider()
	if taskErr != nil {
		taskAdaptor.HandleError(taskErr)
		return
	}

	quotaInstance := relay_util.NewQuota(c, taskAdaptor.GetModelName(), 1000)
	if errWithOA := quotaInstance.PreQuotaConsumption(); errWithOA != nil {
		taskAdaptor.HandleError(base.OpenAIErrToTaskErr(errWithOA))
		return
	}

	taskErr = taskAdaptor.Relay()
	if taskErr == nil {
		CompletedTask(quotaInstance, taskAdaptor, c)
		// 返回结果
		taskAdaptor.GinResponse()
		metrics.RecordProvider(c, 200)
		return
	}

	quotaInstance.Undo(c)

	retryTimes := config.RetryTimes

	if !taskAdaptor.ShouldRetry(c, taskErr) {
		logger.LogError(c.Request.Context(), fmt.Sprintf("relay error happen, status code is %d, won't retry in this case", taskErr.StatusCode))
		retryTimes = 0
	}

	channel := taskAdaptor.GetProvider().GetChannel()
	for i := retryTimes; i > 0; i-- {
		model.ChannelGroup.SetCooldowns(channel.Id, taskAdaptor.GetModelName())
		taskErr = taskAdaptor.SetProvider()
		if taskErr != nil {
			continue
		}

		channel = taskAdaptor.GetProvider().GetChannel()

		// 计算当前实际可用的渠道数量（包括当前失败的渠道）
		groupName := c.GetString("token_group")
		if groupName == "" {
			groupName = c.GetString("group")
		}
		modelName := taskAdaptor.GetModelName()

		// 构建包含当前失败渠道的过滤器
		filters := buildTaskChannelFilters(c)
		// 添加当前失败的渠道到跳过列表中进行计算
		skipChannelIds, _ := utils.GetGinValue[[]int](c, "skip_channel_ids")
		skipChannelIds = append(skipChannelIds, channel.Id)
		tempFilters := append(filters, model.FilterChannelId(skipChannelIds))

		availableChannels := model.ChannelGroup.CountAvailableChannels(groupName, modelName, tempFilters...)

		logger.LogError(c.Request.Context(), fmt.Sprintf("using channel #%d(%s) to retry (remain times %d, available channels %d)", channel.Id, channel.Name, i, availableChannels))

		taskErr = taskAdaptor.Relay()
		if taskErr == nil {
			go CompletedTask(quotaInstance, taskAdaptor, c)
			return
		}

		quotaInstance.Undo(c)
		if !taskAdaptor.ShouldRetry(c, taskErr) {
			break
		}

	}

	if taskErr != nil {
		taskAdaptor.HandleError(taskErr)
	}

}

func CompletedTask(quotaInstance *relay_util.Quota, taskAdaptor base.TaskInterface, c *gin.Context) {
	quotaInstance.Consume(c, &types.Usage{CompletionTokens: 0, PromptTokens: 1, TotalTokens: 1}, false)

	task := taskAdaptor.GetTask()
	task.Quota = int(quotaInstance.GetInputRatio() * 1000)

	err := task.Insert()
	if err != nil {
		logger.SysError(fmt.Sprintf("task error: %s", err.Error()))
	}

	// 激活任务
	ActivateUpdateTaskBulk()
}

func GetRelayMode(c *gin.Context) int {
	relayMode := config.RelayModeUnknown
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/suno") {
		relayMode = config.RelayModeSuno
	} else if strings.HasPrefix(path, "/kling") {
		relayMode = config.RelayModeKling
	}

	return relayMode
}
