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
	ID          string `json:"id"`
	VariantID   string `json:"variant_id"`
	TopicID     string `json:"topic_id"`
	Title       string `json:"title"`
	Body        string `json:"body"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
	AudioStatus string `json:"audio_status"`
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

type reportErrorDTO struct {
	Field    string `json:"field"`
	Expected any    `json:"expected,omitempty"`
	Actual   any    `json:"actual,omitempty"`
}

type reportItemDTO struct {
	TicketID string           `json:"ticket_id"`
	Score    int              `json:"score"`
	Errors   []reportErrorDTO `json:"errors"`
}

type reportDTO struct {
	OverallScore int             `json:"overall_score"`
	Items        []reportItemDTO `json:"items"`
}

type attemptDTO struct {
	ID         string               `json:"id"`
	VariantID  string               `json:"variant_id"`
	UserID     string               `json:"user_id"`
	GrantedBy  string               `json:"granted_by"`
	AttemptNo  int                  `json:"attempt_no"`
	Status     string               `json:"status"`
	StartedAt  *string              `json:"started_at,omitempty"`
	DeadlineAt *string              `json:"deadline_at,omitempty"`
	FinishedAt *string              `json:"finished_at,omitempty"`
	Score      *int                 `json:"score,omitempty"`
	Answers    map[string]answerDTO `json:"answers"`
	Report     reportDTO            `json:"report"`
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
		ID: t.ID.String(), VariantID: t.VariantID.String(), TopicID: t.TopicID.String(),
		Title: t.Title.String(), Body: t.Body, CreatedBy: t.CreatedBy.String(),
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339Nano), AudioStatus: t.AudioStatus,
	}
}

func toAttemptDTO(a models.Attempt) attemptDTO {
	answers := make(map[string]answerDTO, len(a.Answers))
	for id, ans := range a.Answers {
		answers[id.String()] = toAnswerDTO(ans)
	}
	return attemptDTO{
		ID: a.ID.String(), VariantID: a.VariantID.String(), UserID: a.UserID.String(),
		GrantedBy: a.GrantedBy.String(), AttemptNo: a.AttemptNo, Status: a.Status.String(),
		StartedAt: formatTimePtr(a.StartedAt), DeadlineAt: formatTimePtr(a.DeadlineAt),
		FinishedAt: formatTimePtr(a.FinishedAt), Score: a.Score, Answers: answers, Report: toReportDTO(a.Report),
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

func toReportDTO(r models.Report) reportDTO {
	items := make([]reportItemDTO, 0, len(r.Items))
	for _, it := range r.Items {
		errs := make([]reportErrorDTO, 0, len(it.Errors))
		for _, e := range it.Errors {
			errs = append(errs, reportErrorDTO{Field: e.Field, Expected: e.Expected, Actual: e.Actual})
		}
		items = append(items, reportItemDTO{TicketID: it.TicketID.String(), Score: it.Score, Errors: errs})
	}
	return reportDTO{OverallScore: r.OverallScore, Items: items}
}

func toReferenceDTO(r models.ReferenceAnswer) referenceAnswerDTO {
	return referenceAnswerDTO{
		TicketID: r.TicketID.String(), IncidentTypeCode: r.IncidentTypeCode,
		TagCodes: append([]string(nil), r.TagCodes...), ServiceCodes: append([]string(nil), r.ServiceCodes...),
		ApplicantLastName: r.ApplicantLastName, ApplicantFirstName: r.ApplicantFirstName,
		CallerNumber: r.CallerNumber, DictatedNumber: r.DictatedNumber,
	}
}

func formatTimePtr(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.UTC().Format(time.RFC3339Nano)
	return &s
}
