package models

import "time"

type User struct {
	ID        uint      `json:"id"`
	FamilyID  uint      `json:"family_id"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	Phone     string    `json:"phone"`
	Password  string    `json:"-"`
	Points    int       `json:"points"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Family struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Users     []User    `json:"users,omitempty"`
}
