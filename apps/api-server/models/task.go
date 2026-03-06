package models

import (
	"time"

	"gorm.io/gorm"
)

type Task struct {
	ID          uint           `gorm:"primaryKey" json:"id"`
	FamilyID    uint           `gorm:"index" json:"family_id"`
	AssigneeID  uint           `gorm:"index" json:"assignee_id"` // Child UserID
	Title       string         `gorm:"size:255;not null" json:"title"`
	Subject     string         `gorm:"size:100" json:"subject"` // e.g. Math, English
	Status      string         `gorm:"type:enum('pending','completed','verified');default:'pending'" json:"status"`
	PointsValue int            `gorm:"default:1" json:"points_value"`       // Points earned upon completion
	RawText     string         `gorm:"type:text" json:"raw_text,omitempty"` // Original text from parent
	Metadata    string         `gorm:"type:json" json:"metadata,omitempty"` // Extra details like vocab lists
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `gorm:"index" json:"-"`
}
