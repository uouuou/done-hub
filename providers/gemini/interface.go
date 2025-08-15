package gemini

import (
	"done-hub/common/requester"
	"done-hub/providers/base"
	"done-hub/types"
)

type GeminiChatInterface interface {
	base.ProviderInterface
	CreateGeminiChat(request *GeminiChatRequest) (*GeminiChatResponse, *types.OpenAIErrorWithStatusCode)
	CreateGeminiChatStream(request *GeminiChatRequest) (requester.StreamReaderInterface[string], *types.OpenAIErrorWithStatusCode)
}

type GeminiVeoInterface interface {
	base.ProviderInterface
	CreateVeoVideoAndDownload(request *VeoVideoRequest, modelName string) ([]byte, string, *types.OpenAIErrorWithStatusCode)
}
