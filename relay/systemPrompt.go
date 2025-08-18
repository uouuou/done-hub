package relay

import (
	"done-hub/types"
)

// systemPrompt 系统提示词工具
func systemPrompt(systemPrompt string, request *types.ChatCompletionRequest) {
	// 添加系统提示词
	newMsg := []types.ChatCompletionMessage{
		{
			Role:    "system",
			Content: systemPrompt,
		},
	}
	request.Messages = append(newMsg, request.Messages...)
}
