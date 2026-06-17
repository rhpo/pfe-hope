package entity

import "time"

type Speciality struct {
	ID           int64     `json:"id"`
	Name         string    `json:"name"`
	Code         string    `json:"code"`
	YearType     string    `json:"year_type"`
	DepartmentID *int64    `json:"department_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Department *Department `json:"department,omitempty"`
}
