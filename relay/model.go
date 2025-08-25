package relay

import (
	"done-hub/common/config"
	"done-hub/common/utils"
	"done-hub/model"
	"done-hub/providers/claude"
	"done-hub/providers/gemini"
	"done-hub/types"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/gin-gonic/gin"
)

// https://platform.openai.com/docs/api-reference/models/list
type OpenAIModels struct {
	Id      string  `json:"id"`
	Object  string  `json:"object"`
	Created int     `json:"created"`
	OwnedBy *string `json:"owned_by"`
}

func ListModelsByToken(c *gin.Context) {
	tokenId := c.GetInt("token_id")

	// 如果没有tokenId（匿名访问），返回所有公共模型
	if tokenId == 0 {
		publicModels := model.ChannelGroup.GetModelsGroups()
		var allModels []string
		for modelName := range publicModels {
			allModels = append(allModels, modelName)
		}
		sort.Strings(allModels)

		var groupOpenAIModels []*OpenAIModels
		for _, modelName := range allModels {
			groupOpenAIModels = append(groupOpenAIModels, getOpenAIModelWithName(modelName))
		}

		// 根据 OwnedBy 排序
		sort.Slice(groupOpenAIModels, func(i, j int) bool {
			if groupOpenAIModels[i].OwnedBy == nil {
				return true
			}
			if groupOpenAIModels[j].OwnedBy == nil {
				return false
			}
			return *groupOpenAIModels[i].OwnedBy < *groupOpenAIModels[j].OwnedBy
		})

		c.JSON(200, gin.H{
			"object": "list",
			"data":   groupOpenAIModels,
		})
		return
	}

	// 有token，获取用户组并应用模型限制
	groupName := c.GetString("token_group")
	if groupName == "" {
		groupName = c.GetString("group")
	}

	models, err := model.ChannelGroup.GetGroupModels(groupName)
	if err != nil {
		c.JSON(200, gin.H{
			"object": "list",
			"data":   []string{},
		})
		return
	}

	// 获取token设置，过滤models
	tokenSetting := c.GetString("token_setting")
	var filteredModels []string
	if tokenSetting != "" {
		var setting model.TokenSetting
		if err := json.Unmarshal([]byte(tokenSetting), &setting); err == nil {
			// 检查是否设置了models字段（即使是空数组）
			if setting.Models != nil {
				// 如果token设置了特定models，只返回这些models
				for _, model := range models {
					if contains(setting.Models, model) {
						filteredModels = append(filteredModels, model)
					}
				}
			} else {
				// 未设置models字段，返回所有可用模型
				filteredModels = models
			}
		} else {
			// 解析失败，返回所有可用模型
			filteredModels = models
		}
	} else {
		// 没有token设置，返回所有可用模型
		filteredModels = models
	}
	sort.Strings(filteredModels)

	var groupOpenAIModels []*OpenAIModels
	for _, modelName := range filteredModels {
		groupOpenAIModels = append(groupOpenAIModels, getOpenAIModelWithName(modelName))
	}

	// 根据 OwnedBy 排序
	sort.Slice(groupOpenAIModels, func(i, j int) bool {
		if groupOpenAIModels[i].OwnedBy == nil {
			return true
		}
		if groupOpenAIModels[j].OwnedBy == nil {
			return false
		}
		return *groupOpenAIModels[i].OwnedBy < *groupOpenAIModels[j].OwnedBy
	})

	c.JSON(200, gin.H{
		"object": "list",
		"data":   groupOpenAIModels,
	})
}

