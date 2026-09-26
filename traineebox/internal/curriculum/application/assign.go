package application

import (
	"context"

	"traineebox/internal/curriculum/domain/abilities"
	"traineebox/internal/curriculum/domain/errs"
	"traineebox/internal/curriculum/domain/models"
	"traineebox/internal/curriculum/domain/repositories"
	"traineebox/internal/curriculum/domain/value_objects"

	"github.com/google/uuid"
)

type AssignModule struct {
	Modules     repositories.ModuleRepository
	Lessons     repositories.LessonRepository
	Variants    repositories.VariantRepository
	Assignments repositories.AssignmentRepository
	Groups      repositories.GroupDirectory
	Attempts    repositories.AttemptIssuer
}

type AssignModuleInput struct {
	ActorID uuid.UUID
	Role    value_objects.AccountRole
	UserID  *uuid.UUID
	GroupID *uuid.UUID
	ModuleID uuid.UUID
	Picks   []models.LessonVariantPick
}

func (uc AssignModule) Execute(ctx context.Context, in AssignModuleInput) error {
	if err := abilities.AssignModule(in.Role); err != nil {
		return err
	}
	if _, err := uc.Modules.FindByID(ctx, in.ModuleID); err != nil {
		return err
	}
	if err := uc.validatePicks(ctx, in.ModuleID, in.Picks); err != nil {
		return err
	}
	userIDs, sourceGroup, err := uc.resolveUsers(ctx, in)
	if err != nil {
		return err
	}
	for _, userID := range userIDs {
		asg := models.NewUserModule(userID, in.ModuleID, in.ActorID, sourceGroup)
		if err := uc.Assignments.Upsert(ctx, asg); err != nil {
			return err
		}
		for _, pick := range in.Picks {
			if err := uc.Attempts.IssueAvailable(ctx, userID, pick.VariantID, in.ActorID); err != nil {
				return err
			}
		}
	}
	return nil
}

func (uc AssignModule) validatePicks(ctx context.Context, moduleID uuid.UUID, picks []models.LessonVariantPick) error {
	if len(picks) == 0 {
		return errs.ErrInvalidInput
	}
	seen := make(map[uuid.UUID]struct{}, len(picks))
	for _, p := range picks {
		if p.LessonID == uuid.Nil || p.VariantID == uuid.Nil {
			return errs.ErrInvalidInput
		}
		if _, ok := seen[p.LessonID]; ok {
			return errs.ErrInvalidInput
		}
		seen[p.LessonID] = struct{}{}
		lesson, err := uc.Lessons.FindByID(ctx, p.LessonID)
		if err != nil {
			return err
		}
		if lesson.ModuleID != moduleID {
			return errs.ErrInvalidInput
		}
		variant, err := uc.Variants.FindByID(ctx, p.VariantID)
		if err != nil {
			return err
		}
		if variant.LessonID != p.LessonID {
			return errs.ErrInvalidInput
		}
	}
	return nil
}

func (uc AssignModule) resolveUsers(ctx context.Context, in AssignModuleInput) ([]uuid.UUID, *uuid.UUID, error) {
	if in.UserID != nil && in.GroupID == nil {
		return []uuid.UUID{*in.UserID}, nil, nil
	}
	if in.GroupID == nil || in.UserID != nil {
		return nil, nil, errs.ErrInvalidInput
	}
	group, err := uc.Groups.FindByID(ctx, *in.GroupID)
	if err != nil {
		return nil, nil, err
	}
	if !in.Role.IsStaff() {
		return nil, nil, errs.ErrForbidden
	}
	if in.Role != value_objects.AccountRoleAdmin {
		ok := false
		for _, m := range group.Members {
			if m.UserID == in.ActorID && (m.Role == "owner" || m.Role == "teacher") {
				ok = true
				break
			}
		}
		if !ok {
			return nil, nil, errs.ErrForbidden
		}
	}
	ids := make([]uuid.UUID, 0)
	for _, m := range group.Members {
		if m.Role == "student" {
			ids = append(ids, m.UserID)
		}
	}
	return ids, in.GroupID, nil
}
