package model

import (
	"done-hub/common/utils"
	"errors"

	"gorm.io/gorm"
)

type InviteCode struct {
	Id          int    `json:"id"`
	Code        string `json:"code" gorm:"type:varchar(32);uniqueIndex;not null"`
	MaxUses     int    `json:"max_uses" gorm:"default:1;not null"`       // 可使用次数
	UsedCount   int    `json:"used_count" gorm:"default:0;not null"`     // 已使用次数
	Status      int    `json:"status" gorm:"default:1;not null"`         // 状态：1启用，2禁用
	ExpiresAt   int64  `json:"expires_at" gorm:"bigint"`                 // 过期时间戳，0表示永不过期
	CreatedTime int64  `json:"created_time" gorm:"bigint;not null"`      // 创建时间
	UpdatedTime int64  `json:"updated_time" gorm:"bigint;not null"`      // 更新时间
	CreatedBy   int    `json:"created_by" gorm:"not null"`               // 创建者ID
	Name        string `json:"name" gorm:"type:varchar(100);default:''"` // 邀请码名称/备注
}

const (
	InviteCodeStatusEnabled  = 1
	InviteCodeStatusDisabled = 2
)

var allowedInviteCodeOrderFields = map[string]bool{
	"id":           true,
	"code":         true,
	"max_uses":     true,
	"used_count":   true,
	"status":       true,
	"expires_at":   true,
	"created_time": true,
	"updated_time": true,
	"name":         true,
}

func GetInviteCodesList(params *GenericParams) (*DataResult[InviteCode], error) {
	var inviteCodes []*InviteCode
	db := DB
	if params.Keyword != "" {
		db = db.Where("code LIKE ? OR name LIKE ?", "%"+params.Keyword+"%", "%"+params.Keyword+"%")
	}

	return PaginateAndOrder[InviteCode](db, &params.PaginationParams, &inviteCodes, allowedInviteCodeOrderFields)
}

func GetInviteCodeById(id int) (*InviteCode, error) {
	if id == 0 {
		return nil, errors.New("id 为空！")
	}
	inviteCode := InviteCode{Id: id}
	err := DB.First(&inviteCode, "id = ?", id).Error
	return &inviteCode, err
}

func GetInviteCodeByCode(code string) (*InviteCode, error) {
	if code == "" {
		return nil, errors.New("邀请码为空！")
	}
	inviteCode := InviteCode{}
	err := DB.First(&inviteCode, "code = ?", code).Error
	return &inviteCode, err
}

// ValidateInviteCode 验证邀请码是否有效并使用
func ValidateInviteCode(code string) error {
	if code == "" {
		return errors.New("邀请码不能为空")
	}

	var inviteCode InviteCode
	err := DB.Transaction(func(tx *gorm.DB) error {
		// 加锁查询邀请码
		err := tx.Set("gorm:query_option", "FOR UPDATE").Where("code = ?", code).First(&inviteCode).Error
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("邀请码不存在")
			}
			return err
		}

		// 检查邀请码状态
		if inviteCode.Status != InviteCodeStatusEnabled {
			return errors.New("邀请码已被禁用")
		}

		// 检查是否过期
		if inviteCode.ExpiresAt > 0 && inviteCode.ExpiresAt < utils.GetTimestamp() {
			return errors.New("邀请码已过期")
		}

		// 检查使用次数
		if inviteCode.UsedCount >= inviteCode.MaxUses {
			return errors.New("邀请码使用次数已达上限")
		}

		// 增加使用次数
		inviteCode.UsedCount++
		inviteCode.UpdatedTime = utils.GetTimestamp()

		// 如果使用次数达到上限，自动禁用
		if inviteCode.UsedCount >= inviteCode.MaxUses {
			inviteCode.Status = InviteCodeStatusDisabled
		}

		return tx.Save(&inviteCode).Error
	})

	return err
}

func (inviteCode *InviteCode) Insert() error {
	inviteCode.CreatedTime = utils.GetTimestamp()
	inviteCode.UpdatedTime = utils.GetTimestamp()
	return DB.Create(inviteCode).Error
}

func (inviteCode *InviteCode) Update() error {
	inviteCode.UpdatedTime = utils.GetTimestamp()
	return DB.Model(inviteCode).Select("name", "max_uses", "status", "expires_at", "updated_time").Updates(inviteCode).Error
}

func (inviteCode *InviteCode) Delete() error {
	return DB.Delete(inviteCode).Error
}

func DeleteInviteCodeById(id int) error {
	if id == 0 {
		return errors.New("id 为空！")
	}
	inviteCode := InviteCode{Id: id}
	err := DB.Where(inviteCode).First(&inviteCode).Error
	if err != nil {
		return err
	}
	return inviteCode.Delete()
}

// GenerateInviteCode 生成唯一的邀请码
func GenerateInviteCode() string {
	for {
		code := utils.GetRandomString(8) // 生成8位随机字符串
		var count int64
		DB.Model(&InviteCode{}).Where("code = ?", code).Count(&count)
		if count == 0 {
			return code
		}
	}
}

type InviteCodeStatistics struct {
	TotalCount    int64 `json:"total_count"`
	EnabledCount  int64 `json:"enabled_count"`
	DisabledCount int64 `json:"disabled_count"`
	TotalUses     int64 `json:"total_uses"`
}

func GetInviteCodeStatistics() (*InviteCodeStatistics, error) {
	var stats InviteCodeStatistics

	// 总数
	err := DB.Model(&InviteCode{}).Count(&stats.TotalCount).Error
	if err != nil {
		return nil, err
	}

	// 启用数量
	err = DB.Model(&InviteCode{}).Where("status = ?", InviteCodeStatusEnabled).Count(&stats.EnabledCount).Error
	if err != nil {
		return nil, err
	}

	// 禁用数量
	err = DB.Model(&InviteCode{}).Where("status = ?", InviteCodeStatusDisabled).Count(&stats.DisabledCount).Error
	if err != nil {
		return nil, err
	}

	// 总使用次数
	err = DB.Model(&InviteCode{}).Select("COALESCE(SUM(used_count), 0)").Scan(&stats.TotalUses).Error
	if err != nil {
		return nil, err
	}

	return &stats, nil
}
