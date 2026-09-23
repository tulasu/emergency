package models

import (
	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type Answer struct {
	IncidentTypeID     *uuid.UUID
	TagIDs             []uuid.UUID
	ServiceIDs         []uuid.UUID
	ApplicantLastName  string
	ApplicantFirstName string
	CallerNumber       string
	DictatedNumber     string
	Notes              value_objects.Notes
}

func NewAnswer(
	incidentTypeID *uuid.UUID,
	tagIDs, serviceIDs []uuid.UUID,
	lastName, firstName, caller, dictated string,
	notes value_objects.Notes,
) Answer {
	if tagIDs == nil {
		tagIDs = []uuid.UUID{}
	}
	if serviceIDs == nil {
		serviceIDs = []uuid.UUID{}
	}
	return Answer{
		IncidentTypeID:     incidentTypeID,
		TagIDs:             append([]uuid.UUID(nil), tagIDs...),
		ServiceIDs:         append([]uuid.UUID(nil), serviceIDs...),
		ApplicantLastName:  lastName,
		ApplicantFirstName: firstName,
		CallerNumber:       caller,
		DictatedNumber:     dictated,
		Notes:              notes,
	}
}

func EmptyAnswer() Answer {
	return NewAnswer(nil, nil, nil, "", "", "", "", "")
}
