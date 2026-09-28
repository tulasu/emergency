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

type CreateModule struct {
	Modules repositories.ModuleRepository
}

type CreateModuleInput struct {
	ActorID     uuid.UUID
	Role        value_objects.AccountRole
	Title       string
	Description string
}

func (uc CreateModule) Execute(ctx context.Context, in CreateModuleInput) (models.Module, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Module{}, err
	}
	title, err := value_objects.NewTitle(in.Title)
	if err != nil {
		return models.Module{}, err
	}
	module := models.NewModule(title, in.Description, in.ActorID)
	if err := uc.Modules.Create(ctx, module); err != nil {
		return models.Module{}, err
	}
	return module, nil
}

type ListModules struct {
	Modules     repositories.ModuleRepository
	Assignments repositories.AssignmentRepository
	Metrics     repositories.MetricsReader
}

func (uc ListModules) Execute(ctx context.Context, actorID uuid.UUID, role value_objects.AccountRole) ([]models.Module, error) {
	if role.IsStaff() {
		return uc.Modules.List(ctx)
	}
	asgs, err := uc.Assignments.ListByUser(ctx, actorID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Module, 0, len(asgs))
	for _, a := range asgs {
		m, err := uc.Modules.FindByID(ctx, a.ModuleID)
		if err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, nil
}

type GetModule struct {
	Modules     repositories.ModuleRepository
	Assignments repositories.AssignmentRepository
}

func (uc GetModule) Execute(ctx context.Context, actorID uuid.UUID, role value_objects.AccountRole, id uuid.UUID) (models.Module, error) {
	module, err := uc.Modules.FindByID(ctx, id)
	if err != nil {
		return models.Module{}, err
	}
	if role.IsStaff() {
		return module, nil
	}
	ok, err := uc.Assignments.Has(ctx, actorID, id)
	if err != nil {
		return models.Module{}, err
	}
	if !ok {
		return models.Module{}, errs.ErrForbidden
	}
	return module, nil
}

type UpdateModule struct {
	Modules repositories.ModuleRepository
}

type UpdateModuleInput struct {
	Role        value_objects.AccountRole
	ModuleID    uuid.UUID
	Title       string
	Description string
}

func (uc UpdateModule) Execute(ctx context.Context, in UpdateModuleInput) (models.Module, error) {
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
	if err := uc.Modules.Update(ctx, module); err != nil {
		return models.Module{}, err
	}
	return module, nil
}

type DeleteModule struct {
	Modules repositories.ModuleRepository
}

func (uc DeleteModule) Execute(ctx context.Context, role value_objects.AccountRole, id uuid.UUID) error {
	if err := abilities.ManageLibrary(role); err != nil {
		return err
	}
	if _, err := uc.Modules.FindByID(ctx, id); err != nil {
		return err
	}
	return uc.Modules.Delete(ctx, id)
}

type CreateLesson struct {
	Modules repositories.ModuleRepository
	Lessons repositories.LessonRepository
}

type CreateLessonInput struct {
	Role            value_objects.AccountRole
	ModuleID        uuid.UUID
	Title           string
	Position        int
	DurationSeconds *int
}

func (uc CreateLesson) Execute(ctx context.Context, in CreateLessonInput) (models.Lesson, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Lesson{}, err
	}
	if _, err := uc.Modules.FindByID(ctx, in.ModuleID); err != nil {
		return models.Lesson{}, err
	}
	title, err := value_objects.NewTitle(in.Title)
	if err != nil {
		return models.Lesson{}, err
	}
	lesson, err := models.NewLesson(in.ModuleID, title, in.Position, in.DurationSeconds)
	if err != nil {
		return models.Lesson{}, err
	}
	if err := uc.Lessons.Create(ctx, lesson); err != nil {
		return models.Lesson{}, err
	}
	return lesson, nil
}

type ListLessons struct {
	Modules     repositories.ModuleRepository
	Lessons     repositories.LessonRepository
	Assignments repositories.AssignmentRepository
}

func (uc ListLessons) Execute(ctx context.Context, actorID uuid.UUID, role value_objects.AccountRole, moduleID uuid.UUID) ([]models.Lesson, error) {
	if _, err := uc.Modules.FindByID(ctx, moduleID); err != nil {
		return nil, err
	}
	if !role.IsStaff() {
		ok, err := uc.Assignments.Has(ctx, actorID, moduleID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errs.ErrForbidden
		}
	}
	return uc.Lessons.ListByModule(ctx, moduleID)
}

type UpdateLesson struct {
	Lessons repositories.LessonRepository
}

type UpdateLessonInput struct {
	Role            value_objects.AccountRole
	LessonID        uuid.UUID
	Title           string
	Position        int
	DurationSeconds *int
}

