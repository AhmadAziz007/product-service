package model

import (
	"time"

	"gorm.io/gorm"
)

const (
	Published   = true
	Unpublished = false
)

type Category struct {
	ID          int64          `gorm:"primaryKey"`
	ParentID    *int64         `gorm:"column:parent_id"`
	Name        string         `gorm:"column:name;not null"`
	Icon        string         `gorm:"column:icon;not null"`
	Status      bool           `gorm:"column:status;default:true"`
	Slug        string         `gorm:"column:slug;unique"`
	Description string         `gorm:"column:description"`
	CreatedBy   *int64         `gorm:"column:created_by"`
	UpdatedBy   *int64         `gorm:"column:updated_by"`
	CreatedAt   time.Time      `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	UpdatedAt   time.Time      `gorm:"column:updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"column:deleted_at;index"`
	Products    []Product      `gorm:"foreignKey:CategorySlug;references:Slug"`
}
