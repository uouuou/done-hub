package controller

import (
	"done-hub/common"
	"done-hub/common/config"
	"done-hub/common/logger"
	"done-hub/common/notify"
	"done-hub/model"
	"done-hub/types"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/singleflight"
)

var disableGroup singleflight.Group

func shouldEnableChannel(err error, openAIErr *types.OpenAIErrorWithStatusCode) bool {
	if !config.AutomaticEnableChannelEnabled {
		return false
	}
	if err != nil {
		return false
	}
	if openAIErr != nil {
		return false
	}
	return true
}

func ShouldDisableChannel(channelType int, err *types.OpenAIErrorWithStatusCode) bool {
	if !config.AutomaticDisableChannelEnabled || err == nil || err.LocalError {
		return false
	}

	// 状态码检查
	if err.StatusCode == http.StatusUnauthorized {
		return true
	}
	if err.StatusCode == http.StatusForbidden && channelType == config.ChannelTypeGemini {
		return true
	}

	// 错误代码检查
	switch err.OpenAIError.Code {
	case "invalid_api_key", "account_deactivated", "billing_not_active":
		return true
	}

	// 错误类型检查
	switch err.OpenAIError.Type {
	case "insufficient_quota", "authentication_error", "permission_error", "forbidden":
		return true
	}

	switch err.OpenAIError.Param {
	case "PERMISSIONDENIED":
		return true
	}

	return common.DisableChannelKeywordsInstance.IsContains(err.OpenAIError.Message)
}

// disable & notify
func DisableChannel(channelId int, channelName string, reason string, sendNotify bool) {
	key := fmt.Sprintf("disable_channel_%d", channelId)

	// 使用 singleflight 确保同一渠道的并发禁用请求只执行一次
	_, err, _ := disableGroup.Do(key, func() (interface{}, error) {
		// 检查渠道当前状态，避免重复禁用和重复发送邮件
		channel, err := model.GetChannelById(channelId)
		if err != nil {
			return nil, err
		}

		// 如果渠道已经被禁用，不需要重复操作
		if channel.Status == config.ChannelStatusAutoDisabled || channel.Status == config.ChannelStatusManuallyDisabled {
			return nil, nil
		}

		// 执行禁用操作
		model.UpdateChannelStatusById(channelId, config.ChannelStatusAutoDisabled)

		// 发送通知
		if sendNotify {
			subject := fmt.Sprintf("通道「%s」（#%d）已被禁用", channelName, channelId)
			content := fmt.Sprintf("通道「%s」（#%d）已被禁用，原因：%s", channelName, channelId, reason)
			notify.Send(subject, content)
		}

		return nil, nil
	})

	// 处理错误
	if err != nil {
		logger.SysError(fmt.Sprintf("DisableChannel failed for channel %d: %v", channelId, err))
	}
}

// enable & notify
func EnableChannel(channelId int, channelName string, sendNotify bool) {
	model.UpdateChannelStatusById(channelId, config.ChannelStatusEnabled)
	if !sendNotify {
		return
	}

	subject := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
	content := fmt.Sprintf("通道「%s」（#%d）已被启用", channelName, channelId)
	notify.Send(subject, content)
}

func RelayNotFound(c *gin.Context) {
	err := types.OpenAIError{
		Message: fmt.Sprintf("Invalid URL (%s %s)", c.Request.Method, c.Request.URL.Path),
		Type:    "invalid_request_error",
		Param:   "",
		Code:    "",
	}
	c.JSON(http.StatusNotFound, gin.H{
		"error": err,
	})
}
