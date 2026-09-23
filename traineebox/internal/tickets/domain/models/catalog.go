package models

import "github.com/google/uuid"

type TagSelectionMode string

const (
	TagSelectionSingle TagSelectionMode = "single"
	TagSelectionMulti  TagSelectionMode = "multi"
)

func ParseTagSelectionMode(s string) (TagSelectionMode, bool) {
	switch TagSelectionMode(s) {
	case TagSelectionSingle, TagSelectionMulti:
		return TagSelectionMode(s), true
	default:
		return "", false
	}
}

type IncidentType struct {
	ID    uuid.UUID
	Code  string
	Title string
}

type IncidentTagGroup struct {
	ID             uuid.UUID
	IncidentTypeID uuid.UUID
	Code           string
	Title          string
	SelectionMode  TagSelectionMode
	ParentTagID    *uuid.UUID
	SortOrder      int
	Tags           []IncidentTag
}

type IncidentTag struct {
	ID             uuid.UUID
	IncidentTypeID uuid.UUID
	GroupID        uuid.UUID
	Code           string
	Title          string
	SortOrder      int
}

type EmergencyService struct {
	ID    uuid.UUID
	Code  string
	Title string
}
