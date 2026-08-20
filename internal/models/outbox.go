package models

import (
	"time"

	"github.com/google/uuid"
)

type Outbox struct {
	ID          int64      `json:"id"`
	AggregateID uuid.UUID  `json:"aggregate_id"`
	EventType   string     `json:"event_type"`
	Payload     []byte     `json:"payload"` // JSONB
	CreatedAt   time.Time  `json:"created_at"`
	ProcessedAt *time.Time `json:"processed_at,omitempty"`
}
