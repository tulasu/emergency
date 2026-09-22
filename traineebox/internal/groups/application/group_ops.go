package application

import (
	"context"

	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/models"
	"traineebox/internal/groups/domain/repository"
	"traineebox/internal/groups/domain/value_objects"

	"github.com/google/uuid"
)

type RenameGroup struct {
	Groups repository.GroupRepository
}

type RenameGroupInput struct {
	ActorID uuid.UUID
	Admin   bool
	GroupID uuid.UUID
	Name    string
}

func (uc RenameGroup) Execute(ctx context.Context, in RenameGroupInput) (models.Group, error) {
	name, err := value_objects.NewGroupName(in.Name)
	if err != nil {
		return models.Group{}, err
	}
	group, err := uc.Groups.FindByID(ctx, in.GroupID)
	if err != nil {
		return models.Group{}, err
	}
	if err := group.Rename(in.ActorID, in.Admin, name); err != nil {
		return models.Group{}, err
	}
	if err := uc.Groups.UpdateName(ctx, group.ID, group.Name); err != nil {
		return models.Group{}, err
	}
	return group, nil
}

type DeleteGroup struct {
	Groups repository.GroupRepository
}

type DeleteGroupInput struct {
	ActorID uuid.UUID
	Admin   bool
	GroupID uuid.UUID
}

func (uc DeleteGroup) Execute(ctx context.Context, in DeleteGroupInput) error {
	group, err := uc.Groups.FindByID(ctx, in.GroupID)
	if err != nil {
		return err
	}
	if err := group.Delete(in.ActorID, in.Admin); err != nil {
		return err
	}
	return uc.Groups.Delete(ctx, group.ID)
}

type ListGroups struct {
	Groups repository.GroupRepository
}

type ListGroupsInput struct {
	ActorID uuid.UUID
	Admin   bool
}

func (uc ListGroups) Execute(ctx context.Context, in ListGroupsInput) ([]models.Group, error) {
	if in.Admin {
		return uc.Groups.ListAll(ctx)
	}
	return uc.Groups.ListByMember(ctx, in.ActorID)
}

type GetGroup struct {
	Groups repository.GroupRepository
}

type GetGroupInput struct {
	ActorID uuid.UUID
	Admin   bool
	GroupID uuid.UUID
}

func (uc GetGroup) Execute(ctx context.Context, in GetGroupInput) (models.Group, error) {
	group, err := uc.Groups.FindByID(ctx, in.GroupID)
	if err != nil {
		return models.Group{}, err
	}
	if in.Admin {
		return group, nil
	}
	if _, ok := group.MemberOf(in.ActorID); !ok {
		return models.Group{}, errs.ErrForbidden
	}
	return group, nil
}
