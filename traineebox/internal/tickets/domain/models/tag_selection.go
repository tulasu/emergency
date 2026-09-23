package models

import (
	"traineebox/internal/tickets/domain/errs"

	"github.com/google/uuid"
)

// ValidateTagSelection checks tag_ids against catalog groups for the incident type:
// membership, parent visibility, and single/multi selection rules.
func ValidateTagSelection(incidentTypeID *uuid.UUID, tagIDs []uuid.UUID, groups []IncidentTagGroup) error {
	if len(tagIDs) == 0 {
		return nil
	}
	if incidentTypeID == nil || *incidentTypeID == uuid.Nil {
		return errs.ErrInvalidTags
	}

	selected := make(map[uuid.UUID]struct{}, len(tagIDs))
	for _, id := range tagIDs {
		selected[id] = struct{}{}
	}

	allowed := make(map[uuid.UUID]struct{})
	for i := range groups {
		g := &groups[i]
		if g.IncidentTypeID != *incidentTypeID {
			continue
		}
		for _, tag := range g.Tags {
			allowed[tag.ID] = struct{}{}
		}
	}

	for id := range selected {
		if _, ok := allowed[id]; !ok {
			return errs.ErrInvalidTags
		}
	}

	for i := range groups {
		g := &groups[i]
		if g.IncidentTypeID != *incidentTypeID {
			continue
		}
		visible := g.ParentTagID == nil
		if g.ParentTagID != nil {
			_, visible = selected[*g.ParentTagID]
		}

		count := 0
		for _, tag := range g.Tags {
			if _, ok := selected[tag.ID]; ok {
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
	return ValidateTagSelection(a.IncidentTypeID, a.TagIDs, groups)
}

func (r ReferenceAnswer) ValidateTagSelection(groups []IncidentTagGroup) error {
	typeID := r.IncidentTypeID
	return ValidateTagSelection(&typeID, r.TagIDs, groups)
}

// ValidateTagsAgainstType ensures tags belong to the incident type (legacy flat check).
func (a Answer) ValidateTagsAgainstType(allowedByType map[uuid.UUID]struct{}) error {
	if len(a.TagIDs) == 0 {
		return nil
	}
	if a.IncidentTypeID == nil {
		return errs.ErrInvalidTags
	}
	for _, id := range a.TagIDs {
		if _, ok := allowedByType[id]; !ok {
			return errs.ErrInvalidTags
		}
	}
	return nil
}

func (r ReferenceAnswer) ValidateTagsAgainstType(allowedByType map[uuid.UUID]struct{}) error {
	for _, id := range r.TagIDs {
		if _, ok := allowedByType[id]; !ok {
			return errs.ErrInvalidTags
		}
	}
	return nil
}
