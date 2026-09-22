package repositories

import (
	"context"

	"traineebox/internal/groups/domain/models"
	"traineebox/internal/groups/domain/value_objects"

	"github.com/google/uuid"
)

type GroupRepository interface {
	Create(ctx context.Context, group models.Group) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Group, error)
	ListAll(ctx context.Context) ([]models.Group, error)
	ListByMember(ctx context.Context, userID uuid.UUID) ([]models.Group, error)
	UpdateName(ctx context.Context, id uuid.UUID, name value_objects.GroupName) error
	Delete(ctx context.Context, id uuid.UUID) error
	AddMember(ctx context.Context, groupID uuid.UUID, member models.Member) error
	RemoveMember(ctx context.Context, groupID, userID uuid.UUID) error
}

type UserDirectory interface {
	AccountOf(ctx context.Context, id uuid.UUID) (role value_objects.AccountRole, blocked bool, err error)
}
