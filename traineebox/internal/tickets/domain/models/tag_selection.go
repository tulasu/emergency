package models

import (
	"traineebox/internal/tickets/domain/errs"
)

// ValidateTagSelection checks tag codes against catalog groups for the incident type:
// membership, parent visibility, and single/multi selection rules.
func ValidateTagSelection(incidentTypeCode *string, tagCodes []string, groups []IncidentTagGroup) error {
	if len(tagCodes) == 0 {
		return nil
	}
	if incidentTypeCode == nil || *incidentTypeCode == "" {
		return errs.ErrInvalidTags
	}

	selected := make(map[string]struct{}, len(tagCodes))
	for _, code := range tagCodes {
		selected[code] = struct{}{}
	}

	allowed := make(map[string]struct{})
	for i := range groups {
		g := &groups[i]
		if g.IncidentTypeCode != *incidentTypeCode {
			continue
		}
		for _, tag := range g.Tags {
			allowed[tag.Code] = struct{}{}
		}
	}

	for code := range selected {
		if _, ok := allowed[code]; !ok {
			return errs.ErrInvalidTags
		}
	}

	for i := range groups {
		g := &groups[i]
		if g.IncidentTypeCode != *incidentTypeCode {
			continue
		}
		visible := g.ParentTagCode == ""
		if g.ParentTagCode != "" {
			_, visible = selected[g.ParentTagCode]
		}

		count := 0
		for _, tag := range g.Tags {
			if _, ok := selected[tag.Code]; ok {
				count++
			}
		}
		if !visible {
			if count > 0 {
				return errs.ErrInvalidTagSelection
			}
			continue
		}
		if g.SelectionMode == TagSelectionSingle && count > 1 {
			return errs.ErrInvalidTagSelection
		}
	}
	return nil
}

func (a Answer) ValidateTagSelection(groups []IncidentTagGroup) error {
	return ValidateTagSelection(a.IncidentTypeCode, a.TagCodes, groups)
}

func (r ReferenceAnswer) ValidateTagSelection(groups []IncidentTagGroup) error {
	typeCode := r.IncidentTypeCode
	return ValidateTagSelection(&typeCode, r.TagCodes, groups)
}
