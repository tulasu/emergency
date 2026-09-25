package main

import (
	"context"
	"errors"

	authapp "traineebox/internal/auth/application"
	autherrs "traineebox/internal/auth/domain/errs"
	groupsapp "traineebox/internal/groups/application"
	groupserrs "traineebox/internal/groups/domain/errs"

	"github.com/google/uuid"
)

type studentEnroller struct {
	add groupsapp.AddMember
}

func (e studentEnroller) EnrollStudent(ctx context.Context, actorID uuid.UUID, admin bool, groupID, userID uuid.UUID) error {
	_, err := e.add.Execute(ctx, groupsapp.AddMemberInput{
		ActorID: actorID,
		Admin:   admin,
		GroupID: groupID,
		UserID:  userID,
		Role:    "student",
	})
	return mapEnrollError(err)
}

func mapEnrollError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case errors.Is(err, groupserrs.ErrForbidden):
		return autherrs.ErrForbidden
	case errors.Is(err, groupserrs.ErrNotFound):
		return autherrs.ErrNotFound
	case errors.Is(err, groupserrs.ErrConflict):
		return autherrs.ErrConflict
	case errors.Is(err, groupserrs.ErrInvalidInput):
		return autherrs.ErrInvalidInput
	case errors.Is(err, groupserrs.ErrUserBlocked):
		return autherrs.ErrUserBlocked
	default:
		return err
	}
}

var _ authapp.StudentEnroller = studentEnroller{}
