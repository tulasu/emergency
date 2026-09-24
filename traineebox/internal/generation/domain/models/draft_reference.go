package models

type DraftReference struct {
	IncidentTypeCode   string   `json:"incident_type_code"`
	TagCodes           []string `json:"tag_codes"`
	ServiceCodes       []string `json:"service_codes"`
	ApplicantLastName  string   `json:"applicant_last_name"`
	ApplicantFirstName string   `json:"applicant_first_name"`
	CallerNumber       string   `json:"caller_number"`
	DictatedNumber     string   `json:"dictated_number"`
}

func (d DraftReference) Normalize() DraftReference {
	if d.TagCodes == nil {
		d.TagCodes = []string{}
	}
	if d.ServiceCodes == nil {
		d.ServiceCodes = []string{}
	}
	return d
}
