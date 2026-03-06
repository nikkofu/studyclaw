package models

import (
	"time"

	"gorm.io/gorm"
)

type Family struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Name      string    `gorm:"size:255;not null" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Users     []User    `gorm:"foreignKey:FamilyID" json:"users,omitempty"`
}

type User struct {
	ID        uint           `gorm:"primaryKey" json:"id"`
	FamilyID  uint           `gorm:"index" json:"family_id"`
	Name      string         `gorm:"size:100;not null" json:"name"`
	Role      string         `gorm:"type:enum('parent','child');default:'child'" json:"role"`
	Phone     string         `gorm:"size:20;uniqueIndex" json:"phone"`
	Password  string         `gorm:"size:255;not null" json:"-"` // Omit in JSON
	Points    int            `gorm:"default:0" json:"points"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}
