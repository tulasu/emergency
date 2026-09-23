package models

import (
	"traineebox/internal/tickets/domain/errs"

	"github.com/google/uuid"
)

type ReferenceAnswer struct {
	TicketID           uuid.UUID
	IncidentTypeCode   string
	TagCodes           []string
	ServiceCodes       []string
	ApplicantLastName  string
	ApplicantFirstName string
	CallerNumber       string
	DictatedNumber     string
}

func NewReferenceAnswer(
	ticketID uuid.UUID,
	incidentTypeCode string,
	tagCodes, serviceCodes []string,
	lastName, firstName, caller, dictated string,
) (ReferenceAnswer, error) {
	if incidentTypeCode == "" {
		return ReferenceAnswer{}, errs.ErrInvalidInput
	}
	if tagCodes == nil {
		tagCodes = []string{}
	}
	if serviceCodes == nil {
		serviceCodes = []string{}
	}
	return ReferenceAnswer{
		TicketID:           ticketID,
		IncidentTypeCode:   incidentTypeCode,
		TagCodes:           append([]string(nil), tagCodes...),
		ServiceCodes:       append([]string(nil), serviceCodes...),
		ApplicantLastName:  lastName,
		ApplicantFirstName: firstName,
		CallerNumber:       caller,
		DictatedNumber:     dictated,
	}, nil
}
