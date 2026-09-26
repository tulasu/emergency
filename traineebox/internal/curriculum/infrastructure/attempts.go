package infrastructure

import (
	"context"

	"traineebox/internal/curriculum/application"
	ticketsrepos "traineebox/internal/tickets/domain/repositories"

	"github.com/google/uuid"
)

type AttemptViews struct {
	Attempts ticketsrepos.AttemptRepository
}

func (a AttemptViews) ListByUser(ctx context.Context, userID uuid.UUID) ([]application.AttemptView, error) {
	items, err := a.Attempts.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]application.AttemptView, 0, len(items))
	for _, it := range items {
		out = append(out, application.AttemptView{ID: it.ID, VariantID: it.VariantID, Status: it.Status.String()})
	}
	return out, nil
}
