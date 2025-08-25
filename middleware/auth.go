package middleware

import (
	"done-hub/common/config"
	"done-hub/common/utils"
	"done-hub/model"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

func authHelper(c *gin.Context, minRole int) {
	session := sessions.Default(c)
	username := session.Get("username")
	role := session.Get("role")
	id := session.Get("id")
	status := session.Get("status")
	if username == nil {
		// Check access token
		accessToken := c.Request.Header.Get("Authorization")
		if accessToken == "" {
			token := c.Param("accessToken")
			if token == "" {
				c.JSON(http.StatusUnauthorized, gin.H{
					"success": false,
					"message": "无权进行此操作，未登录且未提供 access token",
				})
				c.Abort()
				return
			}
			accessToken = fmt.Sprintf("Bearer %s", token)
		}
		user := model.ValidateAccessToken(accessToken)
		if user != nil && user.Username != "" {
			// Token is valid
			username = user.Username
			role = user.Role
			id = user.Id
			status = user.Status
		} else {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": "无权进行此操作，access token 无效",
			})
			c.Abort()
			return
		}
	}
	if status.(int) == config.UserStatusDisabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "用户已被封禁",
		})
		c.Abort()
		return
	}
	if role.(int) < minRole {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无权进行此操作，权限不足",
		})
		c.Abort()
		return
	}
	c.Set("username", username)
	c.Set("role", role)
	c.Set("id", id)
	c.Next()
}

func TrySetUserBySession() func(c *gin.Context) {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		id := session.Get("id")
		if id == nil {
			c.Next()
			return
		}

		idInt, ok := id.(int)
		if !ok {
			c.Next()
			return
		}

		c.Set("id", idInt)
		userGroup, err := model.CacheGetUserGroup(idInt)
		if err == nil {
			c.Set("group", userGroup)
		}
		c.Next()
	}
}

func UserAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, config.RoleCommonUser)
	}
}

func AdminAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, config.RoleAdminUser)
	}
}

func RootAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		authHelper(c, config.RoleRootUser)
	}
}

func tokenAuth(c *gin.Context, key string) {
	key = strings.TrimPrefix(key, "Bearer ")
	key = strings.TrimPrefix(key, "sk-")

	if len(key) < 48 {
		abortWithMessage(c, http.StatusUnauthorized, "无效的令牌")
		return
	}

	parts := strings.Split(key, "#")
	key = parts[0]
	token, err := model.ValidateUserToken(key)
	if err != nil {
		abortWithMessage(c, http.StatusUnauthorized, err.Error())
		return
	}

	c.Set("id", token.UserId)
	c.Set("token_id", token.Id)
	c.Set("token_name", token.Name)
	c.Set("token_group", token.Group)
	// 直接将设置数据序列化为JSON字符串传递
	settingData := token.Setting.Data()
	settingJSON, _ := json.Marshal(settingData)
	c.Set("token_setting", string(settingJSON))

	// 验证subnet限制
	setting := token.Setting.Data()
	if setting.Subnet != "" {
		clientIP := c.ClientIP()
		if !isIPInSubnet(clientIP, setting.Subnet) {
			abortWithMessage(c, http.StatusForbidden, "访问IP不在允许的子网范围内")
			return
		}
	}
	if len(parts) > 1 {
		if model.IsAdmin(token.UserId) {
			if strings.HasPrefix(parts[1], "!") {
				channelId := utils.String2Int(parts[1][1:])
				c.Set("skip_channel_ids", []int{channelId})
			} else {
				channelId := utils.String2Int(parts[1])
				if channelId == 0 {
					abortWithMessage(c, http.StatusForbidden, "无效的渠道 Id")
					return
				}
				c.Set("specific_channel_id", channelId)
				if len(parts) == 3 && parts[2] == "ignore" {
					c.Set("specific_channel_id_ignore", true)
				}
			}
		} else {
			abortWithMessage(c, http.StatusForbidden, "普通用户不支持指定渠道")
			return
		}
	}
	c.Next()
}

func OpenaiAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		isWebSocket := c.GetHeader("Upgrade") == "websocket"
		key := c.Request.Header.Get("Authorization")

		if isWebSocket && key == "" {
			protocols := c.Request.Header["Sec-Websocket-Protocol"]
			if len(protocols) > 0 {
				protocolList := strings.Split(protocols[0], ",")
				for _, protocol := range protocolList {
					protocol = strings.TrimSpace(protocol)
					if strings.HasPrefix(protocol, "openai-insecure-api-key.") {
						key = strings.TrimPrefix(protocol, "openai-insecure-api-key.")
						break
					}
				}
			}
		}
		tokenAuth(c, key)
	}
}

func ClaudeAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		key := c.Request.Header.Get("x-api-key")
		if key == "" {
			key = c.Request.Header.Get("Authorization")
		}
		tokenAuth(c, key)
	}
}

func GeminiAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		key := c.Request.Header.Get("x-goog-api-key")
		if key == "" {
			// 查询GET参数
			key = c.Query("key")

			if key == "" {
				key = c.Request.Header.Get("Authorization")
			}
		}
		tokenAuth(c, key)
	}
}

func MjAuth() func(c *gin.Context) {
	return func(c *gin.Context) {
		// 判断path :mode
		model := c.Param("mode")

		if model != "" && model != "mj-fast" && model != "mj-turbo" && model != "mj-relax" {
			midjourneyAbortWithMessage(c, 4, "无效的加速模式")
			return
		}

		if model == "" {
			model = "mj-fast"
		}

		model = strings.TrimPrefix(model, "mj-")
		c.Set("mj_model", model)

		key := c.Request.Header.Get("mj-api-secret")
		tokenAuth(c, key)
	}
}

func SpecifiedChannel() func(c *gin.Context) {
	return func(c *gin.Context) {
		channelId := c.GetInt("specific_channel_id")
		c.Set("specific_channel_id_ignore", false)

		if channelId <= 0 {
			abortWithMessage(c, http.StatusForbidden, "必须指定渠道")
			return
		}
		c.Next()
	}
}

// isIPInSubnet 检查IP地址是否在指定的子网内
func isIPInSubnet(ip, subnet string) bool {
	// 如果是单个IP地址，直接比较
	if !strings.Contains(subnet, "/") {
		return ip == subnet
	}

	// 如果是CIDR格式，进行子网匹配
	ipParts := strings.Split(subnet, "/")
	if len(ipParts) != 2 {
		return false
	}

	networkIP := ipParts[0]
	maskLength, err := strconv.Atoi(ipParts[1])
	if err != nil || maskLength < 0 || maskLength > 32 {
		return false
	}

	// 简单的CIDR匹配逻辑
	// 将IP地址转换为32位整数进行比较
	ipInt := ipToInt(ip)
	networkInt := ipToInt(networkIP)
	mask := uint32((0xFFFFFFFF << (32 - maskLength)) & 0xFFFFFFFF)

	return (ipInt & mask) == (networkInt & mask)
}

// ipToInt 将IPv4地址转换为32位整数
func ipToInt(ip string) uint32 {
	parts := strings.Split(ip, ".")
	if len(parts) != 4 {
		return 0
	}

	var result uint32
	for i, part := range parts {
		num, err := strconv.Atoi(part)
		if err != nil || num < 0 || num > 255 {
			return 0
		}
		result |= uint32(num) << (24 - i*8)
	}
	return result
}
