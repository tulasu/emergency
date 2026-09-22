package models

import (
	"traineebox/internal/tickets/domain/errs"

	"github.com/google/uuid"
)

type ReferenceAnswer struct {
	TicketID           uuid.UUID
	IncidentTypeID     uuid.UUID
	TagIDs             []uuid.UUID
	ServiceIDs         []uuid.UUID
	ApplicantLastName  string
	ApplicantFirstName string
	CallerNumber       string
	DictatedNumber     string
}

func NewReferenceAnswer(
	ticketID, incidentTypeID uuid.UUID,
	tagIDs, serviceIDs []uuid.UUID,
	lastName, firstName, caller, dictated string,
) (ReferenceAnswer, error) {
	if incidentTypeID == uuid.Nil {
		return ReferenceAnswer{}, errs.ErrInvalidInput
	}
	if tagIDs == nil {
		tagIDs = []uuid.UUID{}
	}
	if serviceIDs == nil {
		serviceIDs = []uuid.UUID{}
	}
	return ReferenceAnswer{
		TicketID:           ticketID,
		IncidentTypeID:     incidentTypeID,
		TagIDs:             append([]uuid.UUID(nil), tagIDs...),
		ServiceIDs:         append([]uuid.UUID(nil), serviceIDs...),
		ApplicantLastName:  lastName,
		ApplicantFirstName: firstName,
		CallerNumber:       caller,
		DictatedNumber:     dictated,
	}, nil
}

func (r ReferenceAnswer) ValidateTagsAgainstType(allowedByType map[uuid.UUID]struct{}) error {
	for _, id := range r.TagIDs {
		if _, ok := allowedByType[id]; !ok {
			return errs.ErrInvalidTags
		}
	}
	return nil
}
