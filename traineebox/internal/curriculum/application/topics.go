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

type CreateTopic struct {
	Topics repositories.TopicRepository
}

type CreateTopicInput struct {
	ActorID uuid.UUID
	Role    value_objects.AccountRole
	Title   string
}

func (uc CreateTopic) Execute(ctx context.Context, in CreateTopicInput) (models.Topic, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Topic{}, err
	}
	title, err := value_objects.NewTitle(in.Title)
	if err != nil {
		return models.Topic{}, err
	}
	topic := models.NewTopic(title, in.ActorID)
	if err := uc.Topics.Create(ctx, topic); err != nil {
		return models.Topic{}, err
	}
	return topic, nil
}

type ListTopics struct {
	Topics      repositories.TopicRepository
	Assignments repositories.AssignmentRepository
}

type ListTopicsInput struct {
	ActorID uuid.UUID
	Role    value_objects.AccountRole
}

func (uc ListTopics) Execute(ctx context.Context, in ListTopicsInput) ([]models.Topic, error) {
	if in.Role.IsStaff() {
		return uc.Topics.List(ctx)
	}
	all, err := uc.Topics.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.Topic, 0)
	for _, t := range all {
		ok, err := uc.Assignments.UserHasTopic(ctx, in.ActorID, t.ID)
		if err != nil {
			return nil, err
		}
		if ok {
			out = append(out, t)
		}
	}
	return out, nil
}

type GetTopic struct {
	Topics      repositories.TopicRepository
	Assignments repositories.AssignmentRepository
}

type GetTopicInput struct {
	ActorID uuid.UUID
	Role    value_objects.AccountRole
	TopicID uuid.UUID
}

func (uc GetTopic) Execute(ctx context.Context, in GetTopicInput) (models.Topic, error) {
	topic, err := uc.Topics.FindByID(ctx, in.TopicID)
	if err != nil {
		return models.Topic{}, err
	}
	if in.Role.IsStaff() {
		return topic, nil
	}
	ok, err := uc.Assignments.UserHasTopic(ctx, in.ActorID, in.TopicID)
	if err != nil {
		return models.Topic{}, err
	}
	if !ok {
		return models.Topic{}, errs.ErrForbidden
	}
	return topic, nil
}

type UpdateTopic struct {
	Topics repositories.TopicRepository
}

type UpdateTopicInput struct {
	Role    value_objects.AccountRole
	TopicID uuid.UUID
	Title   string
}

func (uc UpdateTopic) Execute(ctx context.Context, in UpdateTopicInput) (models.Topic, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Topic{}, err
	}
	title, err := value_objects.NewTitle(in.Title)
	if err != nil {
		return models.Topic{}, err
	}
	topic, err := uc.Topics.FindByID(ctx, in.TopicID)
	if err != nil {
		return models.Topic{}, err
	}
	topic.Title = title
	if err := uc.Topics.Update(ctx, topic); err != nil {
		return models.Topic{}, err
	}
	return topic, nil
}

type DeleteTopic struct {
	Topics repositories.TopicRepository
}

func (uc DeleteTopic) Execute(ctx context.Context, role value_objects.AccountRole, id uuid.UUID) error {
	if err := abilities.ManageLibrary(role); err != nil {
		return err
	}
	if _, err := uc.Topics.FindByID(ctx, id); err != nil {
		return err
	}
	return uc.Topics.Delete(ctx, id)
}
