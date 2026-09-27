package application

import (
	"context"
	"fmt"
	"strings"
	"time"

	"traineebox/internal/curriculum/domain/abilities"
	"traineebox/internal/curriculum/domain/errs"
	"traineebox/internal/curriculum/domain/models"
	"traineebox/internal/curriculum/domain/repositories"
	"traineebox/internal/curriculum/domain/value_objects"

	"github.com/google/uuid"
)

type ListModulesInput struct {
	ActorID uuid.UUID
	Role    value_objects.AccountRole
	Q       string
	Scope   string
}

func (uc ListModules) ExecuteFiltered(ctx context.Context, in ListModulesInput) ([]models.ModuleListItem, error) {
	var items []models.Module
	var err error
	if in.Role.IsStaff() {
		items, err = uc.Modules.ListFiltered(ctx, repositories.ModuleListFilter{
			Q: in.Q, Scope: in.Scope, ActorID: in.ActorID,
		})
	} else {
		items, err = uc.Execute(ctx, in.ActorID, in.Role)
	}
	if err != nil {
		return nil, err
	}
	out := make([]models.ModuleListItem, 0, len(items))
	for _, m := range items {
		n, err := uc.Modules.CountLessons(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		assigned, err := uc.Modules.CountAssignedUsers(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		groups, users, err := uc.Modules.CountAssignmentBreakdown(ctx, m.ID)
		if err != nil {
			return nil, err
		}
		label := "не назначен"
		if groups > 0 || users > 0 {
			parts := make([]string, 0, 2)
			if groups > 0 {
				parts = append(parts, fmt.Sprintf("%d группы", groups))
			}
			if users > 0 {
				parts = append(parts, fmt.Sprintf("%d ученика", users))
			}
			label = strings.Join(parts, " · ")
		}
		out = append(out, models.ModuleListItem{
			Module:         m,
			LessonCount:    n,
			AssignedCount:  assigned,
			AssignedGroups: groups,
			AssignedUsers:  users,
			AssignedLabel:  label,
			OpenedDone:     0,
			OpenedTotal:    n,
		})
	}
	return out, nil
}

type UpdateModuleInputExtended struct {
	Role             value_objects.AccountRole
	ModuleID         uuid.UUID
	Title            string
	Description      string
	Status           string
	SuccessThreshold *int
}

func (uc UpdateModule) ExecuteExtended(ctx context.Context, in UpdateModuleInputExtended) (models.Module, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Module{}, err
	}
	module, err := uc.Modules.FindByID(ctx, in.ModuleID)
	if err != nil {
		return models.Module{}, err
	}
	title, err := value_objects.NewTitle(in.Title)
	if err != nil {
		return models.Module{}, err
	}
	module.Title = title
	module.Description = in.Description
	if in.Status != "" {
		st, err := value_objects.NewModuleStatus(in.Status)
		if err != nil {
			return models.Module{}, err
		}
		module.Status = st
	}
	if in.SuccessThreshold != nil {
		if *in.SuccessThreshold < 0 || *in.SuccessThreshold > 100 {
			return models.Module{}, errs.ErrInvalidInput
		}
		module.SuccessThreshold = *in.SuccessThreshold
	}
	if err := uc.Modules.Update(ctx, module); err != nil {
		return models.Module{}, err
	}
	return module, nil
}

type GetModuleSummary struct {
	Modules  repositories.ModuleRepository
	Lessons  repositories.LessonRepository
	Variants repositories.VariantRepository
}

func (uc GetModuleSummary) Execute(ctx context.Context, role value_objects.AccountRole, moduleID uuid.UUID) (models.ModuleSummary, error) {
	if err := abilities.ManageLibrary(role); err != nil {
		return models.ModuleSummary{}, err
	}
	module, err := uc.Modules.FindByID(ctx, moduleID)
	if err != nil {
		return models.ModuleSummary{}, err
	}
	lessons, err := uc.Lessons.ListByModule(ctx, moduleID)
	if err != nil {
		return models.ModuleSummary{}, err
	}
	assigned, err := uc.Modules.CountAssignedUsers(ctx, moduleID)
	if err != nil {
		return models.ModuleSummary{}, err
	}
	out := models.ModuleSummary{
		Module:  module,
		Lessons: make([]models.LessonSummary, 0, len(lessons)),
		Assignment: models.AssignmentSummary{
			TotalUsers:  assigned,
			Individuals: assigned,
			Groups:      []models.AssignmentGroupSummary{},
		},
		Attention: []string{},
	}
	for _, l := range lessons {
		vc, err := uc.Lessons.CountVariants(ctx, l.ID)
		if err != nil {
			return models.ModuleSummary{}, err
		}
		tc, err := uc.Lessons.CountTickets(ctx, l.ID)
		if err != nil {
			return models.ModuleSummary{}, err
		}
		vars, err := uc.Variants.ListByLesson(ctx, l.ID)
		if err != nil {
			return models.ModuleSummary{}, err
		}
		labels := make([]string, 0, len(vars))
		hasApproved := false
		for _, v := range vars {
			star := ""
			if v.IsPrimary {
				star = "★"
			}
			draft := ""
			if v.Status.String() == "draft" {
				draft = " черн."
			} else {
				hasApproved = true
			}
			labels = append(labels, strings.TrimSpace(v.Title.String()+star+draft))
		}
		attention := ""
		if !hasApproved {
			attention = "нет утверждённого варианта"
			out.Attention = append(out.Attention, fmt.Sprintf("%s: нет утверждённого варианта", l.Title.String()))
		}
		out.Lessons = append(out.Lessons, models.LessonSummary{
			Lesson:        l,
			VariantCount:  vc,
			TicketCount:   tc,
			VariantsLabel: strings.Join(labels, " · "),
			OpenedFor:     0,
			OpenedTotal:   assigned,
			Attention:     attention,
		})
	}
	return out, nil
}

type ListLessonsPool struct {
	Lessons repositories.LessonRepository
}

func (uc ListLessonsPool) Execute(ctx context.Context, role value_objects.AccountRole, q string) ([]models.Lesson, error) {
	if err := abilities.ManageLibrary(role); err != nil {
		return nil, err
	}
	return uc.Lessons.ListPool(ctx, q)
}

type ArchiveLesson struct {
	Lessons repositories.LessonRepository
}

func (uc ArchiveLesson) Execute(ctx context.Context, role value_objects.AccountRole, lessonID uuid.UUID) (models.Lesson, error) {
	if err := abilities.ManageLibrary(role); err != nil {
		return models.Lesson{}, err
	}
	lesson, err := uc.Lessons.FindByID(ctx, lessonID)
	if err != nil {
		return models.Lesson{}, err
	}
	now := time.Now().UTC()
	lesson.Archive(now)
	if err := uc.Lessons.Update(ctx, lesson); err != nil {
		return models.Lesson{}, err
	}
	return lesson, nil
}

type CopyLessonFromPool struct {
	Modules  repositories.ModuleRepository
	Lessons  repositories.LessonRepository
	Variants repositories.VariantRepository
}

type CopyLessonFromPoolInput struct {
	Role           value_objects.AccountRole
	ModuleID       uuid.UUID
	SourceLessonID uuid.UUID
	Position       int
}

func (uc CopyLessonFromPool) Execute(ctx context.Context, in CopyLessonFromPoolInput) (models.Lesson, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Lesson{}, err
	}
	if _, err := uc.Modules.FindByID(ctx, in.ModuleID); err != nil {
		return models.Lesson{}, err
	}
	src, err := uc.Lessons.FindByID(ctx, in.SourceLessonID)
	if err != nil {
		return models.Lesson{}, err
	}
	lesson, err := models.NewLesson(in.ModuleID, src.Title, in.Position, src.DurationSeconds)
	if err != nil {
		return models.Lesson{}, err
	}
	if err := uc.Lessons.Create(ctx, lesson); err != nil {
		return models.Lesson{}, err
	}
	vars, err := uc.Variants.ListByLesson(ctx, src.ID)
	if err != nil {
		return models.Lesson{}, err
	}
	for _, v := range vars {
		nv := models.NewVariant(lesson.ID, v.Title, v.Position)
		nv.Status = v.Status
		nv.IsPrimary = v.IsPrimary
		if err := uc.Variants.Create(ctx, nv); err != nil {
			return models.Lesson{}, err
		}
	}
	return lesson, nil
}

