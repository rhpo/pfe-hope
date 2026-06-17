package entity

import (
	"time"
)

type Teacher struct {
	ID                 int64      `json:"id"`
	ProfileID          int64      `json:"profile_id"`
	Grade              NullString `json:"grade"`
	DepartmentID       *int64     `json:"department_id"`
	AvailabilityStatus string     `json:"availability_status"`
	UnavailableUntil   NullTime   `json:"unavailable_until"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`

	Profile    *Profile    `json:"profile,omitempty"`
	Department *Department `json:"department,omitempty"`
	Domaines   []*Domain   `json:"domaines,omitempty"`
}
