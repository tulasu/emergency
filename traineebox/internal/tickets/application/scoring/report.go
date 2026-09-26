package scoring

import (
	"traineebox/internal/tickets/domain/models"

	"github.com/google/uuid"
)

func BuildReport(tickets []models.Ticket, refs map[uuid.UUID]models.ReferenceAnswer, answers map[uuid.UUID]models.Answer) (models.Report, error) {
	items := make([]models.ReportItem, 0, len(tickets))
	total := 0
	for _, t := range tickets {
		ref, ok := refs[t.ID]
		if !ok {
			return models.Report{}, nil
		}
		ans := models.EmptyAnswer()
		if a, exists := answers[t.ID]; exists {
			ans = a
		}
		score, err := Score(ref, ans)
		if err != nil {
			return models.Report{}, err
		}
		items = append(items, models.ReportItem{
			TicketID: t.ID,
			Score:    score,
			Errors:   diffErrors(ref, ans),
		})
		total += score
	}
	overall := 100
	if len(items) > 0 {
		overall = total / len(items)
		if overall < 1 {
			overall = 1
		}
		if overall > 100 {
			overall = 100
		}
	}
	return models.Report{OverallScore: overall, Items: items}, nil
}

func diffErrors(ref models.ReferenceAnswer, ans models.Answer) []models.ReportError {
	out := make([]models.ReportError, 0)
	actualType := ""
	if ans.IncidentTypeCode != nil {
		actualType = *ans.IncidentTypeCode
	}
	if actualType != ref.IncidentTypeCode {
		out = append(out, models.ReportError{Field: "incident_type_code", Expected: ref.IncidentTypeCode, Actual: actualType})
	}
	if !setEqual(ref.TagCodes, ans.TagCodes) {
		out = append(out, models.ReportError{Field: "tag_codes", Expected: ref.TagCodes, Actual: ans.TagCodes})
	}
	if !setEqual(ref.ServiceCodes, ans.ServiceCodes) {
		out = append(out, models.ReportError{Field: "service_codes", Expected: ref.ServiceCodes, Actual: ans.ServiceCodes})
	}
	if ref.ApplicantLastName != "" && ans.ApplicantLastName != ref.ApplicantLastName {
		out = append(out, models.ReportError{Field: "applicant_last_name", Expected: ref.ApplicantLastName, Actual: ans.ApplicantLastName})
	}
	if ref.ApplicantFirstName != "" && ans.ApplicantFirstName != ref.ApplicantFirstName {
		out = append(out, models.ReportError{Field: "applicant_first_name", Expected: ref.ApplicantFirstName, Actual: ans.ApplicantFirstName})
	}
	if ref.CallerNumber != "" && ans.CallerNumber != ref.CallerNumber {
		out = append(out, models.ReportError{Field: "caller_number", Expected: ref.CallerNumber, Actual: ans.CallerNumber})
	}
	if ref.DictatedNumber != "" && ans.DictatedNumber != ref.DictatedNumber {
		out = append(out, models.ReportError{Field: "dictated_number", Expected: ref.DictatedNumber, Actual: ans.DictatedNumber})
	}
	return out
}
