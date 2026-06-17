package entity

import (
	"time"
)

type PfeSubject struct {
	ID                    int64      `json:"id"`
	Title                 string     `json:"title"`
	Description           string     `json:"description"`
	GroupType             string     `json:"group_type"` // monome/binome/trinome
	ProposerID            int64      `json:"proposer_id"`
	ProposerRole          string     `json:"proposer_role"` // teacher/company
	CompanyID             NullInt64  `json:"company_id"`
	AcademicYearID        int64      `json:"academic_year_id"`
	Validator1ID          NullInt64  `json:"validator1_id"`
	Validator2ID          NullInt64  `json:"validator2_id"`
	Validator1Decision    NullString `json:"validator1_decision"` // valide/accepte_sous_reserve/refuse
	Validator2Decision    NullString `json:"validator2_decision"`
	Validator1Comment     NullString `json:"validator1_comment"`
	Validator2Comment     NullString `json:"validator2_comment"`
	Status                string     `json:"status"` // en_attente/valide/accepte_sous_reserve/refuse/expire
	CoSupervisorID        NullInt64  `json:"co_supervisor_id"`
	PreAssignedStudentIDs NullString `json:"pre_assigned_student_ids"` // JSON array d'IDs
	CreatedAt             time.Time  `json:"created_at"`
	UpdatedAt             time.Time  `json:"updated_at"`

	IsAssigned bool `json:"is_assigned"`

	Proposer     *Profile  `json:"proposer,omitempty"`
	Company      *Company  `json:"company,omitempty"`
	Validator1   *Teacher  `json:"validator1,omitempty"`
	Validator2   *Teacher  `json:"validator2,omitempty"`
	CoSupervisor *Teacher  `json:"co_supervisor,omitempty"`
	Domains      []*Domain `json:"domains,omitempty"`
}
