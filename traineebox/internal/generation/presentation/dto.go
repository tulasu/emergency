package presentation

import (
	"time"

	"traineebox/internal/generation/domain/models"

	"github.com/google/uuid"
)

type draftReferenceDTO struct {
	IncidentTypeCode   string   `json:"incident_type_code"`
	TagCodes           []string `json:"tag_codes"`
	ServiceCodes       []string `json:"service_codes"`
	ApplicantLastName  string   `json:"applicant_last_name,omitempty"`
	ApplicantFirstName string   `json:"applicant_first_name,omitempty"`
	CallerNumber       string   `json:"caller_number,omitempty"`
	DictatedNumber     string   `json:"dictated_number,omitempty"`
}

type jobDTO struct {
	ID                uuid.UUID         `json:"id"`
	GroupID           uuid.UUID         `json:"group_id"`
	CreatedBy         uuid.UUID         `json:"created_by"`
	Prompt            string            `json:"prompt"`
	Status            string            `json:"status"`
	Version           int               `json:"version"`
	ScenarioText      string            `json:"scenario_text"`
	DraftTitle        string            `json:"draft_title"`
	DraftReference    draftReferenceDTO `json:"draft_reference"`
	ErrorMessage      string            `json:"error_message,omitempty"`
	Attempts          int               `json:"attempts"`
	PublishedTicketID *uuid.UUID        `json:"published_ticket_id,omitempty"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
}

func toJobDTO(j models.Job) jobDTO {
	ref := j.DraftReference.Normalize()
	return jobDTO{
		ID:        j.ID,
		GroupID:   j.GroupID,
		CreatedBy: j.CreatedBy,
		Prompt:    j.Prompt,
		Status:    j.Status.String(),
		Version:   j.Version,
		ScenarioText: j.ScenarioText,
		DraftTitle:   j.DraftTitle,
		DraftReference: draftReferenceDTO{
			IncidentTypeCode:   ref.IncidentTypeCode,
			TagCodes:           ref.TagCodes,
			ServiceCodes:       ref.ServiceCodes,
			ApplicantLastName:  ref.ApplicantLastName,
			ApplicantFirstName: ref.ApplicantFirstName,
			CallerNumber:       ref.CallerNumber,
			DictatedNumber:     ref.DictatedNumber,
		},
		ErrorMessage:      j.ErrorMessage,
		Attempts:          j.Attempts,
		PublishedTicketID: j.PublishedTicketID,
		CreatedAt:         j.CreatedAt,
		UpdatedAt:         j.UpdatedAt,
	}
}

func fromDraftDTO(d draftReferenceDTO) models.DraftReference {
	return models.DraftReference{
		IncidentTypeCode:   d.IncidentTypeCode,
		TagCodes:           d.TagCodes,
		ServiceCodes:       d.ServiceCodes,
		ApplicantLastName:  d.ApplicantLastName,
		ApplicantFirstName: d.ApplicantFirstName,
		CallerNumber:       d.CallerNumber,
		DictatedNumber:     d.DictatedNumber,
	}.Normalize()
}
