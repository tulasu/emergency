package presentation

import (
	"time"

	"traineebox/internal/tickets/domain/models"
)

type incidentTypeDTO struct {
	Code  string `json:"code"`
	Title string `json:"title"`
}

type incidentTagDTO struct {
	Code      string `json:"code"`
	Title     string `json:"title"`
	SortOrder int    `json:"sort_order"`
}

type tagGroupDTO struct {
	Code          string           `json:"code"`
	Title         string           `json:"title"`
	SelectionMode string           `json:"selection_mode"`
	ParentTagCode string           `json:"parent_tag_code,omitempty"`
	SortOrder     int              `json:"sort_order"`
	Tags          []incidentTagDTO `json:"tags"`
}

type serviceDTO struct {
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
	IncidentTypeCode   *string  `json:"incident_type_code,omitempty"`
	TagCodes           []string `json:"tag_codes"`
	ServiceCodes       []string `json:"service_codes"`
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
	IncidentTypeCode   string   `json:"incident_type_code"`
	TagCodes           []string `json:"tag_codes"`
	ServiceCodes       []string `json:"service_codes"`
	ApplicantLastName  string   `json:"applicant_last_name"`
	ApplicantFirstName string   `json:"applicant_first_name"`
	CallerNumber       string   `json:"caller_number"`
	DictatedNumber     string   `json:"dictated_number"`
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
	return answerDTO{
		IncidentTypeCode:   a.IncidentTypeCode,
		TagCodes:           append([]string(nil), a.TagCodes...),
		ServiceCodes:       append([]string(nil), a.ServiceCodes...),
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
		IncidentTypeCode:   r.IncidentTypeCode,
		TagCodes:           append([]string(nil), r.TagCodes...),
		ServiceCodes:       append([]string(nil), r.ServiceCodes...),
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
