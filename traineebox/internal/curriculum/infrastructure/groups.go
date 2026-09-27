package infrastructure

import (
	"context"
	"errors"

	"traineebox/internal/curriculum/domain/errs"
	"traineebox/internal/curriculum/domain/repositories"
	groupserrs "traineebox/internal/groups/domain/errs"
	groupsrepos "traineebox/internal/groups/domain/repositories"

	"github.com/google/uuid"
)

type GroupsAdapter struct {
	Groups groupsrepos.GroupRepository
}

func (a GroupsAdapter) FindByID(ctx context.Context, id uuid.UUID) (repositories.GroupView, error) {
	g, err := a.Groups.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, groupserrs.ErrNotFound) {
			return repositories.GroupView{}, errs.ErrNotFound
		}
		if errors.Is(err, groupserrs.ErrForbidden) {
			return repositories.GroupView{}, errs.ErrForbidden
		}
		return repositories.GroupView{}, err
	}
	members := make([]repositories.GroupMember, 0, len(g.Members))
	for _, m := range g.Members {
		members = append(members, repositories.GroupMember{UserID: m.UserID, Role: m.Role.String()})
	}
	return repositories.GroupView{ID: g.ID, Members: members}, nil
}

func (a GroupsAdapter) UserIDsByGroup(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	g, err := a.Groups.FindByID(ctx, groupID)
	if err != nil {
		if errors.Is(err, groupserrs.ErrNotFound) {
			return nil, errs.ErrNotFound
		}
		if errors.Is(err, groupserrs.ErrForbidden) {
			return nil, errs.ErrForbidden
		}
		return nil, err
	}
	ids := make([]uuid.UUID, 0, len(g.Members))
	for _, m := range g.Members {
		ids = append(ids, m.UserID)
	}
	return ids, nil
}
