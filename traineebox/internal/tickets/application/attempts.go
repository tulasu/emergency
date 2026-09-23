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

type StartAttempt struct {
	Tickets    repositories.TicketRepository
	Attempts   repositories.AttemptRepository
	Membership repositories.GroupMembership
}

type StartAttemptInput struct {
	ActorID  uuid.UUID
	Admin    bool
	TicketID uuid.UUID
}

func (uc StartAttempt) Execute(ctx context.Context, in StartAttemptInput) (models.Attempt, error) {
	ticket, err := uc.Tickets.FindByID(ctx, in.TicketID)
	if err != nil {
		return models.Attempt{}, err
	}
	if err := requireStudent(ctx, uc.Membership, ticket.GroupID, in.ActorID, in.Admin); err != nil {
		return models.Attempt{}, err
	}
	_, inProgressErr := uc.Attempts.FindInProgress(ctx, ticket.ID, in.ActorID)
	hasInProgress := inProgressErr == nil
	if inProgressErr != nil && !errors.Is(inProgressErr, errs.ErrNotFound) {
		return models.Attempt{}, inProgressErr
	}
	finished, err := uc.Attempts.CountFinished(ctx, ticket.ID, in.ActorID)
	if err != nil {
		return models.Attempt{}, err
	}
	now := time.Now().UTC()
	if err := ticket.CanStartAttempt(now, finished, hasInProgress); err != nil {
		return models.Attempt{}, err
	}
	no, err := uc.Attempts.NextAttemptNo(ctx, ticket.ID, in.ActorID)
	if err != nil {
		return models.Attempt{}, err
	}
	attempt := models.NewAttempt(ticket.ID, in.ActorID, no, now, ticket.DeadlineFor(now))
	if err := uc.Attempts.Create(ctx, attempt); err != nil {
		return models.Attempt{}, err
	}
	return attempt, nil
}

type SaveAttemptAnswer struct {
	Tickets    repositories.TicketRepository
	Attempts   repositories.AttemptRepository
	Catalog    repositories.CatalogRepository
	Membership repositories.GroupMembership
}

type SaveAttemptAnswerInput struct {
	ActorID            uuid.UUID
	Admin              bool
	AttemptID          uuid.UUID
	IncidentTypeID     *uuid.UUID
	TagIDs             []uuid.UUID
	ServiceIDs         []uuid.UUID
	ApplicantLastName  string
	ApplicantFirstName string
	CallerNumber       string
	DictatedNumber     string
	Notes              string
}

func (uc SaveAttemptAnswer) Execute(ctx context.Context, in SaveAttemptAnswerInput) (models.Attempt, error) {
	attempt, ticket, err := loadOwnAttempt(ctx, uc.Attempts, uc.Tickets, uc.Membership, in.AttemptID, in.ActorID, in.Admin)
	if err != nil {
		return models.Attempt{}, err
	}
	_ = ticket
	now := time.Now().UTC()
	if attempt.IsExpired(now) {
		return expireAttempt(ctx, uc.Tickets, uc.Attempts, attempt, now)
	}
	notes, err := value_objects.NewNotes(in.Notes)
	if err != nil {
		return models.Attempt{}, err
	}
	answer := models.NewAnswer(
		in.IncidentTypeID, in.TagIDs, in.ServiceIDs,
		in.ApplicantLastName, in.ApplicantFirstName, in.CallerNumber, in.DictatedNumber, notes,
	)
	if err := validateAnswer(ctx, uc.Catalog, answer); err != nil {
		return models.Attempt{}, err
	}
	if err := attempt.SaveDraft(answer, now); err != nil {
		return models.Attempt{}, err
	}
	if err := uc.Attempts.Save(ctx, attempt); err != nil {
		return models.Attempt{}, err
	}
	return attempt, nil
}

type SubmitAttempt struct {
	Tickets    repositories.TicketRepository
	Attempts   repositories.AttemptRepository
	Catalog    repositories.CatalogRepository
	Membership repositories.GroupMembership
}

type SubmitAttemptInput struct {
	ActorID   uuid.UUID
	Admin     bool
	AttemptID uuid.UUID
}

func (uc SubmitAttempt) Execute(ctx context.Context, in SubmitAttemptInput) (models.Attempt, error) {
	attempt, _, err := loadOwnAttempt(ctx, uc.Attempts, uc.Tickets, uc.Membership, in.AttemptID, in.ActorID, in.Admin)
	if err != nil {
		return models.Attempt{}, err
	}
	now := time.Now().UTC()
	if attempt.IsExpired(now) {
		return expireAttempt(ctx, uc.Tickets, uc.Attempts, attempt, now)
	}
	ref, err := uc.Tickets.FindReference(ctx, attempt.TicketID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return models.Attempt{}, errs.ErrNoReferenceAnswer
		}
		return models.Attempt{}, err
	}
	if err := attempt.Submit(now, ref, scoring.Score); err != nil {
		return models.Attempt{}, err
	}
	if err := uc.Attempts.Save(ctx, attempt); err != nil {
		return models.Attempt{}, err
	}
	return attempt, nil
}

