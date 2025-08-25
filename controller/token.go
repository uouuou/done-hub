package controller

import (
	"done-hub/common"
	"done-hub/common/config"
	"done-hub/common/utils"
	"done-hub/model"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetUserTokensList(c *gin.Context) {
	userId := c.GetInt("id")
	var params model.GenericParams
	if err := c.ShouldBindQuery(&params); err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}

	tokens, err := model.GetUserTokensList(userId, &params)
	if err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    tokens,
	})
}

func GetToken(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	token, err := model.GetTokenByIds(id, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    token,
	})
}

func GetPlaygroundToken(c *gin.Context) {
	tokenName := "sys_playground"
	userId := c.GetInt("id")
	token, err := model.GetTokenByName(tokenName, userId)
	if err != nil {
		cleanToken := model.Token{
			UserId: userId,
			Name:   tokenName,
			// Key:            utils.GenerateKey(),
			CreatedTime:    utils.GetTimestamp(),
			AccessedTime:   utils.GetTimestamp(),
			ExpiredTime:    0,
			RemainQuota:    0,
			UnlimitedQuota: true,
		}
		err = cleanToken.Insert()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "创建令牌失败，请去系统手动配置一个名称为：sys_playground 的令牌",
			})
			return
		}
		token = &cleanToken
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    token.Key,
	})
}

func AddToken(c *gin.Context) {
	userId := c.GetInt("id")
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if len(token.Name) > 30 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "令牌名称过长",
		})
		return
	}

	if token.Group != "" {
		err = validateTokenGroup(token.Group, userId)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}

	setting := token.Setting.Data()
	err = validateTokenSetting(&setting)
	if err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}

	// 验证models字段
	if len(setting.Models) > 0 {
		// 获取用户组可用的所有models
		userGroup, err := model.CacheGetUserGroup(userId)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "获取用户组信息失败",
			})
			return
		}

		availableModels, err := model.ChannelGroup.GetGroupModels(userGroup)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "获取可用模型失败",
			})
			return
		}

		// 验证用户选择的models是否都在可用列表中
		for _, model := range setting.Models {
			if !contains(availableModels, model) {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": fmt.Sprintf("模型 %s 不在可用模型列表中", model),
				})
				return
			}
		}
	}

	// 验证subnet字段
	if setting.Subnet != "" {
		if !isValidSubnet(setting.Subnet) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无效的子网格式",
			})
			return
		}
	}

	cleanToken := model.Token{
		UserId: userId,
		Name:   token.Name,
		// Key:            utils.GenerateKey(),
		CreatedTime:    utils.GetTimestamp(),
		AccessedTime:   utils.GetTimestamp(),
		ExpiredTime:    token.ExpiredTime,
		RemainQuota:    token.RemainQuota,
		UnlimitedQuota: token.UnlimitedQuota,
		Group:          token.Group,
		Setting:        token.Setting,
	}
	err = cleanToken.Insert()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func DeleteToken(c *gin.Context) {
	id, _ := strconv.Atoi(c.Param("id"))
	userId := c.GetInt("id")
	err := model.DeleteTokenById(id, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}

func UpdateToken(c *gin.Context) {
	userId := c.GetInt("id")
	statusOnly := c.Query("status_only")
	token := model.Token{}
	err := c.ShouldBindJSON(&token)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if len(token.Name) > 30 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "令牌名称过长",
		})
		return
	}

	setting := token.Setting.Data()
	err = validateTokenSetting(&setting)
	if err != nil {
		common.APIRespondWithError(c, http.StatusOK, err)
		return
	}

	cleanToken, err := model.GetTokenByIds(token.Id, userId)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	if token.Status == config.TokenStatusEnabled {
		if cleanToken.Status == config.TokenStatusExpired && cleanToken.ExpiredTime <= utils.GetTimestamp() && cleanToken.ExpiredTime != -1 {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "令牌已过期，无法启用，请先修改令牌过期时间，或者设置为永不过期",
			})
			return
		}
		if cleanToken.Status == config.TokenStatusExhausted && cleanToken.RemainQuota <= 0 && !cleanToken.UnlimitedQuota {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "令牌可用额度已用尽，无法启用，请先修改令牌剩余额度，或者设置为无限额度",
			})
			return
		}
	}

	if cleanToken.Group != token.Group && token.Group != "" {
		err = validateTokenGroup(token.Group, userId)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
	}

	// 验证models和subnet字段
	if len(setting.Models) > 0 {
		// 获取用户组可用的所有models
		userGroup, err := model.CacheGetUserGroup(userId)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "获取用户组信息失败",
			})
			return
		}

		availableModels, err := model.ChannelGroup.GetGroupModels(userGroup)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "获取可用模型失败",
			})
			return
		}

		// 验证用户选择的models是否都在可用列表中
		for _, model := range setting.Models {
			if !contains(availableModels, model) {
				c.JSON(http.StatusOK, gin.H{
					"success": false,
					"message": fmt.Sprintf("模型 %s 不在可用模型列表中", model),
				})
				return
			}
		}
	}

	// 验证subnet字段
	if setting.Subnet != "" {
		if !isValidSubnet(setting.Subnet) {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无效的子网格式",
			})
			return
		}
	}

	if statusOnly != "" {
		cleanToken.Status = token.Status
	} else {
		// If you add more fields, please also update token.Update()
		cleanToken.Name = token.Name
		cleanToken.ExpiredTime = token.ExpiredTime
		cleanToken.RemainQuota = token.RemainQuota
		cleanToken.UnlimitedQuota = token.UnlimitedQuota
		cleanToken.Group = token.Group
		cleanToken.Setting = token.Setting
	}
	err = cleanToken.Update()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data":    cleanToken,
	})
}

