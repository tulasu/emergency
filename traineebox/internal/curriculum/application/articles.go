package application

import (
	"context"
	"path"
	"strings"
	"time"

	"traineebox/internal/curriculum/domain/abilities"
	"traineebox/internal/curriculum/domain/errs"
	"traineebox/internal/curriculum/domain/models"
	"traineebox/internal/curriculum/domain/repositories"
	"traineebox/internal/curriculum/domain/value_objects"

	"github.com/google/uuid"
)

const maxAttachmentBytes = 20 * 1024 * 1024

type CreateArticle struct {
	Topics   repositories.TopicRepository
	Articles repositories.ArticleRepository
}

type CreateArticleInput struct {
	ActorID uuid.UUID
	Role    value_objects.AccountRole
	TopicID uuid.UUID
	Title   string
	BodyMD  string
}

func (uc CreateArticle) Execute(ctx context.Context, in CreateArticleInput) (models.Article, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Article{}, err
	}
	if _, err := uc.Topics.FindByID(ctx, in.TopicID); err != nil {
		return models.Article{}, err
	}
	title, err := value_objects.NewTitle(in.Title)
	if err != nil {
		return models.Article{}, err
	}
	article := models.NewArticle(in.TopicID, title, in.BodyMD, in.ActorID)
	if err := uc.Articles.Create(ctx, article); err != nil {
		return models.Article{}, err
	}
	return article, nil
}

type ListArticles struct {
	Topics      repositories.TopicRepository
	Articles    repositories.ArticleRepository
	Assignments repositories.AssignmentRepository
}

type ListArticlesInput struct {
	ActorID uuid.UUID
	Role    value_objects.AccountRole
	TopicID uuid.UUID
}

func (uc ListArticles) Execute(ctx context.Context, in ListArticlesInput) ([]models.Article, error) {
	if _, err := uc.Topics.FindByID(ctx, in.TopicID); err != nil {
		return nil, err
	}
	if !in.Role.IsStaff() {
		ok, err := uc.Assignments.UserHasTopic(ctx, in.ActorID, in.TopicID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errs.ErrForbidden
		}
	}
	return uc.Articles.ListByTopic(ctx, in.TopicID)
}

type GetArticle struct {
	Articles    repositories.ArticleRepository
	Assignments repositories.AssignmentRepository
}

type GetArticleInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	ArticleID uuid.UUID
}

func (uc GetArticle) Execute(ctx context.Context, in GetArticleInput) (models.Article, error) {
	article, err := uc.Articles.FindByID(ctx, in.ArticleID)
	if err != nil {
		return models.Article{}, err
	}
	if in.Role.IsStaff() {
		return article, nil
	}
	ok, err := uc.Assignments.UserHasTopic(ctx, in.ActorID, article.TopicID)
	if err != nil {
		return models.Article{}, err
	}
	if !ok {
		return models.Article{}, errs.ErrForbidden
	}
	return article, nil
}

type UpdateArticle struct {
	Articles repositories.ArticleRepository
}

type UpdateArticleInput struct {
	Role      value_objects.AccountRole
	ArticleID uuid.UUID
	Title     string
	BodyMD    string
}

func (uc UpdateArticle) Execute(ctx context.Context, in UpdateArticleInput) (models.Article, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Article{}, err
	}
	article, err := uc.Articles.FindByID(ctx, in.ArticleID)
	if err != nil {
		return models.Article{}, err
	}
	title, err := value_objects.NewTitle(in.Title)
	if err != nil {
		return models.Article{}, err
	}
	article.Title = title
	article.BodyMD = in.BodyMD
	article.UpdatedAt = time.Now().UTC()
	if err := uc.Articles.Update(ctx, article); err != nil {
		return models.Article{}, err
	}
	return article, nil
}

type DeleteArticle struct {
	Articles repositories.ArticleRepository
}

func (uc DeleteArticle) Execute(ctx context.Context, role value_objects.AccountRole, id uuid.UUID) error {
	if err := abilities.ManageLibrary(role); err != nil {
		return err
	}
	if _, err := uc.Articles.FindByID(ctx, id); err != nil {
		return err
	}
	return uc.Articles.Delete(ctx, id)
}

type AddAttachment struct {
	Articles    repositories.ArticleRepository
	Attachments repositories.AttachmentRepository
	Blobs       repositories.BlobStore
}