type CloneVariant struct {
	Variants repositories.VariantRepository
}

type CloneVariantInput struct {
	Role      value_objects.AccountRole
	VariantID uuid.UUID
	Title     string
}

func (uc CloneVariant) Execute(ctx context.Context, in CloneVariantInput) (models.Variant, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Variant{}, err
	}
	src, err := uc.Variants.FindByID(ctx, in.VariantID)
	if err != nil {
		return models.Variant{}, err
	}
	title := src.Title
	if in.Title != "" {
		t, err := value_objects.NewTitle(in.Title)
		if err != nil {
			return models.Variant{}, err
		}
		title = t
	}
	existing, err := uc.Variants.ListByLesson(ctx, src.LessonID)
	if err != nil {
		return models.Variant{}, err
	}
	nv := models.NewVariant(src.LessonID, title, len(existing))
	if err := uc.Variants.Create(ctx, nv); err != nil {
		return models.Variant{}, err
	}
	return nv, nil
}

type UpdateVariantInputExtended struct {
	Role      value_objects.AccountRole
	VariantID uuid.UUID
	Title     string
	Position  int
	Status    string
	IsPrimary *bool
}

func (uc UpdateVariant) ExecuteExtended(ctx context.Context, in UpdateVariantInputExtended) (models.Variant, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Variant{}, err
	}
	variant, err := uc.Variants.FindByID(ctx, in.VariantID)
	if err != nil {
		return models.Variant{}, err
	}
	title, err := value_objects.NewTitle(in.Title)
	if err != nil {
		return models.Variant{}, err
	}
	variant.Title = title
	variant.Position = in.Position
	if in.Status != "" {
		st, err := value_objects.NewVariantStatus(in.Status)
		if err != nil {
			return models.Variant{}, err
		}
		variant.Status = st
	}
	if in.IsPrimary != nil && *in.IsPrimary {
		if err := uc.Variants.ClearPrimary(ctx, variant.LessonID); err != nil {
			return models.Variant{}, err
		}
		variant.IsPrimary = true
	} else if in.IsPrimary != nil {
		variant.IsPrimary = false
	}
	if err := uc.Variants.Update(ctx, variant); err != nil {
		return models.Variant{}, err
	}
	return variant, nil
}
