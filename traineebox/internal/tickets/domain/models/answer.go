package models

import (
	"traineebox/internal/tickets/domain/value_objects"
)

type Answer struct {
	IncidentTypeCode   *string
	TagCodes           []string
	ServiceCodes       []string
	ApplicantLastName  string
	ApplicantFirstName string
	CallerNumber       string
	DictatedNumber     string
	Notes              value_objects.Notes
}

func NewAnswer(
	incidentTypeCode *string,
	tagCodes, serviceCodes []string,
	lastName, firstName, caller, dictated string,
	notes value_objects.Notes,
) Answer {
	if tagCodes == nil {
		tagCodes = []string{}
	}
	if serviceCodes == nil {
		serviceCodes = []string{}
	}
	return Answer{
		IncidentTypeCode:   incidentTypeCode,
		TagCodes:           append([]string(nil), tagCodes...),
		ServiceCodes:       append([]string(nil), serviceCodes...),
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
