package application

import (
	"context"
	"errors"
	"time"

	"traineebox/internal/tickets/application/scoring"
	"traineebox/internal/tickets/domain/abilities"
	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/repositories"
	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type GrantAttempt struct {
	Attempts   repositories.AttemptRepository
	Curriculum repositories.TopicExists
}

type GrantAttemptInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	UserID    uuid.UUID
	VariantID uuid.UUID
}

func (uc GrantAttempt) Execute(ctx context.Context, in GrantAttemptInput) (models.Attempt, error) {
	if err := abilities.GrantAttempt(in.Role); err != nil {
		return models.Attempt{}, err
	}
	if err := uc.Curriculum.VariantExists(ctx, in.VariantID); err != nil {
		return models.Attempt{}, err
	}
	return issueAvailable(ctx, uc.Attempts, in.UserID, in.VariantID, in.ActorID)
}

func issueAvailable(ctx context.Context, attempts repositories.AttemptRepository, userID, variantID, grantedBy uuid.UUID) (models.Attempt, error) {
	open, err := attempts.FindOpen(ctx, variantID, userID)
	if err == nil {
		return open, nil
	}
	if !errors.Is(err, errs.ErrNotFound) {
		return models.Attempt{}, err
	}
	no, err := attempts.NextAttemptNo(ctx, variantID, userID)
	if err != nil {
		return models.Attempt{}, err
	}
	attempt := models.NewAvailableAttempt(variantID, userID, grantedBy, no)
	if err := attempts.Create(ctx, attempt); err != nil {
		if errors.Is(err, errs.ErrConflict) {
			return attempts.FindOpen(ctx, variantID, userID)
		}
		return models.Attempt{}, err
	}
	return attempt, nil
}

type IssueAvailable struct {
	Attempts repositories.AttemptRepository
}

func (uc IssueAvailable) IssueAvailable(ctx context.Context, userID, variantID, grantedBy uuid.UUID) error {
	_, err := issueAvailable(ctx, uc.Attempts, userID, variantID, grantedBy)
	return err
}

type StartAttempt struct {
	Tickets    repositories.TicketRepository
	Attempts   repositories.AttemptRepository
	Curriculum repositories.LessonClock
}

type StartAttemptInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	AttemptID uuid.UUID
}

func (uc StartAttempt) Execute(ctx context.Context, in StartAttemptInput) (models.Attempt, error) {
	if err := abilities.StartAttempt(in.Role); err != nil {
		return models.Attempt{}, err
	}
	attempt, err := uc.Attempts.FindByID(ctx, in.AttemptID)
	if err != nil {
		return models.Attempt{}, err
	}
	if attempt.UserID != in.ActorID {
		return models.Attempt{}, errs.ErrForbidden
	}
	now := time.Now().UTC()
	if attempt.IsExpired(now) {
		return expireAttempt(ctx, uc.Attempts, uc.Tickets, nil, attempt, now)
	}
	duration, err := uc.Curriculum.DurationSecondsByVariant(ctx, attempt.VariantID)
	if err != nil {
		return models.Attempt{}, err
	}
	if err := attempt.Start(now, deadlineFor(now, duration)); err != nil {
		return models.Attempt{}, err
	}
	if err := uc.Attempts.Save(ctx, attempt); err != nil {
		return models.Attempt{}, err
	}
	return attempt, nil
}

func deadlineFor(startedAt time.Time, durationSeconds *int) *time.Time {
	if durationSeconds == nil {
		return nil
	}
	d := startedAt.Add(time.Duration(*durationSeconds) * time.Second)
	return &d
}

type SaveAttemptAnswer struct {
	Tickets  repositories.TicketRepository
	Attempts repositories.AttemptRepository
	Catalog  repositories.CatalogRepository
}

type SaveAttemptAnswerInput struct {
	ActorID            uuid.UUID
	Role               value_objects.AccountRole
	AttemptID          uuid.UUID
	TicketID           uuid.UUID
	IncidentTypeCode   *string
	TagCodes           []string
	ServiceCodes       []string
	ApplicantLastName  string
	ApplicantFirstName string
	CallerNumber       string
	DictatedNumber     string
	Notes              string
}

func (uc SaveAttemptAnswer) Execute(ctx context.Context, in SaveAttemptAnswerInput) (models.Attempt, error) {
	attempt, err := loadOwnAttempt(ctx, uc.Attempts, in.AttemptID, in.ActorID, in.Role)
	if err != nil {
		return models.Attempt{}, err
	}
	ticket, err := uc.Tickets.FindByID(ctx, in.TicketID)
	if err != nil {
		return models.Attempt{}, err
	}
	if ticket.VariantID != attempt.VariantID {
		return models.Attempt{}, errs.ErrInvalidInput
	}
	now := time.Now().UTC()
	if attempt.IsExpired(now) {
		return expireAttempt(ctx, uc.Attempts, uc.Tickets, nil, attempt, now)
	}
	notes, err := value_objects.NewNotes(in.Notes)
	if err != nil {
		return models.Attempt{}, err
	}
	answer := models.NewAnswer(
		in.IncidentTypeCode, in.TagCodes, in.ServiceCodes,
		in.ApplicantLastName, in.ApplicantFirstName, in.CallerNumber, in.DictatedNumber, notes,
	)
	if err := validateAnswer(ctx, uc.Catalog, answer); err != nil {
		return models.Attempt{}, err
	}
	if err := attempt.SaveDraft(in.TicketID, answer, now); err != nil {
		return models.Attempt{}, err
	}
	if err := uc.Attempts.Save(ctx, attempt); err != nil {
		return models.Attempt{}, err
	}
	return attempt, nil
}

type SubmitAttempt struct {
	Tickets  repositories.TicketRepository
	Attempts repositories.AttemptRepository
}

type SubmitAttemptInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	AttemptID uuid.UUID
}

func (uc SubmitAttempt) Execute(ctx context.Context, in SubmitAttemptInput) (models.Attempt, error) {
	attempt, err := loadOwnAttempt(ctx, uc.Attempts, in.AttemptID, in.ActorID, in.Role)
	if err != nil {
		return models.Attempt{}, err
	}
	now := time.Now().UTC()
	if attempt.IsExpired(now) {
		return expireAttempt(ctx, uc.Attempts, uc.Tickets, nil, attempt, now)
	}
	report, err := buildAttemptReport(ctx, uc.Tickets, attempt)
	if err != nil {
		return models.Attempt{}, err
	}
	if err := attempt.Submit(now, report, report.OverallScore); err != nil {
		return models.Attempt{}, err
	}
	if err := uc.Attempts.Save(ctx, attempt); err != nil {
		return models.Attempt{}, err
	}
	return attempt, nil
}

type GetMyAttempt struct {
	Tickets  repositories.TicketRepository
	Attempts repositories.AttemptRepository
}

type GetMyAttemptInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	AttemptID uuid.UUID
}

func (uc GetMyAttempt) Execute(ctx context.Context, in GetMyAttemptInput) (models.Attempt, error) {
	attempt, err := loadOwnAttempt(ctx, uc.Attempts, in.AttemptID, in.ActorID, in.Role)
	if err != nil {
		return models.Attempt{}, err
	}
	now := time.Now().UTC()
	if attempt.IsExpired(now) {
		return expireAttempt(ctx, uc.Attempts, uc.Tickets, nil, attempt, now)
	}
	return attempt, nil
}

type ListMyAttempts struct {
	Attempts repositories.AttemptRepository
}

type ListMyAttemptsInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	VariantID uuid.UUID
}

func (uc ListMyAttempts) Execute(ctx context.Context, in ListMyAttemptsInput) ([]models.Attempt, error) {
	if err := abilities.ViewOwnAttempt(in.Role); err != nil {
		return nil, err
	}
	if in.Role == value_objects.AccountRoleStudent {
		return uc.Attempts.ListByVariantUser(ctx, in.VariantID, in.ActorID)
	}
	return uc.Attempts.ListByVariantUser(ctx, in.VariantID, in.ActorID)
}

type GetAttemptReport struct {
	Attempts repositories.AttemptRepository
}

func (uc GetAttemptReport) Execute(ctx context.Context, actorID uuid.UUID, role value_objects.AccountRole, attemptID uuid.UUID) (models.Attempt, error) {
	return loadOwnAttempt(ctx, uc.Attempts, attemptID, actorID, role)
}

func expireAttempt(
	ctx context.Context,
	attempts repositories.AttemptRepository,
	tickets repositories.TicketRepository,
	_ repositories.CatalogRepository,
	attempt models.Attempt,
	now time.Time,
) (models.Attempt, error) {
	report := models.EmptyReport()
	if tickets != nil {
		built, err := buildAttemptReport(ctx, tickets, attempt)
		if err != nil {
			return models.Attempt{}, err
		}
		report = built
	}
	if err := attempt.Expire(now, report, report.OverallScore); err != nil {
		return models.Attempt{}, err
	}
	if err := attempts.Save(ctx, attempt); err != nil {
		return models.Attempt{}, err
	}
	return attempt, nil
}

func buildAttemptReport(ctx context.Context, tickets repositories.TicketRepository, attempt models.Attempt) (models.Report, error) {
	list, err := tickets.ListByVariant(ctx, attempt.VariantID)
	if err != nil {
		return models.Report{}, err
	}
	if len(list) == 0 {
		return models.Report{}, errs.ErrInvalidInput
	}
	refs := make(map[uuid.UUID]models.ReferenceAnswer, len(list))
	for _, t := range list {
		ref, err := tickets.FindReference(ctx, t.ID)
		if err != nil {
			if errors.Is(err, errs.ErrNotFound) {
				return models.Report{}, errs.ErrNoReferenceAnswer
			}
			return models.Report{}, err
		}
		refs[t.ID] = ref
	}
	return scoring.BuildReport(list, refs, attempt.Answers)
}

func validateAnswer(ctx context.Context, catalog repositories.CatalogRepository, answer models.Answer) error {
	if answer.IncidentTypeCode != nil {
		if _, err := catalog.FindIncidentTypeByCode(ctx, *answer.IncidentTypeCode); err != nil {
			return err
		}
		groups, err := catalog.ListTagGroupsByType(ctx, *answer.IncidentTypeCode)
		if err != nil {
			return err
		}
		if err := answer.ValidateTagSelection(groups); err != nil {
			return err
		}
	} else if len(answer.TagCodes) > 0 {
		return errs.ErrInvalidTags
	}
	if len(answer.ServiceCodes) > 0 {
		ok, err := catalog.ServiceExists(ctx, answer.ServiceCodes)
		if err != nil {
			return err
		}
		if !ok {
			return errs.ErrInvalidInput
		}
	}
	return nil
}

func loadOwnAttempt(
	ctx context.Context,
	attempts repositories.AttemptRepository,
	attemptID, actorID uuid.UUID,
	role value_objects.AccountRole,
) (models.Attempt, error) {
	attempt, err := attempts.FindByID(ctx, attemptID)
	if err != nil {
		return models.Attempt{}, err
	}
	if attempt.UserID != actorID {
		if err := abilities.ViewAnyAttempt(role); err != nil {
			return models.Attempt{}, errs.ErrForbidden
		}
	} else if err := abilities.ViewOwnAttempt(role); err != nil {
		return models.Attempt{}, err
	}
	return attempt, nil
}
