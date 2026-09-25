package application

import (
	"context"
	"errors"
	"time"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/models"
	"traineebox/internal/auth/domain/repositories"
	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

const maxProvisionBatch = 200

type StudentEnroller interface {
	EnrollStudent(ctx context.Context, actorID uuid.UUID, admin bool, groupID, userID uuid.UUID) error
}

type ProvisionUsers struct {
	Users  repositories.UserRepository
	Hasher PasswordHasher
	Enroll StudentEnroller
}

type ProvisionRow struct {
	FullName string
	Login    string
}

type ProvisionInput struct {
	ActorID uuid.UUID
	Admin   bool
	GroupID uuid.UUID
	Users   []ProvisionRow
}

type ProvisionedUser struct {
	User     models.User
	Password string
}

type ProvisionFailure struct {
	Index int
	Login string
	Error string
}

type ProvisionResult struct {
	Created []ProvisionedUser
	Failed  []ProvisionFailure
}

func (uc ProvisionUsers) Execute(ctx context.Context, in ProvisionInput) (ProvisionResult, error) {
	if in.GroupID == uuid.Nil {
		return ProvisionResult{}, errs.ErrInvalidInput
	}
	if len(in.Users) == 0 || len(in.Users) > maxProvisionBatch {
		return ProvisionResult{}, errs.ErrInvalidInput
	}

	out := ProvisionResult{
		Created: make([]ProvisionedUser, 0, len(in.Users)),
		Failed:  make([]ProvisionFailure, 0),
	}
	for i, row := range in.Users {
		created, password, err := uc.provisionOne(ctx, in, row)
		if err != nil {
			out.Failed = append(out.Failed, ProvisionFailure{
				Index: i,
				Login: row.Login,
				Error: provisionErrorCode(err),
			})
			continue
		}
		out.Created = append(out.Created, ProvisionedUser{User: created, Password: password})
	}
	return out, nil
}

func (uc ProvisionUsers) provisionOne(ctx context.Context, in ProvisionInput, row ProvisionRow) (models.User, string, error) {
	fullName, err := normalizeFullName(row.FullName, false)
	if err != nil {
		return models.User{}, "", err
	}
	login, err := value_objects.NewLogin(row.Login)
	if err != nil {
		return models.User{}, "", err
	}
	password, err := generatePassword()
	if err != nil {
		return models.User{}, "", err
	}
	hash, err := uc.Hasher.Hash(password)
	if err != nil {
		return models.User{}, "", err
	}
	user := models.User{
		ID:           uuid.New(),
		Login:        login,
		PasswordHash: hash,
		Role:         value_objects.RoleStudent,
		FullName:     fullName,
		CreatedAt:    time.Now().UTC(),
	}
	if err := uc.Users.Create(ctx, user); err != nil {
		return models.User{}, "", err
	}
	if err := uc.Enroll.EnrollStudent(ctx, in.ActorID, in.Admin, in.GroupID, user.ID); err != nil {
		return models.User{}, "", err
	}
	return user, password, nil
}

func provisionErrorCode(err error) string {
	switch {
	case errors.Is(err, errs.ErrConflict):
		return "conflict"
	case errors.Is(err, errs.ErrInvalidInput):
		return "invalid_input"
	case errors.Is(err, errs.ErrForbidden):
		return "forbidden"
	case errors.Is(err, errs.ErrNotFound):
		return "not_found"
	case errors.Is(err, errs.ErrUserBlocked):
		return "user_blocked"
	default:
		return "failed"
	}
}
