package models

import (
	"time"

	"traineebox/internal/curriculum/domain/errs"
	"traineebox/internal/curriculum/domain/value_objects"

	"github.com/google/uuid"
)

type Module struct {
	ID               uuid.UUID
	Title            value_objects.Title
	Description      string
	Status           value_objects.ModuleStatus
	SuccessThreshold int
	CreatedBy        uuid.UUID
	CreatedAt        time.Time
}

func NewModule(title value_objects.Title, description string, createdBy uuid.UUID) Module {
	return Module{
		ID:               uuid.New(),
		Title:            title,
		Description:      description,
		Status:           value_objects.ModuleStatusDraft,
		SuccessThreshold: 70,
		CreatedBy:        createdBy,
		CreatedAt:        time.Now().UTC(),
	}
}

type Lesson struct {
	ID              uuid.UUID
	ModuleID        uuid.UUID
	Title           value_objects.Title
	Position        int
	DurationSeconds *int
	ArchivedAt      *time.Time
	CreatedAt       time.Time
}

func NewLesson(moduleID uuid.UUID, title value_objects.Title, position int, duration *int) (Lesson, error) {
	if duration != nil && *duration <= 0 {
		return Lesson{}, errs.ErrInvalidInput
	}
	return Lesson{
		ID:              uuid.New(),
		ModuleID:        moduleID,
		Title:           title,
		Position:        position,
		DurationSeconds: duration,
		CreatedAt:       time.Now().UTC(),
	}, nil
}

func (l *Lesson) Archive(now time.Time) {
	l.ArchivedAt = &now
}

type Variant struct {
	ID        uuid.UUID
	LessonID  uuid.UUID
	Title     value_objects.Title
	Position  int
	Status    value_objects.VariantStatus
	IsPrimary bool
	CreatedAt time.Time
}

func NewVariant(lessonID uuid.UUID, title value_objects.Title, position int) Variant {
	return Variant{
		ID:        uuid.New(),
		LessonID:  lessonID,
		Title:     title,
		Position:  position,
		Status:    value_objects.VariantStatusDraft,
		IsPrimary: false,
		CreatedAt: time.Now().UTC(),
	}
}

type UserModule struct {
	UserID        uuid.UUID
	ModuleID      uuid.UUID
	AssignedBy    uuid.UUID
	SourceGroupID *uuid.UUID
	AssignedAt    time.Time
}

func NewUserModule(userID, moduleID, assignedBy uuid.UUID, sourceGroupID *uuid.UUID) UserModule {
	return UserModule{
		UserID:        userID,
		ModuleID:      moduleID,
		AssignedBy:    assignedBy,
		SourceGroupID: sourceGroupID,
		AssignedAt:    time.Now().UTC(),
	}
}

type LessonVariantPick struct {
	LessonID  uuid.UUID
	VariantID uuid.UUID
}

type ModuleListItem struct {
	Module         Module
	LessonCount    int
	AssignedCount  int
	AssignedGroups int
	AssignedUsers  int
	AssignedLabel  string
	OpenedDone     int
	OpenedTotal    int
	SuccessRate    *float64
}

type LessonSummary struct {
	Lesson        Lesson
	VariantCount  int
	TicketCount   int
	VariantsLabel string
	OpenedFor     int
	OpenedTotal   int
	PassedRate    *float64
	AvgSuccess    *float64
	Attention     string
}

type VariantSummary struct {
	Variant            Variant
	TicketCount        int
	AttemptCount       int
	AvgSuccess         *float64
	HardestTicketTitle string
	HardestTicketRate  *float64
}

type AssignmentGroupSummary struct {
	Label string
	Count int
}

type AssignmentSummary struct {
	TotalUsers  int
	Groups      []AssignmentGroupSummary
	Individuals int
}

type ModuleSummary struct {
	Module     Module
	Lessons    []LessonSummary
	Assignment AssignmentSummary
	Attention  []string
}