func validateTokenGroup(tokenGroup string, userId int) error {
	userGroup, _ := model.CacheGetUserGroup(userId)
	if userGroup == "" {
		return errors.New("获取用户组信息失败")
	}

	groupRatio := model.GlobalUserGroupRatio.GetBySymbol(tokenGroup)
	if groupRatio == nil {
		return errors.New("无效的用户组")
	}

	if !groupRatio.Public && userGroup != tokenGroup {
		return errors.New("当前用户组无权使用指定的分组")
	}

	return nil
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

// isValidSubnet 验证子网格式
func isValidSubnet(subnet string) bool {
	// 简单的IP地址验证，支持单个IP或CIDR格式
	if len(subnet) == 0 {
		return false
	}

	// 验证IPv4地址格式
	ipParts := strings.Split(subnet, "/")
	if len(ipParts) > 2 {
		return false
	}

	// 验证IP地址部分
	ip := ipParts[0]
	ipSegments := strings.Split(ip, ".")
	if len(ipSegments) != 4 {
		return false
	}

	for _, segment := range ipSegments {
		if len(segment) == 0 || len(segment) > 3 {
			return false
		}
		// 检查是否都是数字
		for _, char := range segment {
			if char < '0' || char > '9' {
				return false
			}
		}
		// 转换为数字验证范围
		num, err := strconv.Atoi(segment)
		if err != nil || num < 0 || num > 255 {
			return false
		}
	}

	// 如果有子网掩码部分，验证其范围
	if len(ipParts) == 2 {
		mask, err := strconv.Atoi(ipParts[1])
		if err != nil || mask < 0 || mask > 32 {
			return false
		}
	}

	return true
}

func validateTokenSetting(setting *model.TokenSetting) error {
	if setting == nil {
		return nil
	}

	if setting.Heartbeat.Enabled {
		if setting.Heartbeat.TimeoutSeconds < 30 || setting.Heartbeat.TimeoutSeconds > 90 {
			return errors.New("heartbeat timeout seconds must be between 30 and 90")
		}
	}

	// 验证models字段
	if len(setting.Models) > 0 {
		for _, model := range setting.Models {
			if model == "" {
				return errors.New("模型名称不能为空")
			}
		}
		// 去重
		uniqueModels := make(map[string]bool)
		for _, model := range setting.Models {
			if uniqueModels[model] {
				return errors.New("模型列表中包含重复的模型")
			}
			uniqueModels[model] = true
		}
	}

	// 验证subnet字段
	if setting.Subnet != "" {
		if !isValidSubnet(setting.Subnet) {
			return errors.New("无效的子网格式")
		}
	}

	return nil
}