// https://generativelanguage.googleapis.com/v1beta/models?key=xxxxxxx
func ListGeminiModelsByToken(c *gin.Context) {
	tokenId := c.GetInt("token_id")

	// 如果没有tokenId（匿名访问），返回所有公共模型
	if tokenId == 0 {
		publicModels := model.ChannelGroup.GetModelsGroups()
		var allModels []string
		for modelName := range publicModels {
			allModels = append(allModels, modelName)
		}
		sort.Strings(allModels)

		var geminiModels []gemini.ModelDetails
		for _, modelName := range allModels {
			price := model.PricingInstance.GetPrice(modelName)
			if price.ChannelType == config.ChannelTypeGemini || price.ChannelType == config.ChannelTypeVertexAI {
				geminiModels = append(geminiModels, gemini.ModelDetails{
					Name:        fmt.Sprintf("models/%s", modelName),
					DisplayName: cases.Title(language.Und).String(strings.ReplaceAll(modelName, "-", " ")),
					SupportedGenerationMethods: []string{
						"generateContent",
					},
				})
			}
		}

		c.JSON(200, gemini.ModelListResponse{
			Models: geminiModels,
		})
		return
	}

	// 有token，获取用户组并应用模型限制
	groupName := c.GetString("token_group")
	if groupName == "" {
		groupName = c.GetString("group")
	}

	models, err := model.ChannelGroup.GetGroupModels(groupName)
	if err != nil {
		c.JSON(200, gemini.ModelListResponse{
			Models: []gemini.ModelDetails{},
		})
		return
	}

	// 获取token设置，过滤models
	tokenSetting := c.GetString("token_setting")
	var filteredModels []string
	if tokenSetting != "" {
		var setting model.TokenSetting
		if err := json.Unmarshal([]byte(tokenSetting), &setting); err == nil {
			// 检查是否设置了models字段（即使是空数组）
			if setting.Models != nil {
				// 如果token设置了特定models，只返回这些models
				for _, model := range models {
					if contains(setting.Models, model) {
						filteredModels = append(filteredModels, model)
					}
				}
			} else {
				// 未设置models字段，返回所有可用模型
				filteredModels = models
			}
		} else {
			// 解析失败，返回所有可用模型
			filteredModels = models
		}
	} else {
		// 没有token设置，返回所有可用模型
		filteredModels = models
	}
	sort.Strings(filteredModels)

	var geminiModels []gemini.ModelDetails
	for _, modelName := range filteredModels {
		// Get the price to check if it's a Gemini model (channel_type=25 or 42)
		price := model.PricingInstance.GetPrice(modelName)
		if price.ChannelType == config.ChannelTypeGemini || price.ChannelType == config.ChannelTypeVertexAI {
			geminiModels = append(geminiModels, gemini.ModelDetails{
				Name:        fmt.Sprintf("models/%s", modelName),
				DisplayName: cases.Title(language.Und).String(strings.ReplaceAll(modelName, "-", " ")),
				SupportedGenerationMethods: []string{
					"generateContent",
				},
			})
		}
	}

	c.JSON(200, gemini.ModelListResponse{
		Models: geminiModels,
	})
}

func ListClaudeModelsByToken(c *gin.Context) {
	tokenId := c.GetInt("token_id")

	// 如果没有tokenId（匿名访问），返回所有公共模型
	if tokenId == 0 {
		publicModels := model.ChannelGroup.GetModelsGroups()
		var allModels []string
		for modelName := range publicModels {
			allModels = append(allModels, modelName)
		}
		sort.Strings(allModels)

		var claudeModelsData []claude.Model
		for _, modelName := range allModels {
			price := model.PricingInstance.GetPrice(modelName)
			if price.ChannelType == config.ChannelTypeAnthropic {
				claudeModelsData = append(claudeModelsData, claude.Model{
					ID:   modelName,
					Type: "model",
				})
			}
		}

		c.JSON(200, claude.ModelListResponse{
			Data: claudeModelsData,
		})
		return
	}

	// 有token，获取用户组并应用模型限制
	groupName := c.GetString("token_group")
	if groupName == "" {
		groupName = c.GetString("group")
	}

	models, err := model.ChannelGroup.GetGroupModels(groupName)
	if err != nil {
		c.JSON(200, claude.ModelListResponse{
			Data: []claude.Model{},
		})
		return
	}

	// 获取token设置，过滤models
	tokenSetting := c.GetString("token_setting")
	var filteredModels []string
	if tokenSetting != "" {
		var setting model.TokenSetting
		if err := json.Unmarshal([]byte(tokenSetting), &setting); err == nil {
			// 检查是否设置了models字段（即使是空数组）
			if setting.Models != nil {
				// 如果token设置了特定models，只返回这些models
				for _, model := range models {
					if contains(setting.Models, model) {
						filteredModels = append(filteredModels, model)
					}
				}
			} else {
				// 未设置models字段，返回所有可用模型
				filteredModels = models
			}
		} else {
			// 解析失败，返回所有可用模型
			filteredModels = models
		}
	} else {
		// 没有token设置，返回所有可用模型
		filteredModels = models
	}
	sort.Strings(filteredModels)

	var claudeModelsData []claude.Model
	for _, modelName := range filteredModels {
		// Get the price to check if it's a Claude model (channel_type=25)
		price := model.PricingInstance.GetPrice(modelName)
		if price.ChannelType == config.ChannelTypeAnthropic {
			claudeModelsData = append(claudeModelsData, claude.Model{
				ID:   modelName,
				Type: "model",
			})
		}
	}

	c.JSON(200, claude.ModelListResponse{
		Data: claudeModelsData,
	})
}

