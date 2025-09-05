package middleware

import (
	"done-hub/model"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Distribute() func(c *gin.Context) {
	return func(c *gin.Context) {
		userId := c.GetInt("id")
		userGroup, _ := model.CacheGetUserGroup(userId)
		c.Set("group", userGroup)

		tokenGroup := c.GetString("token_group")
		if tokenGroup == "" {
			tokenGroup = userGroup
			c.Set("token_group", tokenGroup)
		}

		groupRatio := model.GlobalUserGroupRatio.GetBySymbol(tokenGroup)
		if groupRatio == nil {
			abortWithMessage(c, http.StatusForbidden, fmt.Sprintf("分组 %s 不存在", tokenGroup))
			return
		}

		// 验证用户是否有权使用该分组
		if tokenGroup != userGroup {
			if !groupRatio.Public {
				abortWithMessage(c, http.StatusForbidden, fmt.Sprintf("无权使用分组 %s", tokenGroup))
				return
			}
		}

		c.Set("group_ratio", groupRatio.Ratio)
		c.Next()
	}
}
