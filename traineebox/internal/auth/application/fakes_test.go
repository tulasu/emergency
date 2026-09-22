package application

import (
	"context"
	"sync"
	"time"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

type fakeUsers struct {
	mu    sync.Mutex
	byID  map[uuid.UUID]models.User
	login map[string]uuid.UUID
}

func newFakeUsers(users ...models.User) *fakeUsers {
	f := &fakeUsers{
		byID:  make(map[uuid.UUID]models.User),
		login: make(map[string]uuid.UUID),
	}
	for _, u := range users {
		f.byID[u.ID] = u
		f.login[u.Login.String()] = u.ID
	}
	return f
}

func (f *fakeUsers) Create(_ context.Context, user models.User) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.login[user.Login.String()]; ok {
		return errs.ErrConflict
	}
	f.byID[user.ID] = user
	f.login[user.Login.String()] = user.ID
	return nil
}

func (f *fakeUsers) FindByID(_ context.Context, id uuid.UUID) (models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return models.User{}, errs.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) FindByLogin(_ context.Context, login value_objects.Login) (models.User, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	id, ok := f.login[login.String()]
	if !ok {
		return models.User{}, errs.ErrNotFound
	}
	return f.byID[id], nil
}

func (f *fakeUsers) SetBlocked(_ context.Context, id uuid.UUID, blocked bool) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return errs.ErrNotFound
	}
	if blocked {
		now := time.Now().UTC()
		u.BlockedAt = &now
	} else {
		u.BlockedAt = nil
	}
	f.byID[id] = u
	return nil
}

func (f *fakeUsers) SetRole(_ context.Context, id uuid.UUID, role value_objects.Role) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	u, ok := f.byID[id]
	if !ok {
		return errs.ErrNotFound
	}
	u.Role = role
	f.byID[id] = u
	return nil
}

type fakeSessions struct {
	mu       sync.Mutex
	byHash   map[string]models.Session
	deleted  []string
	created  []models.Session
}

func newFakeSessions(sessions ...models.Session) *fakeSessions {
	f := &fakeSessions{byHash: make(map[string]models.Session)}
	for _, s := range sessions {
		f.byHash[s.TokenHash] = s
	}
	return f
}

func (f *fakeSessions) Create(_ context.Context, session models.Session) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if _, ok := f.byHash[session.TokenHash]; ok {
		return errs.ErrConflict
	}
	f.byHash[session.TokenHash] = session
	f.created = append(f.created, session)
	return nil
}

func (f *fakeSessions) FindByTokenHash(_ context.Context, tokenHash string) (models.Session, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	s, ok := f.byHash[tokenHash]
	if !ok {
		return models.Session{}, errs.ErrNotFound
	}
	return s, nil
}

func (f *fakeSessions) DeleteByTokenHash(_ context.Context, tokenHash string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.byHash, tokenHash)
	f.deleted = append(f.deleted, tokenHash)
	return nil
}

func (f *fakeSessions) DeleteExpired(_ context.Context, now time.Time) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	for hash, s := range f.byHash {
		if !s.ExpiresAt.After(now) {
			delete(f.byHash, hash)
			f.deleted = append(f.deleted, hash)
		}
	}
	return nil
}
