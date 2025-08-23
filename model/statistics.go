package model

import (
	"done-hub/common"
	"fmt"
	"os"
	"strings"
	"time"
)

type Statistics struct {
	Date             time.Time `gorm:"primary_key;type:date" json:"date"`
	UserId           int       `json:"user_id" gorm:"primary_key"`
	ChannelId        int       `json:"channel_id" gorm:"primary_key"`
	ModelName        string    `json:"model_name" gorm:"primary_key;type:varchar(255)"`
	RequestCount     int       `json:"request_count"`
	Quota            int       `json:"quota"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	RequestTime      int       `json:"request_time"`
}

func GetUserModelStatisticsByPeriod(userId int, startTime, endTime string) (LogStatistic []*LogStatisticGroupModel, err error) {
	dateStr := "date"
	if common.UsingPostgreSQL {
		dateStr = "TO_CHAR(date, 'YYYY-MM-DD') as date"
	} else if common.UsingSQLite {
		dateStr = "strftime('%Y-%m-%d', date) as date"
	}

	err = DB.Raw(`
		SELECT `+dateStr+`,
		model_name, 
		sum(request_count) as request_count,
		sum(quota) as quota,
		sum(prompt_tokens) as prompt_tokens,
		sum(completion_tokens) as completion_tokens,
		sum(request_time) as request_time
		FROM statistics
		WHERE user_id= ?
		AND date BETWEEN ? AND ?
		GROUP BY date, model_name
		ORDER BY date, model_name
	`, userId, startTime, endTime).Scan(&LogStatistic).Error
	return
}

func GetChannelExpensesStatisticsByPeriod(startTime, endTime, groupType string, userID int) (LogStatistics []*LogStatisticGroupChannel, err error) {

	var whereClause strings.Builder
	whereClause.WriteString("WHERE date BETWEEN ? AND ?")
	args := []interface{}{startTime, endTime}

	if userID > 0 {
		whereClause.WriteString(" AND user_id = ?")
		args = append(args, userID)
	}

	dateStr := "date"
	if common.UsingPostgreSQL {
		dateStr = "TO_CHAR(date, 'YYYY-MM-DD') as date"
	} else if common.UsingSQLite {
		dateStr = "strftime('%%Y-%%m-%%d', date) as date"
	} else {
		// MySQL/TiDB - 显式格式化日期以确保兼容性
		dateStr = "DATE_FORMAT(date, '%%Y-%%m-%%d') as date"
	}

	baseSelect := `
        SELECT ` + dateStr + `,
        sum(request_count) as request_count,
        sum(quota) as quota,
        sum(prompt_tokens) as prompt_tokens,
        sum(completion_tokens) as completion_tokens,
        sum(request_time) as request_time,`

	var sql string
	if groupType == "model" {
		sql = baseSelect + `
            model_name as channel
            FROM statistics
            %s
            GROUP BY date, model_name
            ORDER BY date, model_name`
	} else if groupType == "model_type" {
		sql = baseSelect + `
            model_owned_by.name as channel
            FROM statistics
            JOIN prices ON statistics.model_name = prices.model
			JOIN model_owned_by ON prices.channel_type = model_owned_by.id
            %s
            GROUP BY date, model_owned_by.name
            ORDER BY date, model_owned_by.name`

	} else {
		sql = baseSelect + `
            MAX(channels.name) as channel
            FROM statistics
            JOIN channels ON statistics.channel_id = channels.id
            %s
            GROUP BY date, channel_id
            ORDER BY date, channel_id`
	}

	sql = fmt.Sprintf(sql, whereClause.String())
	err = DB.Raw(sql, args...).Scan(&LogStatistics).Error
	if err != nil {
		return nil, err
	}

	return LogStatistics, nil
}

type StatisticsUpdateType int

const (
	StatisticsUpdateTypeToDay     StatisticsUpdateType = 1
	StatisticsUpdateTypeYesterday StatisticsUpdateType = 2
	StatisticsUpdateTypeALL       StatisticsUpdateType = 3
)

func UpdateStatistics(updateType StatisticsUpdateType) error {
	sql := `
	%s statistics (date, user_id, channel_id, model_name, request_count, quota, prompt_tokens, completion_tokens, request_time)
	SELECT 
		%s as date,
		user_id,
		channel_id,
		model_name, 
		count(1) as request_count,
		sum(quota) as quota,
		sum(prompt_tokens) as prompt_tokens,
		sum(completion_tokens) as completion_tokens,
		sum(request_time) as request_time
	FROM logs
	WHERE
		type = 2
		%s
	GROUP BY date, channel_id, user_id, model_name
	ORDER BY date, model_name
	%s
	`

	// 获取系统时区偏移
	getTimezoneOffset := func() (string, string) {
		// 优先使用系统本地时区（Docker中通过TZ环境变量设置）
		location := time.Local

		// 也可以通过环境变量TZ覆盖
		if tzEnv := os.Getenv("TZ"); tzEnv != "" {
			if loc, err := time.LoadLocation(tzEnv); err == nil {
				location = loc
			}
		}

		// 获取当前时间在指定时区的偏移量
		now := time.Now().In(location)
		_, offset := now.Zone()

		// 计算小时偏移
		hours := offset / 3600
		minutes := (offset % 3600) / 60

		// 生成不同数据库需要的格式
		var sqliteOffset, mysqlOffset string
		if hours >= 0 {
			sqliteOffset = fmt.Sprintf("+%d hours", hours)
			if minutes != 0 {
				sqliteOffset += fmt.Sprintf(" %d minutes", minutes)
			}
			mysqlOffset = fmt.Sprintf("+%02d:%02d", hours, minutes)
		} else {
			sqliteOffset = fmt.Sprintf("%d hours", hours) // 负数自带减号
			if minutes != 0 {
				sqliteOffset += fmt.Sprintf(" %d minutes", -minutes) // 分钟也要是负数
			}
			mysqlOffset = fmt.Sprintf("-%02d:%02d", -hours, -minutes)
		}

		return sqliteOffset, mysqlOffset
	}

	sqlPrefix := ""
	sqlWhere := ""
	sqlDate := ""
	sqlSuffix := ""
	if common.UsingSQLite {
		sqlPrefix = "INSERT OR REPLACE INTO"
		// 动态获取时区偏移，而不是硬编码+8 hours
		sqliteOffset, _ := getTimezoneOffset()
		sqlDate = fmt.Sprintf("strftime('%%Y-%%m-%%d', datetime(created_at, 'unixepoch', '%s'))", sqliteOffset)
		sqlSuffix = ""
	} else if common.UsingPostgreSQL {
		sqlPrefix = "INSERT INTO"
		// PostgreSQL使用系统时区
		tzName := "UTC"
		if tzEnv := os.Getenv("TZ"); tzEnv != "" {
			tzName = tzEnv
		}
		sqlDate = fmt.Sprintf("DATE_TRUNC('day', TO_TIMESTAMP(created_at) AT TIME ZONE '%s')::DATE", tzName)
		sqlSuffix = `ON CONFLICT (date, user_id, channel_id, model_name) DO UPDATE SET
		request_count = EXCLUDED.request_count,
		quota = EXCLUDED.quota,
		prompt_tokens = EXCLUDED.prompt_tokens,
		completion_tokens = EXCLUDED.completion_tokens,
		request_time = EXCLUDED.request_time`
	} else {
		sqlPrefix = "INSERT INTO"
		// MySQL动态获取时区偏移
		_, mysqlOffset := getTimezoneOffset()
		sqlDate = fmt.Sprintf("DATE_FORMAT(CONVERT_TZ(FROM_UNIXTIME(created_at), '+00:00', '%s'), '%%Y-%%m-%%d')", mysqlOffset)
		sqlSuffix = `ON DUPLICATE KEY UPDATE
		request_count = VALUES(request_count),
		quota = VALUES(quota),
		prompt_tokens = VALUES(prompt_tokens),
		completion_tokens = VALUES(completion_tokens),
		request_time = VALUES(request_time)`
	}

	// 使用系统本地时区计算时间戳
	location := time.Local
	if tzEnv := os.Getenv("TZ"); tzEnv != "" {
		if loc, err := time.LoadLocation(tzEnv); err == nil {
			location = loc
		}
	}

	now := time.Now().In(location)
	todayTimestamp := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, location).Unix()

	switch updateType {
	case StatisticsUpdateTypeToDay:
		sqlWhere = fmt.Sprintf("AND created_at >= %d", todayTimestamp)
	case StatisticsUpdateTypeYesterday:
		yesterdayTimestamp := todayTimestamp - 86400
		sqlWhere = fmt.Sprintf("AND created_at >= %d AND created_at < %d", yesterdayTimestamp, todayTimestamp)
	}

	err := DB.Exec(fmt.Sprintf(sql, sqlPrefix, sqlDate, sqlWhere, sqlSuffix)).Error
	return err
}
