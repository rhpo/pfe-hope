package entity

import "time"

type Profile struct {
	ID        int64     `json:"id"`
	Role      string    `json:"role"`
	FullName  string    `json:"full_name"`
	Email     string    `json:"email"`
	AvatarURL *string   `json:"avatar_url"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	Teacher *Teacher `json:"teacher,omitempty"`
	Student *Student `json:"student,omitempty"`
	Company *Company `json:"company,omitempty"`
}
