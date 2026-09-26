package infrastructure

import (
	"context"

	"traineebox/internal/curriculum/domain/models"
	"traineebox/internal/curriculum/domain/repositories"

	"github.com/google/uuid"
)

type Topics struct{ *Store }

func (t Topics) Create(ctx context.Context, topic models.Topic) error {
	return t.CreateTopic(ctx, topic)
}
func (t Topics) FindByID(ctx context.Context, id uuid.UUID) (models.Topic, error) {
	return t.FindTopicByID(ctx, id)
}
func (t Topics) List(ctx context.Context) ([]models.Topic, error) { return t.ListTopics(ctx) }
func (t Topics) Update(ctx context.Context, topic models.Topic) error {
	return t.UpdateTopic(ctx, topic)
}
func (t Topics) Delete(ctx context.Context, id uuid.UUID) error { return t.DeleteTopic(ctx, id) }

type Articles struct{ *Store }

func (a Articles) Create(ctx context.Context, article models.Article) error {
	return a.CreateArticle(ctx, article)
}
func (a Articles) FindByID(ctx context.Context, id uuid.UUID) (models.Article, error) {
	return a.FindArticleByID(ctx, id)
}
func (a Articles) ListByTopic(ctx context.Context, topicID uuid.UUID) ([]models.Article, error) {
	return a.ListArticlesByTopic(ctx, topicID)
}
func (a Articles) Update(ctx context.Context, article models.Article) error {
	return a.UpdateArticle(ctx, article)
}
func (a Articles) Delete(ctx context.Context, id uuid.UUID) error { return a.DeleteArticle(ctx, id) }

type Attachments struct{ *Store }

func (a Attachments) Create(ctx context.Context, att models.Attachment) error {
	return a.CreateAttachment(ctx, att)
}
func (a Attachments) FindByID(ctx context.Context, id uuid.UUID) (models.Attachment, error) {
	return a.FindAttachmentByID(ctx, id)
}
func (a Attachments) ListByArticle(ctx context.Context, articleID uuid.UUID) ([]models.Attachment, error) {
	return a.ListAttachmentsByArticle(ctx, articleID)
}
func (a Attachments) Delete(ctx context.Context, id uuid.UUID) error {
	return a.DeleteAttachment(ctx, id)
}

type Modules struct{ *Store }

func (m Modules) Create(ctx context.Context, module models.Module) error {
	return m.CreateModule(ctx, module)
}
func (m Modules) FindByID(ctx context.Context, id uuid.UUID) (models.Module, error) {
	return m.FindModuleByID(ctx, id)
}
func (m Modules) List(ctx context.Context) ([]models.Module, error) { return m.ListModules(ctx) }
func (m Modules) Update(ctx context.Context, module models.Module) error {
	return m.UpdateModule(ctx, module)
}
func (m Modules) Delete(ctx context.Context, id uuid.UUID) error { return m.DeleteModule(ctx, id) }

type Lessons struct{ *Store }

func (l Lessons) Create(ctx context.Context, lesson models.Lesson) error {
	return l.CreateLesson(ctx, lesson)
}
func (l Lessons) FindByID(ctx context.Context, id uuid.UUID) (models.Lesson, error) {
	return l.FindLessonByID(ctx, id)
}
func (l Lessons) ListByModule(ctx context.Context, moduleID uuid.UUID) ([]models.Lesson, error) {
	return l.ListLessonsByModule(ctx, moduleID)
}
func (l Lessons) FindByVariant(ctx context.Context, variantID uuid.UUID) (models.Lesson, error) {
	return l.FindLessonByVariant(ctx, variantID)
}
func (l Lessons) Update(ctx context.Context, lesson models.Lesson) error {
	return l.UpdateLesson(ctx, lesson)
}
func (l Lessons) Delete(ctx context.Context, id uuid.UUID) error { return l.DeleteLesson(ctx, id) }

type Variants struct{ *Store }

func (v Variants) Create(ctx context.Context, variant models.Variant) error {
	return v.CreateVariant(ctx, variant)
}
func (v Variants) FindByID(ctx context.Context, id uuid.UUID) (models.Variant, error) {
	return v.FindVariantByID(ctx, id)
}
func (v Variants) ListByLesson(ctx context.Context, lessonID uuid.UUID) ([]models.Variant, error) {
	return v.ListVariantsByLesson(ctx, lessonID)
}
func (v Variants) Update(ctx context.Context, variant models.Variant) error {
	return v.UpdateVariant(ctx, variant)
}
func (v Variants) Delete(ctx context.Context, id uuid.UUID) error { return v.DeleteVariant(ctx, id) }

type Assignments struct{ *Store }

func (a Assignments) Upsert(ctx context.Context, asg models.UserModule) error {
	return a.UpsertAssignment(ctx, asg)
}
func (a Assignments) ListByUser(ctx context.Context, userID uuid.UUID) ([]models.UserModule, error) {
	return a.ListAssignmentsByUser(ctx, userID)
}
func (a Assignments) Find(ctx context.Context, userID, moduleID uuid.UUID) (models.UserModule, error) {
	return a.FindAssignment(ctx, userID, moduleID)
}
func (a Assignments) Has(ctx context.Context, userID, moduleID uuid.UUID) (bool, error) {
	return a.HasAssignment(ctx, userID, moduleID)
}

var _ repositories.TopicRepository = Topics{}
var _ repositories.ArticleRepository = Articles{}
var _ repositories.AttachmentRepository = Attachments{}
var _ repositories.ModuleRepository = Modules{}
var _ repositories.LessonRepository = Lessons{}
var _ repositories.VariantRepository = Variants{}
var _ repositories.AssignmentRepository = Assignments{}
