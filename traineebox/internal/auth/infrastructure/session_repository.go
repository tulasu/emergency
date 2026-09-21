package infrastructure

import (
	"context"
	"errors"
	"time"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/infrastructure/authsql"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type SessionRepository struct {
	q *authsql.Queries
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{q: authsql.New(pool)}
}

func (r *SessionRepository) Create(ctx context.Context, session models.Session) error {
	return r.q.CreateSession(ctx, authsql.CreateSessionParams{
		ID:        session.ID,
		UserID:    session.UserID,
		TokenHash: session.TokenHash,
		ExpiresAt: session.ExpiresAt,
		CreatedAt: session.CreatedAt,
	})
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (models.Session, error) {
	row, err := r.q.GetSessionByTokenHash(ctx, tokenHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return models.Session{}, errs.ErrNotFound
		}
		return models.Session{}, err
	}
	return models.Session{
		ID:        row.ID,
		UserID:    row.UserID,
		TokenHash: row.TokenHash,
		ExpiresAt: row.ExpiresAt,
		CreatedAt: row.CreatedAt,
	}, nil
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	return r.q.DeleteSessionByTokenHash(ctx, tokenHash)
}

func (r *SessionRepository) DeleteExpired(ctx context.Context, now time.Time) error {
	return r.q.DeleteExpiredSessions(ctx, now)
}
