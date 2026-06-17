package entity

import (
	"time"
)

type Notification struct {
	ID          int64     `json:"id"`
	RecipientID int64     `json:"recipient_id"`
	Type        string    `json:"type"`
	Payload     string    `json:"payload"`
	ReadAt      NullTime  `json:"read_at"`
	CreatedAt   time.Time `json:"created_at"`
}
