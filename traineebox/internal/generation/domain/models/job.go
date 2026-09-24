package models

import (
	"strings"
	"time"

	"traineebox/internal/generation/domain/errs"
	"traineebox/internal/generation/domain/value_objects"

	"github.com/google/uuid"
)

type Job struct {
	ID                uuid.UUID
	GroupID           uuid.UUID
	CreatedBy         uuid.UUID
	Prompt            string
	Status            value_objects.JobStatus
	Version           int
	ScenarioText      string
	DraftTitle        string
	DraftReference    DraftReference
	ErrorMessage      string
	Attempts          int
	PublishedTicketID *uuid.UUID
	ClaimedBy         string
	ClaimedAt         *time.Time
	LeaseUntil        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

func NewJob(groupID, createdBy uuid.UUID, prompt string) (Job, error) {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" {
		return Job{}, errs.ErrInvalidInput
	}
	now := time.Now().UTC()
	return Job{
		ID:             uuid.New(),
		GroupID:        groupID,
		CreatedBy:      createdBy,
		Prompt:         prompt,
		Status:         value_objects.JobStatusQueued,
		Version:        1,
		DraftReference: DraftReference{TagCodes: []string{}, ServiceCodes: []string{}},
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}

func (j *Job) ApplyDraft(title, scenario string, ref DraftReference) error {
	if !j.Status.CanEditDraft() {
		return errs.ErrInvalidState
	}
	title = strings.TrimSpace(title)
	if title == "" {
		return errs.ErrInvalidInput
	}
	j.DraftTitle = title
	j.ScenarioText = scenario
	j.DraftReference = ref.Normalize()
	j.UpdatedAt = time.Now().UTC()
	return nil
}

func (j *Job) MarkRetry() error {
	if !j.Status.CanRetry() {
		return errs.ErrInvalidState
	}
	j.Status = value_objects.JobStatusQueued
	j.ErrorMessage = ""
	j.ClaimedBy = ""
	j.ClaimedAt = nil
	j.LeaseUntil = nil
	j.UpdatedAt = time.Now().UTC()
	return nil
}

func (j *Job) Cancel() error {
	if !j.Status.CanCancel() {
		return errs.ErrInvalidState
	}
	j.Status = value_objects.JobStatusCancelled
	j.ClaimedBy = ""
	j.ClaimedAt = nil
	j.LeaseUntil = nil
	j.UpdatedAt = time.Now().UTC()
	return nil
}

func (j *Job) MarkPublished(ticketID uuid.UUID) error {
	if !j.Status.CanApprove() {
		return errs.ErrInvalidState
	}
	if j.DraftTitle == "" || j.DraftReference.IncidentTypeCode == "" {
		return errs.ErrInvalidInput
	}
	j.Status = value_objects.JobStatusPublished
	j.PublishedTicketID = &ticketID
	j.ClaimedBy = ""
	j.ClaimedAt = nil
	j.LeaseUntil = nil
	j.UpdatedAt = time.Now().UTC()
	return nil
}
