package presentation

import (
	"time"

	"traineebox/internal/tickets/domain/models"

	"github.com/google/uuid"
)

type incidentTypeDTO struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Title string `json:"title"`
}

type incidentTagDTO struct {
	ID             string `json:"id"`
	IncidentTypeID string `json:"incident_type_id"`
	Code           string `json:"code"`
	Title          string `json:"title"`
}

type serviceDTO struct {
	ID    string `json:"id"`
	Code  string `json:"code"`
	Title string `json:"title"`
}

type ticketDTO struct {
	ID              string  `json:"id"`
	GroupID         string  `json:"group_id"`
	Title           string  `json:"title"`
	Body            string  `json:"body"`
	MaxAttempts     *int    `json:"max_attempts,omitempty"`
	AvailableFrom   *string `json:"available_from,omitempty"`
	AvailableUntil  *string `json:"available_until,omitempty"`
	DurationSeconds *int    `json:"duration_seconds,omitempty"`
	CreatedBy       string  `json:"created_by"`
	CreatedAt       string  `json:"created_at"`
}

type answerDTO struct {
	IncidentTypeID     *string  `json:"incident_type_id,omitempty"`
	TagIDs             []string `json:"tag_ids"`
	ServiceIDs         []string `json:"service_ids"`
	ApplicantLastName  string   `json:"applicant_last_name"`
	ApplicantFirstName string   `json:"applicant_first_name"`
	CallerNumber       string   `json:"caller_number"`
	DictatedNumber     string   `json:"dictated_number"`
	Notes              string   `json:"notes"`
}

type attemptDTO struct {
	ID         string    `json:"id"`
	TicketID   string    `json:"ticket_id"`
	UserID     string    `json:"user_id"`
	AttemptNo  int       `json:"attempt_no"`
	Status     string    `json:"status"`
	StartedAt  string    `json:"started_at"`
	DeadlineAt *string   `json:"deadline_at,omitempty"`
	FinishedAt *string   `json:"finished_at,omitempty"`
	Score      *int      `json:"score,omitempty"`
	Answer     answerDTO `json:"answer"`
}

type referenceAnswerDTO struct {
	TicketID           string   `json:"ticket_id"`
	IncidentTypeID     string   `json:"incident_type_id"`
	TagIDs             []string `json:"tag_ids"`
	ServiceIDs         []string `json:"service_ids"`
	ApplicantLastName  string   `json:"applicant_last_name"`
	ApplicantFirstName string   `json:"applicant_first_name"`
	CallerNumber       string   `json:"caller_number"`
	DictatedNumber     string   `json:"dictated_number"`
}

type emptyOutput struct {
	Body struct{}
}

func toTicketDTO(t models.Ticket) ticketDTO {
	return ticketDTO{
		ID:              t.ID.String(),
		GroupID:         t.GroupID.String(),
		Title:           t.Title.String(),
		Body:            t.Body,
		MaxAttempts:     t.MaxAttempts,
		AvailableFrom:   formatTimePtr(t.AvailableFrom),
		AvailableUntil:  formatTimePtr(t.AvailableUntil),
		DurationSeconds: t.DurationSeconds,
		CreatedBy:       t.CreatedBy.String(),
		CreatedAt:       t.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toAttemptDTO(a models.Attempt) attemptDTO {
	return attemptDTO{
		ID:         a.ID.String(),
		TicketID:   a.TicketID.String(),
		UserID:     a.UserID.String(),
		AttemptNo:  a.AttemptNo,
		Status:     a.Status.String(),
		StartedAt:  a.StartedAt.UTC().Format(time.RFC3339Nano),
		DeadlineAt: formatTimePtr(a.DeadlineAt),
		FinishedAt: formatTimePtr(a.FinishedAt),
		Score:      a.Score,
		Answer:     toAnswerDTO(a.Answer),
	}
}

func toAnswerDTO(a models.Answer) answerDTO {
	var typeID *string
	if a.IncidentTypeID != nil {
		s := a.IncidentTypeID.String()
		typeID = &s
	}
	return answerDTO{
		IncidentTypeID:     typeID,
		TagIDs:             uuidsToStrings(a.TagIDs),
		ServiceIDs:         uuidsToStrings(a.ServiceIDs),
		ApplicantLastName:  a.ApplicantLastName,
		ApplicantFirstName: a.ApplicantFirstName,
		CallerNumber:       a.CallerNumber,
		DictatedNumber:     a.DictatedNumber,
		Notes:              a.Notes.String(),
	}
}

func toReferenceDTO(r models.ReferenceAnswer) referenceAnswerDTO {
	return referenceAnswerDTO{
		TicketID:           r.TicketID.String(),
		IncidentTypeID:     r.IncidentTypeID.String(),
		TagIDs:             uuidsToStrings(r.TagIDs),
		ServiceIDs:         uuidsToStrings(r.ServiceIDs),
		ApplicantLastName:  r.ApplicantLastName,
		ApplicantFirstName: r.ApplicantFirstName,
		CallerNumber:       r.CallerNumber,
		DictatedNumber:     r.DictatedNumber,
	}
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}

func uuidsToStrings(ids []uuid.UUID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func parseOptionalUUID(s *string) (*uuid.UUID, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	id, err := uuid.Parse(*s)
	if err != nil {
		return nil, err
	}
	return &id, nil
}

func parseUUIDList(ss []string) ([]uuid.UUID, error) {
	out := make([]uuid.UUID, 0, len(ss))
	for _, s := range ss {
		id, err := uuid.Parse(s)
		if err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	return out, nil
}

func parseOptionalTime(s *string) (*time.Time, error) {
	if s == nil || *s == "" {
		return nil, nil
	}
	t, err := time.Parse(time.RFC3339Nano, *s)
	if err != nil {
		t, err = time.Parse(time.RFC3339, *s)
		if err != nil {
			return nil, err
		}
	}
	u := t.UTC()
	return &u, nil
}
