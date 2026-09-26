package application

import (
	"context"

	"traineebox/internal/curriculum/domain/models"
	"traineebox/internal/curriculum/domain/repositories"
	"traineebox/internal/curriculum/domain/value_objects"

	"github.com/google/uuid"
)

type AssignedLesson struct {
	Lesson    models.Lesson
	Variant   models.Variant
	AttemptID *uuid.UUID
	Status    string
}

type AssignedModule struct {
	Module models.Module
	Lessons []AssignedLesson
}

type AttemptView struct {
	ID        uuid.UUID
	VariantID uuid.UUID
	Status    string
}

type AttemptLister interface {
	ListByUser(ctx context.Context, userID uuid.UUID) ([]AttemptView, error)
}

type ListMyModules struct {
	Modules     repositories.ModuleRepository
	Lessons     repositories.LessonRepository
	Variants    repositories.VariantRepository
	Assignments repositories.AssignmentRepository
	Attempts    AttemptLister
}

func (uc ListMyModules) Execute(ctx context.Context, actorID uuid.UUID, role value_objects.AccountRole) ([]AssignedModule, error) {
	_ = role
	asgs, err := uc.Assignments.ListByUser(ctx, actorID)
	if err != nil {
		return nil, err
	}
	attempts, err := uc.Attempts.ListByUser(ctx, actorID)
	if err != nil {
		return nil, err
	}
	latest := latestAttemptByVariant(attempts)
	out := make([]AssignedModule, 0, len(asgs))
	for _, asg := range asgs {
		module, err := uc.Modules.FindByID(ctx, asg.ModuleID)
		if err != nil {
			return nil, err
		}
		lessons, err := uc.Lessons.ListByModule(ctx, module.ID)
		if err != nil {
			return nil, err
		}
		item := AssignedModule{Module: module, Lessons: make([]AssignedLesson, 0, len(lessons))}
		for _, lesson := range lessons {
			variants, err := uc.Variants.ListByLesson(ctx, lesson.ID)
			if err != nil {
				return nil, err
			}
			for _, v := range variants {
				if av, ok := latest[v.ID]; ok {
					id := av.ID
					item.Lessons = append(item.Lessons, AssignedLesson{
						Lesson:    lesson,
						Variant:   v,
						AttemptID: &id,
						Status:    av.Status,
					})
					break
				}
			}
		}
		out = append(out, item)
	}
	return out, nil
}

func latestAttemptByVariant(attempts []AttemptView) map[uuid.UUID]AttemptView {
	out := make(map[uuid.UUID]AttemptView, len(attempts))
	for _, a := range attempts {
		prev, ok := out[a.VariantID]
		if !ok {
			out[a.VariantID] = a
			continue
		}
		if preferAttempt(a, prev) {
			out[a.VariantID] = a
		}
	}
	return out
}

func preferAttempt(a, b AttemptView) bool {
	rank := func(s string) int {
		switch s {
		case "in_progress":
			return 3
		case "available":
			return 2
		default:
			return 1
		}
	}
	return rank(a.Status) >= rank(b.Status)
}
