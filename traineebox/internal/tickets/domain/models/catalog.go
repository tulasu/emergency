package models

import "github.com/google/uuid"

type IncidentType struct {
	ID    uuid.UUID
	Code  string
	Title string
}

type IncidentTag struct {
	ID             uuid.UUID
	IncidentTypeID uuid.UUID
	Code           string
	Title          string
}

type EmergencyService struct {
	ID    uuid.UUID
	Code  string
	Title string
}
