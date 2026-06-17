package entity

import (
	"time"
)

type Wish struct {
	ID             int64     `json:"id"`
	StudentID      int64     `json:"student_id"`
	SubjectID      int64     `json:"subject_id"`
	AcademicYearID int64     `json:"academic_year_id"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Student      *Student      `json:"student,omitempty"`
	Subject      *PfeSubject   `json:"subject,omitempty"`
	AcademicYear *AcademicYear `json:"academic_year,omitempty"`
}
