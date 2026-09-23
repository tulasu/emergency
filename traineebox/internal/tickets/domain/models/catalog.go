package models

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
	Code  string
	Title string
}

type IncidentTagGroup struct {
	IncidentTypeCode string
	Code             string
	Title            string
	SelectionMode    TagSelectionMode
	ParentTagCode    string
	SortOrder        int
	Tags             []IncidentTag
}

type IncidentTag struct {
	IncidentTypeCode string
	GroupCode        string
	Code             string
	Title            string
	SortOrder        int
}

type EmergencyService struct {
	Code  string
	Title string
}
