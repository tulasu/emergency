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
	Curriculum repositories.TopicExists
}

type CreateTicketInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	VariantID uuid.UUID
	TopicID   uuid.UUID
	Title     string
	Body      string
}

func (uc CreateTicket) Execute(ctx context.Context, in CreateTicketInput) (models.Ticket, error) {
	if err := abilities.ManageTicket(in.Role); err != nil {
		return models.Ticket{}, err
	}
	if err := uc.Curriculum.VariantExists(ctx, in.VariantID); err != nil {
		return models.Ticket{}, err
	}
	if err := uc.Curriculum.TopicExists(ctx, in.TopicID); err != nil {
		return models.Ticket{}, err
	}
	title, err := value_objects.NewTicketTitle(in.Title)
	if err != nil {
		return models.Ticket{}, err
	}
	ticket := models.NewTicket(in.VariantID, in.TopicID, title, in.Body, in.ActorID)
	if err := uc.Tickets.Create(ctx, ticket); err != nil {
		return models.Ticket{}, err
	}
	return ticket, nil
}

type ListTicketsByVariant struct {
	Tickets  repositories.TicketRepository
	Attempts repositories.AttemptRepository
}

type ListTicketsByVariantInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	VariantID uuid.UUID
}

func (uc ListTicketsByVariant) Execute(ctx context.Context, in ListTicketsByVariantInput) ([]models.Ticket, error) {
	if err := abilities.ViewTicket(in.Role); err != nil {
		return nil, err
	}
	if in.Role == value_objects.AccountRoleStudent {
		ok, err := uc.Attempts.HasAny(ctx, in.VariantID, in.ActorID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errs.ErrForbidden
		}
	}
	return uc.Tickets.ListByVariant(ctx, in.VariantID)
}

type GetTicket struct {
	Tickets  repositories.TicketRepository
	Attempts repositories.AttemptRepository
}

type GetTicketInput struct {
	ActorID  uuid.UUID
	Role     value_objects.AccountRole
	TicketID uuid.UUID
}

func (uc GetTicket) Execute(ctx context.Context, in GetTicketInput) (models.Ticket, error) {
	ticket, err := uc.Tickets.FindByID(ctx, in.TicketID)
	if err != nil {
		return models.Ticket{}, err
	}
	if err := abilities.ViewTicket(in.Role); err != nil {
		return models.Ticket{}, err
	}
	if in.Role == value_objects.AccountRoleStudent {
		if ticket.VariantID == nil {
			return models.Ticket{}, errs.ErrForbidden
		}
		ok, err := uc.Attempts.HasAny(ctx, *ticket.VariantID, in.ActorID)
		if err != nil {
			return models.Ticket{}, err
		}
		if !ok {
			return models.Ticket{}, errs.ErrForbidden
		}
	}
	return ticket, nil
}

type UpdateTicket struct {
	Tickets    repositories.TicketRepository
	Curriculum repositories.TopicExists
}

type UpdateTicketInput struct {
	Role     value_objects.AccountRole
	TicketID uuid.UUID
	TopicID  uuid.UUID
	Title    string
	Body     string
}

func (uc UpdateTicket) Execute(ctx context.Context, in UpdateTicketInput) (models.Ticket, error) {
	if err := abilities.ManageTicket(in.Role); err != nil {
		return models.Ticket{}, err
	}
	ticket, err := uc.Tickets.FindByID(ctx, in.TicketID)
	if err != nil {
		return models.Ticket{}, err
	}
	if err := uc.Curriculum.TopicExists(ctx, in.TopicID); err != nil {
		return models.Ticket{}, err
	}
	title, err := value_objects.NewTicketTitle(in.Title)
	if err != nil {
		return models.Ticket{}, err
	}
	ticket.TopicID = in.TopicID
	ticket.Title = title
	ticket.Body = in.Body
	if err := uc.Tickets.Update(ctx, ticket); err != nil {
		return models.Ticket{}, err
	}
	return ticket, nil
}

type AudioTicketDeleter interface {
	DeleteTicket(ctx context.Context, ticketID uuid.UUID) error
}

type DeleteTicket struct {
	Tickets repositories.TicketRepository
	Audio   AudioTicketDeleter
}

func (uc DeleteTicket) Execute(ctx context.Context, role value_objects.AccountRole, id uuid.UUID) error {
	if err := abilities.ManageTicket(role); err != nil {
		return err
	}
	if _, err := uc.Tickets.FindByID(ctx, id); err != nil {
		return err
	}
	if err := uc.Tickets.Delete(ctx, id); err != nil {
		return err
	}
	if uc.Audio != nil {
		audioCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = uc.Audio.DeleteTicket(audioCtx, id)
	}
	return nil
}

type SetReferenceAnswer struct {
	Tickets repositories.TicketRepository
	Catalog repositories.CatalogRepository
}

type SetReferenceAnswerInput struct {
	Role               value_objects.AccountRole
	TicketID           uuid.UUID
	IncidentTypeCode   string
	TagCodes           []string
	ServiceCodes       []string
	ApplicantLastName  string
	ApplicantFirstName string
	CallerNumber       string
	DictatedNumber     string
}

func (uc SetReferenceAnswer) Execute(ctx context.Context, in SetReferenceAnswerInput) (models.ReferenceAnswer, error) {
	if err := abilities.ManageTicket(in.Role); err != nil {
		return models.ReferenceAnswer{}, err
	}
	if _, err := uc.Tickets.FindByID(ctx, in.TicketID); err != nil {
		return models.ReferenceAnswer{}, err
	}
	if _, err := uc.Catalog.FindIncidentTypeByCode(ctx, in.IncidentTypeCode); err != nil {
		return models.ReferenceAnswer{}, err
	}
	groups, err := uc.Catalog.ListTagGroupsByType(ctx, in.IncidentTypeCode)
	if err != nil {
		return models.ReferenceAnswer{}, err
	}
	ref, err := models.NewReferenceAnswer(
		in.TicketID, in.IncidentTypeCode, in.TagCodes, in.ServiceCodes,
		in.ApplicantLastName, in.ApplicantFirstName, in.CallerNumber, in.DictatedNumber,
	)
	if err != nil {
		return models.ReferenceAnswer{}, err
	}
	if err := ref.ValidateTagSelection(groups); err != nil {
		return models.ReferenceAnswer{}, err
	}
	if len(in.ServiceCodes) > 0 {
		ok, err := uc.Catalog.ServiceExists(ctx, in.ServiceCodes)
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
