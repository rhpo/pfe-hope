package entity

import (
	"time"
)

type DefenseJury struct {
	ID                    int64     `json:"id"`
	AssignmentID          int64     `json:"assignment_id"`
	PresidentID           int64     `json:"president_id"`
	MemberID              int64     `json:"member_id"`
	PresidentConfirmed    bool      `json:"president_confirmed"`
	MemberConfirmed       bool      `json:"member_confirmed"`
	PresidentWantsPrinted bool      `json:"president_wants_printed"`
	MemberWantsPrinted    bool      `json:"member_wants_printed"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`

	Assignment *PfeAssignment `json:"assignment,omitempty"`
	President  *Teacher       `json:"president,omitempty"`
	Member     *Teacher       `json:"member,omitempty"`
}
