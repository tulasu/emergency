package repositories

import (
	"context"

	"traineebox/internal/curriculum/domain/models"

	"github.com/google/uuid"
)

type TopicRepository interface {
	Create(ctx context.Context, topic models.Topic) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Topic, error)
	List(ctx context.Context) ([]models.Topic, error)
	Update(ctx context.Context, topic models.Topic) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type ArticleRepository interface {
	Create(ctx context.Context, article models.Article) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Article, error)
	ListByTopic(ctx context.Context, topicID uuid.UUID) ([]models.Article, error)
	Update(ctx context.Context, article models.Article) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AttachmentRepository interface {
	Create(ctx context.Context, att models.Attachment) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Attachment, error)
	ListByArticle(ctx context.Context, articleID uuid.UUID) ([]models.Attachment, error)
	Delete(ctx context.Context, id uuid.UUID) error
}

type ModuleListFilter struct {
	Q       string
	Scope   string
	ActorID uuid.UUID
}

type ModuleRepository interface {
	Create(ctx context.Context, module models.Module) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Module, error)
	List(ctx context.Context) ([]models.Module, error)
	ListFiltered(ctx context.Context, filter ModuleListFilter) ([]models.Module, error)
	CountLessons(ctx context.Context, moduleID uuid.UUID) (int, error)
	CountAssignedUsers(ctx context.Context, moduleID uuid.UUID) (int, error)
	CountAssignmentBreakdown(ctx context.Context, moduleID uuid.UUID) (groups int, users int, err error)
	Update(ctx context.Context, module models.Module) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type LessonRepository interface {
	Create(ctx context.Context, lesson models.Lesson) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Lesson, error)
	ListByModule(ctx context.Context, moduleID uuid.UUID) ([]models.Lesson, error)
	ListPool(ctx context.Context, q string) ([]models.Lesson, error)
	FindByVariant(ctx context.Context, variantID uuid.UUID) (models.Lesson, error)
	CountVariants(ctx context.Context, lessonID uuid.UUID) (int, error)
	CountTickets(ctx context.Context, lessonID uuid.UUID) (int, error)
	Update(ctx context.Context, lesson models.Lesson) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type VariantRepository interface {
	Create(ctx context.Context, variant models.Variant) error
	FindByID(ctx context.Context, id uuid.UUID) (models.Variant, error)
	ListByLesson(ctx context.Context, lessonID uuid.UUID) ([]models.Variant, error)
	ClearPrimary(ctx context.Context, lessonID uuid.UUID) error
	Update(ctx context.Context, variant models.Variant) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type AssignmentRepository interface {
	Upsert(ctx context.Context, asg models.UserModule) error
	ListByUser(ctx context.Context, userID uuid.UUID) ([]models.UserModule, error)
	Find(ctx context.Context, userID, moduleID uuid.UUID) (models.UserModule, error)
	Has(ctx context.Context, userID, moduleID uuid.UUID) (bool, error)
	UserHasTopic(ctx context.Context, userID, topicID uuid.UUID) (bool, error)
}

type BlobStore interface {
	Put(ctx context.Context, key string, data []byte) error
	Get(ctx context.Context, key string) ([]byte, error)
	Delete(ctx context.Context, key string) error
}

type AttemptIssuer interface {
	IssueAvailable(ctx context.Context, userID, variantID, grantedBy uuid.UUID) error
}

type GroupMember struct {
	UserID uuid.UUID
	Role   string
}

type GroupView struct {
	ID      uuid.UUID
	Members []GroupMember
}

type GroupDirectory interface {
	FindByID(ctx context.Context, id uuid.UUID) (GroupView, error)
}
