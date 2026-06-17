package entity

import "time"

type Promotion struct {
	ID             int64     `json:"id"`
	Label          string    `json:"label"`
	AcademicYearID int64     `json:"academic_year_id"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}