func (uc UpdateLesson) Execute(ctx context.Context, in UpdateLessonInput) (models.Lesson, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Lesson{}, err
	}
	lesson, err := uc.Lessons.FindByID(ctx, in.LessonID)
	if err != nil {
		return models.Lesson{}, err
	}
	title, err := value_objects.NewTitle(in.Title)
	if err != nil {
		return models.Lesson{}, err
	}
	if in.DurationSeconds != nil && *in.DurationSeconds <= 0 {
		return models.Lesson{}, errs.ErrInvalidInput
	}
	lesson.Title = title
	lesson.Position = in.Position
	lesson.DurationSeconds = in.DurationSeconds
	if err := uc.Lessons.Update(ctx, lesson); err != nil {
		return models.Lesson{}, err
	}
	return lesson, nil
}

type DeleteLesson struct {
	Lessons repositories.LessonRepository
}

func (uc DeleteLesson) Execute(ctx context.Context, role value_objects.AccountRole, id uuid.UUID) error {
	if err := abilities.ManageLibrary(role); err != nil {
		return err
	}
	if _, err := uc.Lessons.FindByID(ctx, id); err != nil {
		return err
	}
	return uc.Lessons.Delete(ctx, id)
}

type CreateVariant struct {
	Lessons  repositories.LessonRepository
	Variants repositories.VariantRepository
}

type CreateVariantInput struct {
	Role     value_objects.AccountRole
	LessonID uuid.UUID
	Title    string
	Position int
}

func (uc CreateVariant) Execute(ctx context.Context, in CreateVariantInput) (models.Variant, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Variant{}, err
	}
	if _, err := uc.Lessons.FindByID(ctx, in.LessonID); err != nil {
		return models.Variant{}, err
	}
	title, err := value_objects.NewTitle(in.Title)
	if err != nil {
		return models.Variant{}, err
	}
	variant := models.NewVariant(in.LessonID, title, in.Position)
	if err := uc.Variants.Create(ctx, variant); err != nil {
		return models.Variant{}, err
	}
	return variant, nil
}

type ListVariants struct {
	Lessons     repositories.LessonRepository
	Variants    repositories.VariantRepository
	Assignments repositories.AssignmentRepository
	Metrics     repositories.MetricsReader
	StaffOnly   bool
}

func (uc ListVariants) Execute(ctx context.Context, _ uuid.UUID, role value_objects.AccountRole, lessonID uuid.UUID) ([]models.Variant, error) {
	if err := abilities.ManageLibrary(role); err != nil {
		return nil, err
	}
	if _, err := uc.Lessons.FindByID(ctx, lessonID); err != nil {
		return nil, err
	}
	return uc.Variants.ListByLesson(ctx, lessonID)
}

func (uc ListVariants) ExecuteSummaries(ctx context.Context, actorID uuid.UUID, role value_objects.AccountRole, lessonID uuid.UUID) ([]models.VariantSummary, error) {
	items, err := uc.Execute(ctx, actorID, role, lessonID)
	if err != nil {
		return nil, err
	}
	out := make([]models.VariantSummary, 0, len(items))
	for _, v := range items {
		sum := models.VariantSummary{Variant: v}
		if uc.Metrics != nil {
			m, err := uc.Metrics.VariantMetrics(ctx, v.ID)
			if err != nil {
				return nil, err
			}
			sum.TicketCount = m.TicketCount
			sum.AttemptCount = m.AttemptCount
			sum.AvgSuccess = m.AvgSuccess
			sum.HardestTicketTitle = m.HardestTicketTitle
			sum.HardestTicketRate = m.HardestTicketRate
		}
		out = append(out, sum)
	}
	return out, nil
}

type UpdateVariant struct {
	Variants repositories.VariantRepository
}

type UpdateVariantInput struct {
	Role      value_objects.AccountRole
	VariantID uuid.UUID
	Title     string
	Position  int
}

func (uc UpdateVariant) Execute(ctx context.Context, in UpdateVariantInput) (models.Variant, error) {
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
	if err := uc.Variants.Update(ctx, variant); err != nil {
		return models.Variant{}, err
	}
	return variant, nil
}

type DeleteVariant struct {
	Variants repositories.VariantRepository
}

func (uc DeleteVariant) Execute(ctx context.Context, role value_objects.AccountRole, id uuid.UUID) error {
	if err := abilities.ManageLibrary(role); err != nil {
		return err
	}
	if _, err := uc.Variants.FindByID(ctx, id); err != nil {
		return err
	}
	return uc.Variants.Delete(ctx, id)
}

type GetVariant struct {
	Variants repositories.VariantRepository
}

func (uc GetVariant) Execute(ctx context.Context, role value_objects.AccountRole, id uuid.UUID) (models.Variant, error) {
	if err := abilities.ManageLibrary(role); err != nil {
		return models.Variant{}, err
	}
	return uc.Variants.FindByID(ctx, id)
}