type GetMyAttempt struct {
	Tickets    repositories.TicketRepository
	Attempts   repositories.AttemptRepository
	Membership repositories.GroupMembership
}

type GetMyAttemptInput struct {
	ActorID   uuid.UUID
	Admin     bool
	AttemptID uuid.UUID
}

func (uc GetMyAttempt) Execute(ctx context.Context, in GetMyAttemptInput) (models.Attempt, error) {
	attempt, _, err := loadOwnAttempt(ctx, uc.Attempts, uc.Tickets, uc.Membership, in.AttemptID, in.ActorID, in.Admin)
	if err != nil {
		return models.Attempt{}, err
	}
	now := time.Now().UTC()
	if attempt.IsExpired(now) {
		return expireAttempt(ctx, uc.Tickets, uc.Attempts, attempt, now)
	}
	return attempt, nil
}

type ListMyAttempts struct {
	Tickets    repositories.TicketRepository
	Attempts   repositories.AttemptRepository
	Membership repositories.GroupMembership
}

type ListMyAttemptsInput struct {
	ActorID  uuid.UUID
	Admin    bool
	TicketID uuid.UUID
}

func (uc ListMyAttempts) Execute(ctx context.Context, in ListMyAttemptsInput) ([]models.Attempt, error) {
	ticket, err := uc.Tickets.FindByID(ctx, in.TicketID)
	if err != nil {
		return nil, err
	}
	if err := requireViewOwn(ctx, uc.Membership, ticket.GroupID, in.ActorID, in.Admin); err != nil {
		return nil, err
	}
	attempts, err := uc.Attempts.ListByTicketUser(ctx, in.TicketID, in.ActorID)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	out := make([]models.Attempt, 0, len(attempts))
	for _, a := range attempts {
		if a.IsExpired(now) {
			expired, err := expireAttempt(ctx, uc.Tickets, uc.Attempts, a, now)
			if err != nil {
				return nil, err
			}
			out = append(out, expired)
			continue
		}
		out = append(out, a)
	}
	return out, nil
}

func expireAttempt(
	ctx context.Context,
	tickets repositories.TicketRepository,
	attempts repositories.AttemptRepository,
	attempt models.Attempt,
	now time.Time,
) (models.Attempt, error) {
	ref, err := tickets.FindReference(ctx, attempt.TicketID)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) {
			return models.Attempt{}, errs.ErrNoReferenceAnswer
		}
		return models.Attempt{}, err
	}
	if err := attempt.Expire(now, ref, scoring.Score); err != nil {
		return models.Attempt{}, err
	}
	if err := attempts.Save(ctx, attempt); err != nil {
		return models.Attempt{}, err
	}
	return attempt, nil
}

func validateAnswer(ctx context.Context, catalog repositories.CatalogRepository, answer models.Answer) error {
	if answer.IncidentTypeID != nil {
		if _, err := catalog.FindIncidentTypeByID(ctx, *answer.IncidentTypeID); err != nil {
			return err
		}
		groups, err := catalog.ListTagGroupsByType(ctx, *answer.IncidentTypeID)
		if err != nil {
			return err
		}
		if err := answer.ValidateTagSelection(groups); err != nil {
			return err
		}
	} else if len(answer.TagIDs) > 0 {
		return errs.ErrInvalidTags
	}
	if len(answer.ServiceIDs) > 0 {
		ok, err := catalog.ServiceExists(ctx, answer.ServiceIDs)
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
	tickets repositories.TicketRepository,
	membership repositories.GroupMembership,
	attemptID, actorID uuid.UUID,
	admin bool,
) (models.Attempt, models.Ticket, error) {
	attempt, err := attempts.FindByID(ctx, attemptID)
	if err != nil {
		return models.Attempt{}, models.Ticket{}, err
	}
	if attempt.UserID != actorID && !admin {
		return models.Attempt{}, models.Ticket{}, errs.ErrForbidden
	}
	ticket, err := tickets.FindByID(ctx, attempt.TicketID)
	if err != nil {
		return models.Attempt{}, models.Ticket{}, err
	}
	if err := requireViewOwn(ctx, membership, ticket.GroupID, actorID, admin); err != nil {
		return models.Attempt{}, models.Ticket{}, err
	}
	return attempt, ticket, nil
}

func requireStudent(ctx context.Context, membership repositories.GroupMembership, groupID, actorID uuid.UUID, admin bool) error {
	if admin {
		return abilities.StartAttempt("", true)
	}
	role, err := membership.RoleOf(ctx, groupID, actorID)
	if err != nil {
		return err
	}
	return abilities.StartAttempt(role, false)
}

func requireViewOwn(ctx context.Context, membership repositories.GroupMembership, groupID, actorID uuid.UUID, admin bool) error {
	if admin {
		return abilities.ViewOwnAttempt("", true)
	}
	role, err := membership.RoleOf(ctx, groupID, actorID)
	if err != nil {
		return err
	}
	return abilities.ViewOwnAttempt(role, false)
}
