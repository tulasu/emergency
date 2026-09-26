package infrastructure

import (
	"context"
	"errors"

	"traineebox/internal/curriculum/domain/errs"
	"traineebox/internal/curriculum/domain/models"
	"traineebox/internal/curriculum/domain/value_objects"
	"traineebox/internal/curriculum/infrastructure/curriculumsql"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
	q    *curriculumsql.Queries
}

func NewStore(pool *pgxpool.Pool) *Store {
	return &Store{pool: pool, q: curriculumsql.New(pool)}
}

func (s *Store) CreateTopic(ctx context.Context, topic models.Topic) error {
	return s.q.CreateTopic(ctx, curriculumsql.CreateTopicParams{
		ID: topic.ID, Title: topic.Title.String(), CreatedBy: topic.CreatedBy, CreatedAt: topic.CreatedAt,
	})
}

func (s *Store) FindTopicByID(ctx context.Context, id uuid.UUID) (models.Topic, error) {
	row, err := s.q.GetTopicByID(ctx, id)
	if err != nil {
		return models.Topic{}, mapNotFound(err)
	}
	return models.Topic{ID: row.ID, Title: value_objects.Title(row.Title), CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt}, nil
}

func (s *Store) ListTopics(ctx context.Context) ([]models.Topic, error) {
	rows, err := s.q.ListTopics(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.Topic, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.Topic{ID: row.ID, Title: value_objects.Title(row.Title), CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (s *Store) UpdateTopic(ctx context.Context, topic models.Topic) error {
	return s.q.UpdateTopic(ctx, curriculumsql.UpdateTopicParams{ID: topic.ID, Title: topic.Title.String()})
}

func (s *Store) DeleteTopic(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteTopic(ctx, id)
}

func (s *Store) CreateArticle(ctx context.Context, article models.Article) error {
	return s.q.CreateArticle(ctx, curriculumsql.CreateArticleParams{
		ID: article.ID, TopicID: article.TopicID, Title: article.Title.String(), BodyMd: article.BodyMD,
		CreatedBy: article.CreatedBy, CreatedAt: article.CreatedAt, UpdatedAt: article.UpdatedAt,
	})
}

func (s *Store) FindArticleByID(ctx context.Context, id uuid.UUID) (models.Article, error) {
	row, err := s.q.GetArticleByID(ctx, id)
	if err != nil {
		return models.Article{}, mapNotFound(err)
	}
	return mapArticle(row), nil
}

func (s *Store) ListArticlesByTopic(ctx context.Context, topicID uuid.UUID) ([]models.Article, error) {
	rows, err := s.q.ListArticlesByTopic(ctx, topicID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Article, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapArticle(row))
	}
	return out, nil
}

func (s *Store) UpdateArticle(ctx context.Context, article models.Article) error {
	return s.q.UpdateArticle(ctx, curriculumsql.UpdateArticleParams{
		ID: article.ID, Title: article.Title.String(), BodyMd: article.BodyMD, UpdatedAt: article.UpdatedAt,
	})
}

func (s *Store) DeleteArticle(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteArticle(ctx, id)
}

func (s *Store) CreateAttachment(ctx context.Context, att models.Attachment) error {
	return s.q.CreateAttachment(ctx, curriculumsql.CreateAttachmentParams{
		ID: att.ID, ArticleID: att.ArticleID, Filename: att.Filename, ContentType: att.ContentType,
		SizeBytes: att.SizeBytes, StorageKey: att.StorageKey, CreatedAt: att.CreatedAt,
	})
}

func (s *Store) FindAttachmentByID(ctx context.Context, id uuid.UUID) (models.Attachment, error) {
	row, err := s.q.GetAttachmentByID(ctx, id)
	if err != nil {
		return models.Attachment{}, mapNotFound(err)
	}
	return models.Attachment{
		ID: row.ID, ArticleID: row.ArticleID, Filename: row.Filename, ContentType: row.ContentType,
		SizeBytes: row.SizeBytes, StorageKey: row.StorageKey, CreatedAt: row.CreatedAt,
	}, nil
}

func (s *Store) ListAttachmentsByArticle(ctx context.Context, articleID uuid.UUID) ([]models.Attachment, error) {
	rows, err := s.q.ListAttachmentsByArticle(ctx, articleID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Attachment, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.Attachment{
			ID: row.ID, ArticleID: row.ArticleID, Filename: row.Filename, ContentType: row.ContentType,
			SizeBytes: row.SizeBytes, StorageKey: row.StorageKey, CreatedAt: row.CreatedAt,
		})
	}
	return out, nil
}

func (s *Store) DeleteAttachment(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteAttachment(ctx, id)
}

func (s *Store) CreateModule(ctx context.Context, module models.Module) error {
	return s.q.CreateModule(ctx, curriculumsql.CreateModuleParams{
		ID: module.ID, Title: module.Title.String(), Description: module.Description,
		CreatedBy: module.CreatedBy, CreatedAt: module.CreatedAt,
	})
}

func (s *Store) FindModuleByID(ctx context.Context, id uuid.UUID) (models.Module, error) {
	row, err := s.q.GetModuleByID(ctx, id)
	if err != nil {
		return models.Module{}, mapNotFound(err)
	}
	return models.Module{ID: row.ID, Title: value_objects.Title(row.Title), Description: row.Description, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt}, nil
}

func (s *Store) ListModules(ctx context.Context) ([]models.Module, error) {
	rows, err := s.q.ListModules(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]models.Module, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.Module{ID: row.ID, Title: value_objects.Title(row.Title), Description: row.Description, CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (s *Store) UpdateModule(ctx context.Context, module models.Module) error {
	return s.q.UpdateModule(ctx, curriculumsql.UpdateModuleParams{
		ID: module.ID, Title: module.Title.String(), Description: module.Description,
	})
}

func (s *Store) DeleteModule(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteModule(ctx, id)
}

func (s *Store) CreateLesson(ctx context.Context, lesson models.Lesson) error {
	return s.q.CreateLesson(ctx, curriculumsql.CreateLessonParams{
		ID: lesson.ID, ModuleID: lesson.ModuleID, Title: lesson.Title.String(),
		Position: int32(lesson.Position), DurationSeconds: intPtrToInt32(lesson.DurationSeconds), CreatedAt: lesson.CreatedAt,
	})
}

func (s *Store) FindLessonByID(ctx context.Context, id uuid.UUID) (models.Lesson, error) {
	row, err := s.q.GetLessonByID(ctx, id)
	if err != nil {
		return models.Lesson{}, mapNotFound(err)
	}
	return mapLesson(row), nil
}

func (s *Store) ListLessonsByModule(ctx context.Context, moduleID uuid.UUID) ([]models.Lesson, error) {
	rows, err := s.q.ListLessonsByModule(ctx, moduleID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Lesson, 0, len(rows))
	for _, row := range rows {
		out = append(out, mapLesson(row))
	}
	return out, nil
}

func (s *Store) FindLessonByVariant(ctx context.Context, variantID uuid.UUID) (models.Lesson, error) {
	row, err := s.q.GetLessonByVariant(ctx, variantID)
	if err != nil {
		return models.Lesson{}, mapNotFound(err)
	}
	return mapLesson(row), nil
}

func (s *Store) UpdateLesson(ctx context.Context, lesson models.Lesson) error {
	return s.q.UpdateLesson(ctx, curriculumsql.UpdateLessonParams{
		ID: lesson.ID, Title: lesson.Title.String(), Position: int32(lesson.Position),
		DurationSeconds: intPtrToInt32(lesson.DurationSeconds),
	})
}

func (s *Store) DeleteLesson(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteLesson(ctx, id)
}

func (s *Store) CreateVariant(ctx context.Context, variant models.Variant) error {
	return s.q.CreateVariant(ctx, curriculumsql.CreateVariantParams{
		ID: variant.ID, LessonID: variant.LessonID, Title: variant.Title.String(),
		Position: int32(variant.Position), CreatedAt: variant.CreatedAt,
	})
}

func (s *Store) FindVariantByID(ctx context.Context, id uuid.UUID) (models.Variant, error) {
	row, err := s.q.GetVariantByID(ctx, id)
	if err != nil {
		return models.Variant{}, mapNotFound(err)
	}
	return models.Variant{ID: row.ID, LessonID: row.LessonID, Title: value_objects.Title(row.Title), Position: int(row.Position), CreatedAt: row.CreatedAt}, nil
}

func (s *Store) ListVariantsByLesson(ctx context.Context, lessonID uuid.UUID) ([]models.Variant, error) {
	rows, err := s.q.ListVariantsByLesson(ctx, lessonID)
	if err != nil {
		return nil, err
	}
	out := make([]models.Variant, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.Variant{ID: row.ID, LessonID: row.LessonID, Title: value_objects.Title(row.Title), Position: int(row.Position), CreatedAt: row.CreatedAt})
	}
	return out, nil
}

func (s *Store) UpdateVariant(ctx context.Context, variant models.Variant) error {
	return s.q.UpdateVariant(ctx, curriculumsql.UpdateVariantParams{
		ID: variant.ID, Title: variant.Title.String(), Position: int32(variant.Position),
	})
}

func (s *Store) DeleteVariant(ctx context.Context, id uuid.UUID) error {
	return s.q.DeleteVariant(ctx, id)
}

func (s *Store) UpsertAssignment(ctx context.Context, asg models.UserModule) error {
	return s.q.UpsertUserModule(ctx, curriculumsql.UpsertUserModuleParams{
		UserID: asg.UserID, ModuleID: asg.ModuleID, AssignedBy: asg.AssignedBy,
		SourceGroupID: asg.SourceGroupID, AssignedAt: asg.AssignedAt,
	})
}

func (s *Store) ListAssignmentsByUser(ctx context.Context, userID uuid.UUID) ([]models.UserModule, error) {
	rows, err := s.q.ListUserModules(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]models.UserModule, 0, len(rows))
	for _, row := range rows {
		out = append(out, models.UserModule{
			UserID: row.UserID, ModuleID: row.ModuleID, AssignedBy: row.AssignedBy,
			SourceGroupID: row.SourceGroupID, AssignedAt: row.AssignedAt,
		})
	}
	return out, nil
}

func (s *Store) FindAssignment(ctx context.Context, userID, moduleID uuid.UUID) (models.UserModule, error) {
	row, err := s.q.GetUserModule(ctx, curriculumsql.GetUserModuleParams{UserID: userID, ModuleID: moduleID})
	if err != nil {
		return models.UserModule{}, mapNotFound(err)
	}
	return models.UserModule{
		UserID: row.UserID, ModuleID: row.ModuleID, AssignedBy: row.AssignedBy,
		SourceGroupID: row.SourceGroupID, AssignedAt: row.AssignedAt,
	}, nil
}

func (s *Store) HasAssignment(ctx context.Context, userID, moduleID uuid.UUID) (bool, error) {
	return s.q.HasUserModule(ctx, curriculumsql.HasUserModuleParams{UserID: userID, ModuleID: moduleID})
}

func (s *Store) UserHasTopic(ctx context.Context, userID, topicID uuid.UUID) (bool, error) {
	ids, err := s.q.ListTopicIDsForUser(ctx, userID)
	if err != nil {
		return false, err
	}
	for _, id := range ids {
		if id == topicID {
			return true, nil
		}
	}
	return false, nil
}

func mapArticle(row curriculumsql.TopicArticle) models.Article {
	return models.Article{
		ID: row.ID, TopicID: row.TopicID, Title: value_objects.Title(row.Title), BodyMD: row.BodyMd,
		CreatedBy: row.CreatedBy, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
}

func mapLesson(row curriculumsql.Lesson) models.Lesson {
	return models.Lesson{
		ID: row.ID, ModuleID: row.ModuleID, Title: value_objects.Title(row.Title),
		Position: int(row.Position), DurationSeconds: int32PtrToInt(row.DurationSeconds), CreatedAt: row.CreatedAt,
	}
}

func mapNotFound(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return errs.ErrNotFound
	}
	return err
}

func intPtrToInt32(v *int) *int32 {
	if v == nil {
		return nil
	}
	n := int32(*v)
	return &n
}

func int32PtrToInt(v *int32) *int {
	if v == nil {
		return nil
	}
	n := int(*v)
	return &n
}
