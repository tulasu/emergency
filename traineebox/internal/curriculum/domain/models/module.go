package models

import (
	"time"

	"traineebox/internal/curriculum/domain/errs"
	"traineebox/internal/curriculum/domain/value_objects"

	"github.com/google/uuid"
)

type Module struct {
	ID          uuid.UUID
	Title       value_objects.Title
	Description string
	CreatedBy   uuid.UUID
	CreatedAt   time.Time
}

func NewModule(title value_objects.Title, description string, createdBy uuid.UUID) Module {
	return Module{
		ID:          uuid.New(),
		Title:       title,
		Description: description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now().UTC(),
	}
}

type Lesson struct {
	ID              uuid.UUID
	ModuleID        uuid.UUID
	Title           value_objects.Title
	Position        int
	DurationSeconds *int
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

type Variant struct {
	ID        uuid.UUID
	LessonID  uuid.UUID
	Title     value_objects.Title
	Position  int
	CreatedAt time.Time
}

func NewVariant(lessonID uuid.UUID, title value_objects.Title, position int) Variant {
	return Variant{
		ID:        uuid.New(),
		LessonID:  lessonID,
		Title:     title,
		Position:  position,
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
