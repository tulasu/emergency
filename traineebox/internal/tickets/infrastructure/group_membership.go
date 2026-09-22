package infrastructure

import (
	"context"
	"errors"

	"traineebox/internal/tickets/domain/errs"
	"traineebox/internal/tickets/domain/value_objects"
	"traineebox/internal/tickets/infrastructure/ticketssql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type GroupMembership struct {
	q *ticketssql.Queries
}

func NewGroupMembership(pool *pgxpool.Pool) *GroupMembership {
	return &GroupMembership{q: ticketssql.New(pool)}
}

func (m *GroupMembership) RoleOf(ctx context.Context, groupID, userID uuid.UUID) (value_objects.MemberRole, error) {
	role, err := m.q.GetMemberRole(ctx, ticketssql.GetMemberRoleParams{
		GroupID: groupID, UserID: userID,
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", errs.ErrForbidden
		}
		return "", err
	}
	return value_objects.ParseMemberRole(role)
}
