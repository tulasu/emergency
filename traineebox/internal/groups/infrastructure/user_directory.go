package infrastructure

import (
	"context"
	"errors"

	"traineebox/internal/groups/domain/errs"
	"traineebox/internal/groups/domain/value_objects"
	"traineebox/internal/groups/infrastructure/groupssql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserDirectory struct {
	q *groupssql.Queries
}

func NewUserDirectory(pool *pgxpool.Pool) *UserDirectory {
	return &UserDirectory{q: groupssql.New(pool)}
}

func (d *UserDirectory) AccountOf(ctx context.Context, id uuid.UUID) (value_objects.AccountRole, bool, error) {
	row, err := d.q.GetAccountByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return "", false, errs.ErrNotFound
		}
		return "", false, err
	}
	role, err := value_objects.ParseAccountRole(row.Role)
	if err != nil {
		return "", false, err
	}
	return role, row.BlockedAt != nil, nil
}
