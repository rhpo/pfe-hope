package entity

import (
	"time"
)


type PfeProgressReport struct {
	ID           int64      `json:"id"`
	AssignmentID int64      `json:"assignment_id"`
	MeetingDate  time.Time  `json:"meeting_date"`
	Duration     int        `json:"duration"`
	MeetingType  string     `json:"meeting_type"`
	Topics       string     `json:"topics"`
	Status       string     `json:"status"`
	Observation  NullString `json:"observation"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`


	Assignment *PfeAssignment `json:"assignment,omitempty"`
}
