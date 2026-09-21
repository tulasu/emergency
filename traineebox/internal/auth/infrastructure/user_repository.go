package infrastructure

import (
	"context"
	"errors"
	"time"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/value_objects"
	"traineebox/internal/auth/infrastructure/authsql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {
	q *authsql.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{q: authsql.New(pool)}
}

func (r *UserRepository) Create(ctx context.Context, user models.User) error {
	err := r.q.CreateUser(ctx, authsql.CreateUserParams{
		ID:           user.ID,
		Login:        user.Login.String(),
		PasswordHash: user.PasswordHash.String(),
		Role:         string(user.Role),
		BlockedAt:    user.BlockedAt,
		CreatedAt:    user.CreatedAt,
	})
	if isUniqueViolation(err) {
		return errs.ErrConflict
	}
	return err
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (models.User, error) {
	row, err := r.q.GetUserByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, errs.ErrNotFound
		}
		return models.User{}, err
	}
	return mapUser(row), nil
}

func (r *UserRepository) FindByLogin(ctx context.Context, login value_objects.Login) (models.User, error) {
	row, err := r.q.GetUserByLogin(ctx, login.String())
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.User{}, errs.ErrNotFound
		}
		return models.User{}, err
	}
	return mapUser(row), nil
}

func (r *UserRepository) SetBlocked(ctx context.Context, id uuid.UUID, blocked bool) error {
	var blockedAt *time.Time
	if blocked {
		now := time.Now().UTC()
		blockedAt = &now
	}
	return r.q.SetUserBlocked(ctx, authsql.SetUserBlockedParams{
		ID:        id,
		BlockedAt: blockedAt,
	})
}

func (r *UserRepository) SetRole(ctx context.Context, id uuid.UUID, role value_objects.Role) error {
	return r.q.SetUserRole(ctx, authsql.SetUserRoleParams{
		ID:   id,
		Role: string(role),
	})
}

func mapUser(row authsql.User) models.User {
	return models.User{
		ID:           row.ID,
		Login:        value_objects.Login(row.Login),
		PasswordHash: value_objects.PasswordHash(row.PasswordHash),
		Role:         value_objects.Role(row.Role),
		BlockedAt:    row.BlockedAt,
		CreatedAt:    row.CreatedAt,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
