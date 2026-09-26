package infrastructure

import (
	"context"
	"errors"

	currerrs "traineebox/internal/curriculum/domain/errs"
	curriculuminfra "traineebox/internal/curriculum/infrastructure"
	"traineebox/internal/tickets/domain/errs"

	"github.com/google/uuid"
)

type CurriculumLookup struct {
	Lessons  curriculuminfra.Lessons
	Variants curriculuminfra.Variants
	Topics   curriculuminfra.Topics
}

func (c CurriculumLookup) DurationSecondsByVariant(ctx context.Context, variantID uuid.UUID) (*int, error) {
	lesson, err := c.Lessons.FindByVariant(ctx, variantID)
	if err != nil {
		return nil, mapCurriculumErr(err)
	}
	return lesson.DurationSeconds, nil
}

func (c CurriculumLookup) TopicExists(ctx context.Context, topicID uuid.UUID) error {
	_, err := c.Topics.FindByID(ctx, topicID)
	return mapCurriculumErr(err)
}

func (c CurriculumLookup) VariantExists(ctx context.Context, variantID uuid.UUID) error {
	_, err := c.Variants.FindByID(ctx, variantID)
	return mapCurriculumErr(err)
}

func mapCurriculumErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, currerrs.ErrNotFound) {
		return errs.ErrNotFound
	}
	return err
}
