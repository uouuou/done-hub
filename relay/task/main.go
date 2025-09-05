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

	// 在重试开始前计算并缓存总渠道数，避免重试过程中动态变化
	groupName := c.GetString("token_group")
	if groupName == "" {
		groupName = c.GetString("group")
	}
	modelName := taskAdaptor.GetModelName()
	totalChannelsAtStart := model.ChannelGroup.CountAvailableChannels(groupName, modelName)
	c.Set("total_channels_at_start", totalChannelsAtStart)
	c.Set("attempt_count", 1) // 初始化尝试计数

	channel := taskAdaptor.GetProvider().GetChannel()
	for i := retryTimes; i > 0; i-- {
		model.ChannelGroup.SetCooldowns(channel.Id, taskAdaptor.GetModelName())
		taskErr = taskAdaptor.SetProvider()
		if taskErr != nil {
			continue
		}

		channel = taskAdaptor.GetProvider().GetChannel()

		// 计算渠道信息用于日志显示
		groupName := c.GetString("token_group")
		if groupName == "" {
			groupName = c.GetString("group")
		}
		modelName := taskAdaptor.GetModelName()

		// 使用请求开始时缓存的总渠道数，保持一致性
		totalChannels := c.GetInt("total_channels_at_start")

		// 更新尝试计数
		attemptCount := c.GetInt("attempt_count")
		c.Set("attempt_count", attemptCount+1)

		// 计算剩余可重试的渠道数（不包括当前渠道，因为当前渠道正在使用）
		filters := buildTaskChannelFilters(c)
		skipChannelIds, _ := utils.GetGinValue[[]int](c, "skip_channel_ids")
		tempFilters := append(filters, model.FilterChannelId(skipChannelIds))
		remainChannels := model.ChannelGroup.CountAvailableChannels(groupName, modelName, tempFilters...)

		logger.LogError(c.Request.Context(), fmt.Sprintf("using channel #%d(%s) to retry (attempt %d/%d, remain channels %d, total channels %d)", channel.Id, channel.Name, attemptCount, totalChannels, remainChannels, totalChannels))

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
