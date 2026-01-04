// model/updateLog.go

package model

import (
	"errors"
	"fmt"
	"go-backend/common"
	"time"

	"gorm.io/gorm"
)

type UpdateLog struct {
	ID          uint       `gorm:"primaryKey" json:"id"`
	Version     string     `json:"version"`
	LastCheck   *time.Time `json:"last_check,omitempty"`
	Status      string     `json:"status"`
	Error       string     `json:"error"`
	UpdatedTime time.Time  `json:"updated_time"`
}

func InitUpdateLogTable() error {
	// search for first record
	var log UpdateLog
	if err := DB.First(&log).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			// create initial record
			initialLog := UpdateLog{
				Version:     common.Version + common.Build,
				Status:      "initialized",
				UpdatedTime: time.Now(),
			}
			if err := DB.Create(&initialLog).Error; err != nil {
				return err
			}
			return nil
		} else {
			return err
		}
	}
	common.SysLog("UpdateLog table already initialized")
	return nil
}

func UpdateTime() error {
	return DB.Model(&UpdateLog{}).Where("id = ?", 1).Update("updated_time", time.Now()).Error
}

func SetStatus(status string) error {
	return DB.Model(&UpdateLog{}).Where("id = ?", 1).Updates(map[string]interface{}{
		"status": status,
	}).Error
}

func SetLastCheck(t time.Time) error {
	return DB.Model(&UpdateLog{}).Where("id = ?", 1).Update("last_check", t).Error
}

func ClearUpdateError() error {
	return DB.Model(&UpdateLog{}).Where("id = ?", 1).Update("error", " ").Error
}

func WriteUpdateErrorLog(errMsg string) error {
	// get old log
	var log UpdateLog
	if err := DB.First(&log).Error; err != nil {
		return err
	}
	timeFmt := time.Now().Format("2006-01-02 15:04:05")
	errMsg = fmt.Sprintf("[%s] %s", timeFmt, errMsg)
	log.Error += "\n" + errMsg
	return DB.Save(&log).Error
}

func LastUpdateCheck() (*time.Time, error) {
	var log UpdateLog
	if err := DB.First(&log).Error; err != nil {
		return nil, err
	}
	return log.LastCheck, nil
}
