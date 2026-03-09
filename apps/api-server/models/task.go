package models

import "time"

type Task struct {
	ID          uint      `json:"id"`
	FamilyID    uint      `json:"family_id"`
	AssigneeID  uint      `json:"assignee_id"`
	Title       string    `json:"title"`
	Subject     string    `json:"subject"`
	RawText     string    `json:"raw_text"`
	Status      string    `json:"status"` // "pending", "completed"
	PointsValue int       `json:"points_value"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
