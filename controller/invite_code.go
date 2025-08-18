package controller

import (
	"done-hub/common/utils"
	"done-hub/model"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func GetInviteCodesList(c *gin.Context) {
	params := model.GenericParams{}
	if err := c.ShouldBindQuery(&params); err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	inviteCodes, err := model.GetInviteCodesList(&params)
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
		"data":    inviteCodes,
	})
}

func GetInviteCode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}
	inviteCode, err := model.GetInviteCodeById(id)
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
		"data":    inviteCode,
	})
}

type CreateInviteCodeRequest struct {
	Name      string `json:"name"`
	MaxUses   int    `json:"max_uses"`
	ExpiresAt int64  `json:"expires_at"`
	Count     int    `json:"count"` // 批量创建数量
}

func CreateInviteCode(c *gin.Context) {
	var req CreateInviteCodeRequest
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	// 参数验证
	if req.MaxUses <= 0 {
		req.MaxUses = 1
	}
	if req.Count <= 0 {
		req.Count = 1
	}
	if req.Count > 100 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "批量创建数量不能超过100个",
		})
		return
	}

	userId := c.GetInt("id")
	var createdCodes []string

	for i := 0; i < req.Count; i++ {
		inviteCode := model.InviteCode{
			Code:      model.GenerateInviteCode(),
			Name:      req.Name,
			MaxUses:   req.MaxUses,
			ExpiresAt: req.ExpiresAt,
			Status:    model.InviteCodeStatusEnabled,
			CreatedBy: userId,
		}

		if req.Count > 1 {
			inviteCode.Name = req.Name + "_" + strconv.Itoa(i+1)
		}

		err = inviteCode.Insert()
		if err != nil {
			c.JSON(http.StatusOK, gin.H{
				"success": false,
				"message": err.Error(),
			})
			return
		}
		createdCodes = append(createdCodes, inviteCode.Code)
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
		"data": gin.H{
			"codes": createdCodes,
			"count": len(createdCodes),
		},
	})
}

type UpdateInviteCodeRequest struct {
	Name      string `json:"name"`
	MaxUses   int    `json:"max_uses"`
	Status    int    `json:"status"`
	ExpiresAt int64  `json:"expires_at"`
}

func UpdateInviteCode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	var req UpdateInviteCodeRequest
	err = json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	inviteCode, err := model.GetInviteCodeById(id)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	// 参数验证
	if req.MaxUses <= 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "最大使用次数必须大于0",
		})
		return
	}

	if req.MaxUses < inviteCode.UsedCount {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "最大使用次数不能小于已使用次数",
		})
		return
	}

	if req.Status != model.InviteCodeStatusEnabled && req.Status != model.InviteCodeStatusDisabled {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的状态值",
		})
		return
	}

	inviteCode.Name = req.Name
	inviteCode.MaxUses = req.MaxUses
	inviteCode.Status = req.Status
	inviteCode.ExpiresAt = req.ExpiresAt

	err = inviteCode.Update()
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

func DeleteInviteCode(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": err.Error(),
		})
		return
	}

	err = model.DeleteInviteCodeById(id)
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

func GetInviteCodeStatistics(c *gin.Context) {
	stats, err := model.GetInviteCodeStatistics()
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
		"data":    stats,
	})
}

// BatchDeleteInviteCodes 批量删除邀请码
func BatchDeleteInviteCodes(c *gin.Context) {
	var req struct {
		Ids []int `json:"ids"`
	}
	err := json.NewDecoder(c.Request.Body).Decode(&req)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "无效的参数",
		})
		return
	}

	if len(req.Ids) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "请选择要删除的邀请码",
		})
		return
	}

	var failedIds []int
	for _, id := range req.Ids {
		err = model.DeleteInviteCodeById(id)
		if err != nil {
			failedIds = append(failedIds, id)
		}
	}

	if len(failedIds) > 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"message": "部分邀请码删除失败，ID: " + strings.Join(utils.IntSliceToStringSlice(failedIds), ", "),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "",
	})
}
