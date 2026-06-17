package entity

import (
	"time"
)

type CompanyReport struct {
	ID             int64     `json:"id"`
	CompanyID      int64     `json:"company_id"`
	SubmittedBy    int64     `json:"submitted_by"`
	CorrectionType string    `json:"correction_type"`
	Description    string    `json:"description"`
	RequestedValue string    `json:"requested_value"`
	Status         string    `json:"status"`
	ResolvedAt     NullTime  `json:"resolved_at"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`

	Company *Company `json:"company,omitempty"`
}