func ListModelsForAdmin(c *gin.Context) {
	prices := model.PricingInstance.GetAllPrices()
	var openAIModels []OpenAIModels
	for modelId, price := range prices {
		openAIModels = append(openAIModels, OpenAIModels{
			Id:      modelId,
			Object:  "model",
			Created: 1677649963,
			OwnedBy: getModelOwnedBy(price.ChannelType),
		})
	}
	// 根据 OwnedBy 排序
	sort.Slice(openAIModels, func(i, j int) bool {
		if openAIModels[i].OwnedBy == nil {
			return true // 假设 nil 值小于任何非 nil 值
		}
		if openAIModels[j].OwnedBy == nil {
			return false // 假设任何非 nil 值大于 nil 值
		}
		return *openAIModels[i].OwnedBy < *openAIModels[j].OwnedBy
	})

	c.JSON(200, gin.H{
		"object": "list",
		"data":   openAIModels,
	})
}

func RetrieveModel(c *gin.Context) {
	modelName := c.Param("model")
	openaiModel := getOpenAIModelWithName(modelName)
	if *openaiModel.OwnedBy != model.UnknownOwnedBy {
		c.JSON(200, openaiModel)
	} else {
		openAIError := types.OpenAIError{
			Message: fmt.Sprintf("The model '%s' does not exist", modelName),
			Type:    "invalid_request_error",
			Param:   "model",
			Code:    "model_not_found",
		}
		c.JSON(200, gin.H{
			"error": openAIError,
		})
	}
}

func getModelOwnedBy(channelType int) (ownedBy *string) {
	ownedByName := model.ModelOwnedBysInstance.GetName(channelType)
	if ownedByName != "" {
		return &ownedByName
	}

	return &model.UnknownOwnedBy
}

func getOpenAIModelWithName(modelName string) *OpenAIModels {
	price := model.PricingInstance.GetPrice(modelName)

	return &OpenAIModels{
		Id:      modelName,
		Object:  "model",
		Created: 1677649963,
		OwnedBy: getModelOwnedBy(price.ChannelType),
	}
}

func GetModelOwnedBy(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    model.ModelOwnedBysInstance.GetAll(),
	})
}

type ModelPrice struct {
	Type   string  `json:"type"`
	Input  float64 `json:"input"`
	Output float64 `json:"output"`
}

type AvailableModelResponse struct {
	Groups  []string     `json:"groups"`
	OwnedBy string       `json:"owned_by"`
	Price   *model.Price `json:"price"`
}

func AvailableModel(c *gin.Context) {
	groupName := c.GetString("group")

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    getAvailableModels(groupName),
	})
}

func GetAvailableModels(groupName string) map[string]*AvailableModelResponse {
	return getAvailableModels(groupName)
}

func getAvailableModels(groupName string) map[string]*AvailableModelResponse {
	publicModels := model.ChannelGroup.GetModelsGroups()
	publicGroups := model.GlobalUserGroupRatio.GetPublicGroupList()
	if groupName != "" && !utils.Contains(groupName, publicGroups) {
		publicGroups = append(publicGroups, groupName)
	}

	availableModels := make(map[string]*AvailableModelResponse, len(publicModels))

	for modelName, group := range publicModels {
		groups := []string{}
		for _, publicGroup := range publicGroups {
			if group[publicGroup] {
				groups = append(groups, publicGroup)
			}
		}

		if len(groups) == 0 {
			continue
		}

		if _, ok := availableModels[modelName]; !ok {
			price := model.PricingInstance.GetPrice(modelName)
			availableModels[modelName] = &AvailableModelResponse{
				Groups:  groups,
				OwnedBy: *getModelOwnedBy(price.ChannelType),
				Price:   price,
			}
		}
	}

	return availableModels
}

// contains 检查字符串切片是否包含某个字符串
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
