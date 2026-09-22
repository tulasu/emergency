package application

import (
	"context"

	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/models"
	"traineebox/internal/groups/domain/repositories"
	"traineebox/internal/groups/domain/value_objects"

	"github.com/google/uuid"
)

type AddMember struct {
	Groups    repositories.GroupRepository
	Directory repositories.UserDirectory
}

type AddMemberInput struct {
	ActorID  uuid.UUID
	Admin    bool
	GroupID  uuid.UUID
	UserID   uuid.UUID
	Role     string
}

func (uc AddMember) Execute(ctx context.Context, in AddMemberInput) (models.Group, error) {
	role, err := value_objects.ParseMemberRole(in.Role)
	if err != nil {
		return models.Group{}, err
	}
	if role == value_objects.MemberRoleOwner {
		return models.Group{}, errs.ErrInvalidInput
	}

	account, blocked, err := uc.Directory.AccountOf(ctx, in.UserID)
	if err != nil {
		return models.Group{}, err
	}
	if blocked {
		return models.Group{}, errs.ErrUserBlocked
	}

	group, err := uc.Groups.FindByID(ctx, in.GroupID)
	if err != nil {
		return models.Group{}, err
	}
	if err := group.AddMember(in.ActorID, in.Admin, in.UserID, account, role); err != nil {
		return models.Group{}, err
	}
	member, _ := group.MemberOf(in.UserID)
	if err := uc.Groups.AddMember(ctx, group.ID, member); err != nil {
		return models.Group{}, err
	}
	return group, nil
}

type RemoveMember struct {
	Groups repositories.GroupRepository
}

type RemoveMemberInput struct {
	ActorID uuid.UUID
	Admin   bool
	GroupID uuid.UUID
	UserID  uuid.UUID
}

func (uc RemoveMember) Execute(ctx context.Context, in RemoveMemberInput) (models.Group, error) {
	group, err := uc.Groups.FindByID(ctx, in.GroupID)
	if err != nil {
		return models.Group{}, err
	}
	if err := group.RemoveMember(in.ActorID, in.Admin, in.UserID); err != nil {
		return models.Group{}, err
	}
	if err := uc.Groups.RemoveMember(ctx, group.ID, in.UserID); err != nil {
		return models.Group{}, err
	}
	return group, nil
}
