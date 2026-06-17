package entity

import "time"

type Company struct {
	ID           int64     `json:"id"`
	ProfileID    int64     `json:"profile_id"`
	CompanyName  *string   `json:"company_name"`
	Sector       *string   `json:"sector"`
	Description  *string   `json:"description"`
	LogoURL      *string   `json:"logo_url"`
	ContactEmail *string   `json:"contact_email"`
	ContactPhone *string   `json:"contact_phone"`
	Website      *string   `json:"website"`
	IsVerified   bool      `json:"is_verified"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`

	Profile *Profile `json:"profile,omitempty"`
}
