package application

import (
	"context"

	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/models"
	"traineebox/internal/groups/domain/repository"
	"traineebox/internal/groups/domain/value_objects"

	"github.com/google/uuid"
)

type CreateGroup struct {
	Groups    repository.GroupRepository
	Directory repository.UserDirectory
}

type CreateGroupInput struct {
	ActorID uuid.UUID
	Name    string
}

func (uc CreateGroup) Execute(ctx context.Context, in CreateGroupInput) (models.Group, error) {
	role, blocked, err := uc.Directory.AccountOf(ctx, in.ActorID)
	if err != nil {
		return models.Group{}, err
	}
	if blocked {
		return models.Group{}, errs.ErrUserBlocked
	}
	if role != value_objects.AccountRoleTeacher {
		return models.Group{}, errs.ErrForbidden
	}
	name, err := value_objects.NewGroupName(in.Name)
	if err != nil {
		return models.Group{}, err
	}
	group := models.NewGroup(name, in.ActorID)
	if err := uc.Groups.Create(ctx, group); err != nil {
		return models.Group{}, err
	}
	return group, nil
}
