package entity

import (
	"time"
)

type AcademicYear struct {
	ID                int64     `json:"id"`
	Label             string    `json:"label"`  // ex: 2024-2025
	Status            string    `json:"status"` // active/cloturee
	SubmissionOpenAt  NullTime  `json:"submission_open_at"`
	SubmissionCloseAt NullTime  `json:"submission_close_at"`
	MaxWishes         int       `json:"max_wishes"`
	CreatedAt         time.Time `json:"created_at"`
	UpdatedAt         time.Time `json:"updated_at"`
}
