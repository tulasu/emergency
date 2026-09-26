package models

import (
	"time"

	"traineebox/internal/calls/domain/value_objects"

	"github.com/google/uuid"
)

type Call struct {
	ID         uuid.UUID
	AttemptID  uuid.UUID
	TicketID   uuid.UUID
	UserID     uuid.UUID
	ScenarioID string
	BankDigest string
	ChannelID  string
	Status     string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

func (c Call) BlocksRecall() bool {
	return c.Status == value_objects.StatusAnswered || c.Status == value_objects.StatusCompleted ||
		c.Status == value_objects.StatusRinging || c.Status == value_objects.StatusOriginating
}

type Turn struct {
	N         int
	Utterance string
	Reply     string
	Style     string
}
