package application

import (
	"context"
	"time"

	"traineebox/internal/tickets/domain/abilities"
	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/models"
	"traineebox/internal/tickets/domain/repositories"
	"traineebox/internal/tickets/domain/value_objects"

	"github.com/google/uuid"
)

type CreateLibraryTicket struct {
	Tickets    repositories.TicketRepository
	Curriculum repositories.TopicExists
}

type CreateLibraryTicketInput struct {
	ActorID uuid.UUID
	Role    value_objects.AccountRole
	TopicID uuid.UUID
	Title   string
	Body    string
}

func (uc CreateLibraryTicket) Execute(ctx context.Context, in CreateLibraryTicketInput) (models.Ticket, error) {
	if err := abilities.ManageTicket(in.Role); err != nil {
		return models.Ticket{}, err
	}
	if err := uc.Curriculum.TopicExists(ctx, in.TopicID); err != nil {
		return models.Ticket{}, err
	}
	title, err := value_objects.NewTicketTitle(in.Title)
	if err != nil {
		return models.Ticket{}, err
	}
	ticket := models.NewLibraryTicket(in.TopicID, title, in.Body, in.ActorID)
	if err := uc.Tickets.Create(ctx, ticket); err != nil {
		return models.Ticket{}, err
	}
	return ticket, nil
}

type ListLibraryTickets struct {
	Tickets repositories.TicketRepository
}

func (uc ListLibraryTickets) Execute(ctx context.Context, role value_objects.AccountRole, q string) ([]models.Ticket, error) {
	if err := abilities.ManageTicket(role); err != nil {
		return nil, err
	}
	return uc.Tickets.ListLibrary(ctx, q)
}

type CopyTicketFromPool struct {
	Tickets    repositories.TicketRepository
	Curriculum repositories.TopicExists
}

type CopyTicketFromPoolInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	VariantID uuid.UUID
	TicketID  uuid.UUID
}

func (uc CopyTicketFromPool) Execute(ctx context.Context, in CopyTicketFromPoolInput) (models.Ticket, error) {
	if err := abilities.ManageTicket(in.Role); err != nil {
		return models.Ticket{}, err
	}
	if err := uc.Curriculum.VariantExists(ctx, in.VariantID); err != nil {
		return models.Ticket{}, err
	}
	src, err := uc.Tickets.FindByID(ctx, in.TicketID)
	if err != nil {
		return models.Ticket{}, err
	}
	copy := models.NewTicket(in.VariantID, src.TopicID, src.Title, src.Body, in.ActorID)
	copy.ScenarioJSON = src.ScenarioJSON
	copy.ScenarioVersion = src.ScenarioVersion
	copy.Mode = src.Mode
	copy.Briefing = src.Briefing
	if err := uc.Tickets.Create(ctx, copy); err != nil {
		return models.Ticket{}, err
	}
	if ref, err := uc.Tickets.FindReference(ctx, src.ID); err == nil {
		ref.TicketID = copy.ID
		_ = uc.Tickets.SaveReference(ctx, ref)
	}
	return copy, nil
}

type OpenVariant struct {
	Grant   GrantAttempt
	Groups  GroupUserLister
	Modules ModuleAssigneeLister
}

type GroupUserLister interface {
	UserIDsByGroup(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error)
}

type ModuleAssigneeLister interface {
	UserIDsByModule(ctx context.Context, moduleID uuid.UUID) ([]uuid.UUID, error)
}

type OpenVariantInput struct {
	ActorID       uuid.UUID
	Role          value_objects.AccountRole
	VariantID     uuid.UUID
	Mode          string
	ModuleID      *uuid.UUID
	GroupIDs      []uuid.UUID
	UserIDs       []uuid.UUID
	AvailableFrom *time.Time
	DeadlineAt    *time.Time
}

func (uc OpenVariant) Execute(ctx context.Context, in OpenVariantInput) (int, error) {
	if err := abilities.GrantAttempt(in.Role); err != nil {
		return 0, err
	}
	var userIDs []uuid.UUID
	switch in.Mode {
	case "users":
		userIDs = in.UserIDs
	case "groups":
		seen := map[uuid.UUID]struct{}{}
		for _, gid := range in.GroupIDs {
			ids, err := uc.Groups.UserIDsByGroup(ctx, gid)
			if err != nil {
				return 0, err
			}
			for _, id := range ids {
				if _, ok := seen[id]; !ok {
					seen[id] = struct{}{}
					userIDs = append(userIDs, id)
				}
			}
		}
	case "all":
		if in.ModuleID == nil {
			return 0, errs.ErrInvalidInput
		}
		ids, err := uc.Modules.UserIDsByModule(ctx, *in.ModuleID)
		if err != nil {
			return 0, err
		}
		userIDs = ids
	default:
		return 0, errs.ErrInvalidInput
	}
	n := 0
	for _, uid := range userIDs {
		_, err := uc.Grant.Execute(ctx, GrantAttemptInput{
			ActorID: in.ActorID, Role: in.Role, UserID: uid, VariantID: in.VariantID,
			AvailableFrom: in.AvailableFrom, DeadlineAt: in.DeadlineAt,
		})
		if err != nil {
			return n, err
		}
		n++
	}
	return n, nil
}
