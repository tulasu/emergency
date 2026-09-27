package models

import (
	"time"

	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type Ticket struct {
	ID              uuid.UUID
	VariantID       *uuid.UUID
	TopicID         uuid.UUID
	Title           value_objects.TicketTitle
	Body            string
	CreatedBy       uuid.UUID
	CreatedAt       time.Time
	ScenarioJSON    string
	ScenarioVersion string
	AudioDigest     string
	AudioStatus     string
	Mode            string
	Briefing        string
	Reference       string
}

func NewTicket(variantID, topicID uuid.UUID, title value_objects.TicketTitle, body string, createdBy uuid.UUID) Ticket {
	vid := variantID
	return Ticket{
		ID:          uuid.New(),
		VariantID:   &vid,
		TopicID:     topicID,
		Title:       title,
		Body:        body,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now().UTC(),
		AudioStatus: "none",
	}
}

func NewLibraryTicket(topicID uuid.UUID, title value_objects.TicketTitle, body string, createdBy uuid.UUID) Ticket {
	return Ticket{
		ID:          uuid.New(),
		VariantID:   nil,
		TopicID:     topicID,
		Title:       title,
		Body:        body,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now().UTC(),
		AudioStatus: "none",
	}
}

func (t Ticket) DeadlineFor(startedAt time.Time, durationSeconds *int) *time.Time {
	if durationSeconds == nil {
		return nil
	}
	d := startedAt.Add(time.Duration(*durationSeconds) * time.Second)
	return &d
}
