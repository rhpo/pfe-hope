package entity

import (
	"time"
)

type AuditLog struct {
	ID        int64      `json:"id"`
	ActorID   int64      `json:"actor_id"`
	Action    string     `json:"action"`
	Entity    string     `json:"entity"`
	EntityID  NullInt64  `json:"entity_id"`
	Metadata  NullString `json:"metadata"`
	CreatedAt time.Time  `json:"created_at"`
}
