package main

import (
	"context"

	ticketsinfra "traineebox/internal/tickets/infrastructure"
	ticketspresentation "traineebox/internal/tickets/presentation"

	"github.com/google/uuid"
)

type ticketExtrasBridge struct {
	repo *ticketsinfra.TicketRepository
}

func (b ticketExtrasBridge) LibraryExtras(ctx context.Context, ticketID uuid.UUID) (ticketspresentation.LibraryTicketExtras, error) {
	e, err := b.repo.LibraryExtras(ctx, ticketID)
	if err != nil {
		return ticketspresentation.LibraryTicketExtras{}, err
	}
	return ticketspresentation.LibraryTicketExtras{
		IncidentTypeCode: e.IncidentTypeCode,
		IncidentType:     e.IncidentType,
		SlotsTotal:       e.SlotsTotal,
		SlotsRequired:    e.SlotsRequired,
		VariantUsage:     e.VariantUsage,
	}, nil
}
