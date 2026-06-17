package entity

import (
	"time"
)

type Defense struct {
	ID              int64       `json:"id"`
	AssignmentID    int64       `json:"assignment_id"`
	JuryID          int64       `json:"jury_id"`
	ScheduledAt     NullTime    `json:"scheduled_at"`
	Room            NullString  `json:"room"`
	DefenseDeadline NullTime    `json:"defense_deadline"`
	Status          string      `json:"status"`
	Result          NullString  `json:"result"`
	FinalGrade      NullFloat64 `json:"final_grade"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`

	Assignment *PfeAssignment `json:"assignment,omitempty"`
	Jury       *DefenseJury   `json:"jury,omitempty"`
}
