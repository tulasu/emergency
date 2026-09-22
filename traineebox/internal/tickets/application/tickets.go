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

type CreateTicket struct {
	Tickets    repositories.TicketRepository
	Membership repositories.GroupMembership
}

type CreateTicketInput struct {
	ActorID         uuid.UUID
	Admin           bool
	GroupID         uuid.UUID
	Title           string
	Body            string
	MaxAttempts     *int
	AvailableFrom   *time.Time
	AvailableUntil  *time.Time
	DurationSeconds *int
}

func (uc CreateTicket) Execute(ctx context.Context, in CreateTicketInput) (models.Ticket, error) {
	if err := uc.requireManage(ctx, in.GroupID, in.ActorID, in.Admin); err != nil {
		return models.Ticket{}, err
	}
	title, err := value_objects.NewTicketTitle(in.Title)
	if err != nil {
		return models.Ticket{}, err
	}
	ticket, err := models.NewTicket(
		in.GroupID, title, in.Body, in.ActorID,
		in.MaxAttempts, in.AvailableFrom, in.AvailableUntil, in.DurationSeconds,
	)
	if err != nil {
		return models.Ticket{}, err
	}
	if err := uc.Tickets.Create(ctx, ticket); err != nil {
		return models.Ticket{}, err
	}
	return ticket, nil
}

func (uc CreateTicket) requireManage(ctx context.Context, groupID, actorID uuid.UUID, admin bool) error {
	if admin {
		return abilities.ManageTicket("", true)
	}
	role, err := uc.Membership.RoleOf(ctx, groupID, actorID)
	if err != nil {
		return err
	}
	return abilities.ManageTicket(role, false)
}

type ListTicketsByGroup struct {
	Tickets    repositories.TicketRepository
	Membership repositories.GroupMembership
}

type ListTicketsByGroupInput struct {
	ActorID uuid.UUID
	Admin   bool
	GroupID uuid.UUID
}

func (uc ListTicketsByGroup) Execute(ctx context.Context, in ListTicketsByGroupInput) ([]models.Ticket, error) {
	if err := requireView(ctx, uc.Membership, in.GroupID, in.ActorID, in.Admin); err != nil {
		return nil, err
	}
	return uc.Tickets.ListByGroup(ctx, in.GroupID)
}

type GetTicket struct {
	Tickets    repositories.TicketRepository
	Membership repositories.GroupMembership
}

type GetTicketInput struct {
	ActorID  uuid.UUID
	Admin    bool
	TicketID uuid.UUID
}

func (uc GetTicket) Execute(ctx context.Context, in GetTicketInput) (models.Ticket, error) {
	ticket, err := uc.Tickets.FindByID(ctx, in.TicketID)
	if err != nil {
		return models.Ticket{}, err
	}
	if err := requireView(ctx, uc.Membership, ticket.GroupID, in.ActorID, in.Admin); err != nil {
		return models.Ticket{}, err
	}
	return ticket, nil
}

type SetReferenceAnswer struct {
	Tickets    repositories.TicketRepository
	Catalog    repositories.CatalogRepository
	Membership repositories.GroupMembership
}

type SetReferenceAnswerInput struct {
	ActorID            uuid.UUID
	Admin              bool
	TicketID           uuid.UUID
	IncidentTypeID     uuid.UUID
	TagIDs             []uuid.UUID
	ServiceIDs         []uuid.UUID
	ApplicantLastName  string
	ApplicantFirstName string
	CallerNumber       string
	DictatedNumber     string
}

func (uc SetReferenceAnswer) Execute(ctx context.Context, in SetReferenceAnswerInput) (models.ReferenceAnswer, error) {
	ticket, err := uc.Tickets.FindByID(ctx, in.TicketID)
	if err != nil {
		return models.ReferenceAnswer{}, err
	}
	if in.Admin {
		if err := abilities.ManageTicket("", true); err != nil {
			return models.ReferenceAnswer{}, err
		}
	} else {
		role, err := uc.Membership.RoleOf(ctx, ticket.GroupID, in.ActorID)
		if err != nil {
			return models.ReferenceAnswer{}, err
		}
		if err := abilities.ManageTicket(role, false); err != nil {
			return models.ReferenceAnswer{}, err
		}
	}
	if _, err := uc.Catalog.FindIncidentTypeByID(ctx, in.IncidentTypeID); err != nil {
		return models.ReferenceAnswer{}, err
	}
	allowed, err := uc.Catalog.FindTagIDsForType(ctx, in.IncidentTypeID)
	if err != nil {
		return models.ReferenceAnswer{}, err
	}
	ref, err := models.NewReferenceAnswer(
		in.TicketID, in.IncidentTypeID, in.TagIDs, in.ServiceIDs,
		in.ApplicantLastName, in.ApplicantFirstName, in.CallerNumber, in.DictatedNumber,
	)
	if err != nil {
		return models.ReferenceAnswer{}, err
	}
	if err := ref.ValidateTagsAgainstType(allowed); err != nil {
		return models.ReferenceAnswer{}, err
	}
	if len(in.ServiceIDs) > 0 {
		ok, err := uc.Catalog.ServiceExists(ctx, in.ServiceIDs)
		if err != nil {
			return models.ReferenceAnswer{}, err
		}
		if !ok {
			return models.ReferenceAnswer{}, errs.ErrInvalidInput
		}
	}
	if err := uc.Tickets.SaveReference(ctx, ref); err != nil {
		return models.ReferenceAnswer{}, err
	}
	return ref, nil
}

func requireView(ctx context.Context, membership repositories.GroupMembership, groupID, actorID uuid.UUID, admin bool) error {
	if admin {
		return abilities.ViewTicket("", true)
	}
	role, err := membership.RoleOf(ctx, groupID, actorID)
	if err != nil {
		return err
	}
	return abilities.ViewTicket(role, false)
}
