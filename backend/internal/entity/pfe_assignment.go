package entity

import (
	"time"
)

type PfeAssignment struct {
	ID             int64      `json:"id"`
	PfeCode        string     `json:"pfe_code"`
	SubjectID      int64      `json:"subject_id"`
	AcademicYearID int64      `json:"academic_year_id"`
	StudentID      int64      `json:"student_id"`
	Student2ID     NullInt64  `json:"student2_id"`
	Student3ID     NullInt64  `json:"student3_id"`
	SupervisorID   int64      `json:"supervisor_id"`
	CoSupervisorID NullInt64  `json:"co_supervisor_id"`
	MemoireURL     NullString `json:"memoire_url"`
	Status         string     `json:"status"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`

	Subject      *PfeSubject   `json:"subject,omitempty"`
	AcademicYear *AcademicYear `json:"academic_year,omitempty"`
	Student      *Student      `json:"student,omitempty"`
	Student2     *Student      `json:"student2,omitempty"`
	Student3     *Student      `json:"student3,omitempty"`
	Supervisor   *Teacher      `json:"supervisor,omitempty"`
	CoSupervisor *Teacher      `json:"co_supervisor,omitempty"`
}