type AddAttachmentInput struct {
	Role        value_objects.AccountRole
	ArticleID   uuid.UUID
	Filename    string
	ContentType string
	Data        []byte
}

func (uc AddAttachment) Execute(ctx context.Context, in AddAttachmentInput) (models.Attachment, error) {
	if err := abilities.ManageLibrary(in.Role); err != nil {
		return models.Attachment{}, err
	}
	if _, err := uc.Articles.FindByID(ctx, in.ArticleID); err != nil {
		return models.Attachment{}, err
	}
	if len(in.Data) == 0 {
		return models.Attachment{}, errs.ErrInvalidInput
	}
	if int64(len(in.Data)) > maxAttachmentBytes {
		return models.Attachment{}, errs.ErrTooLarge
	}
	filename := path.Base(strings.ReplaceAll(in.Filename, "\\", "/"))
	if filename == "" || filename == "." || filename == "/" {
		return models.Attachment{}, errs.ErrInvalidInput
	}
	contentType := strings.TrimSpace(in.ContentType)
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	id := uuid.New()
	key := id.String()
	att := models.Attachment{
		ID:          id,
		ArticleID:   in.ArticleID,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   int64(len(in.Data)),
		StorageKey:  key,
		CreatedAt:   time.Now().UTC(),
	}
	if err := uc.Blobs.Put(ctx, key, in.Data); err != nil {
		return models.Attachment{}, err
	}
	if err := uc.Attachments.Create(ctx, att); err != nil {
		_ = uc.Blobs.Delete(ctx, key)
		return models.Attachment{}, err
	}
	return att, nil
}

type ListAttachments struct {
	Articles    repositories.ArticleRepository
	Attachments repositories.AttachmentRepository
	Assignments repositories.AssignmentRepository
}

type ListAttachmentsInput struct {
	ActorID   uuid.UUID
	Role      value_objects.AccountRole
	ArticleID uuid.UUID
}

func (uc ListAttachments) Execute(ctx context.Context, in ListAttachmentsInput) ([]models.Attachment, error) {
	article, err := uc.Articles.FindByID(ctx, in.ArticleID)
	if err != nil {
		return nil, err
	}
	if !in.Role.IsStaff() {
		ok, err := uc.Assignments.UserHasTopic(ctx, in.ActorID, article.TopicID)
		if err != nil {
			return nil, err
		}
		if !ok {
			return nil, errs.ErrForbidden
		}
	}
	return uc.Attachments.ListByArticle(ctx, in.ArticleID)
}

type GetAttachment struct {
	Articles    repositories.ArticleRepository
	Attachments repositories.AttachmentRepository
	Assignments repositories.AssignmentRepository
	Blobs       repositories.BlobStore
}

type GetAttachmentInput struct {
	ActorID      uuid.UUID
	Role         value_objects.AccountRole
	AttachmentID uuid.UUID
}

func (uc GetAttachment) Execute(ctx context.Context, in GetAttachmentInput) (models.Attachment, []byte, error) {
	att, err := uc.Attachments.FindByID(ctx, in.AttachmentID)
	if err != nil {
		return models.Attachment{}, nil, err
	}
	article, err := uc.Articles.FindByID(ctx, att.ArticleID)
	if err != nil {
		return models.Attachment{}, nil, err
	}
	if !in.Role.IsStaff() {
		ok, err := uc.Assignments.UserHasTopic(ctx, in.ActorID, article.TopicID)
		if err != nil {
			return models.Attachment{}, nil, err
		}
		if !ok {
			return models.Attachment{}, nil, errs.ErrForbidden
		}
	}
	data, err := uc.Blobs.Get(ctx, att.StorageKey)
	if err != nil {
		return models.Attachment{}, nil, err
	}
	return att, data, nil
}

type DeleteAttachment struct {
	Attachments repositories.AttachmentRepository
	Blobs       repositories.BlobStore
}

func (uc DeleteAttachment) Execute(ctx context.Context, role value_objects.AccountRole, id uuid.UUID) error {
	if err := abilities.ManageLibrary(role); err != nil {
		return err
	}
	att, err := uc.Attachments.FindByID(ctx, id)
	if err != nil {
		return err
	}
	if err := uc.Attachments.Delete(ctx, id); err != nil {
		return err
	}
	_ = uc.Blobs.Delete(ctx, att.StorageKey)
	return nil
}
