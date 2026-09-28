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
	pool *pgxpool.Pool
	q    *authsql.Queries
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool, q: authsql.New(pool)}
}

func (r *UserRepository) Create(ctx context.Context, user models.User) error {
	err := r.q.CreateUser(ctx, authsql.CreateUserParams{
		ID:           user.ID,
		Login:        user.Login.String(),
		PasswordHash: user.PasswordHash.String(),
		Role:         string(user.Role),
		FullName:     user.FullName,
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

func (r *UserRepository) List(ctx context.Context) ([]models.User, error) {
	rows, err := r.pool.Query(ctx, `
		SELECT id, login, password_hash, role, full_name, blocked_at, created_at
		FROM users
		ORDER BY full_name, login`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.User
	for rows.Next() {
		var row authsql.User
		if err := rows.Scan(&row.ID, &row.Login, &row.PasswordHash, &row.Role, &row.FullName, &row.BlockedAt, &row.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, mapUser(row))
	}
	return out, rows.Err()
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

func (r *UserRepository) SetPasswordHash(ctx context.Context, id uuid.UUID, hash value_objects.PasswordHash) error {
	_, err := r.pool.Exec(ctx, `UPDATE users SET password_hash = $2 WHERE id = $1`, id, hash.String())
	return err
}

func mapUser(row authsql.User) models.User {
	return models.User{
		ID:           row.ID,
		Login:        value_objects.Login(row.Login),
		PasswordHash: value_objects.PasswordHash(row.PasswordHash),
		Role:         value_objects.Role(row.Role),
		FullName:     row.FullName,
		BlockedAt:    row.BlockedAt,
		CreatedAt:    row.CreatedAt,
	}
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
