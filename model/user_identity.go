package model

import (
	"errors"
	"strings"
	"time"

	"gorm.io/gorm/clause"
)

type UserIdentity struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uint      `gorm:"index;not null" json:"user_id"`
	Provider  string    `gorm:"size:50;not null;index:idx_provider_subject,unique" json:"provider"`
	Subject   string    `gorm:"size:255;not null;index:idx_provider_subject,unique" json:"subject"`
	Email     string    `gorm:"size:255;index" json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func UpsertUserIdentity(userID uint, provider, subject, email string) error {
	provider = strings.TrimSpace(provider)
	subject = strings.TrimSpace(subject)
	if provider == "" || subject == "" {
		return errors.New("provider and subject are required")
	}

	record := UserIdentity{
		UserID:   userID,
		Provider: provider,
		Subject:  subject,
		Email:    strings.TrimSpace(email),
	}

	return DB.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "provider"},
			{Name: "subject"},
		},
		DoUpdates: clause.AssignmentColumns([]string{"user_id", "email", "updated_at"}),
	}).Create(&record).Error
}
