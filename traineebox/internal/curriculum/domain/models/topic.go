package models

import (
	"time"

	"traineebox/internal/curriculum/domain/value_objects"

	"github.com/google/uuid"
)

type Topic struct {
	ID        uuid.UUID
	Title     value_objects.Title
	CreatedBy uuid.UUID
	CreatedAt time.Time
}

func NewTopic(title value_objects.Title, createdBy uuid.UUID) Topic {
	return Topic{
		ID:        uuid.New(),
		Title:     title,
		CreatedBy: createdBy,
		CreatedAt: time.Now().UTC(),
	}
}

type Article struct {
	ID        uuid.UUID
	TopicID   uuid.UUID
	Title     value_objects.Title
	BodyMD    string
	CreatedBy uuid.UUID
	CreatedAt time.Time
	UpdatedAt time.Time
}

func NewArticle(topicID uuid.UUID, title value_objects.Title, bodyMD string, createdBy uuid.UUID) Article {
	now := time.Now().UTC()
	return Article{
		ID:        uuid.New(),
		TopicID:   topicID,
		Title:     title,
		BodyMD:    bodyMD,
		CreatedBy: createdBy,
		CreatedAt: now,
		UpdatedAt: now,
	}
}

type Attachment struct {
	ID          uuid.UUID
	ArticleID   uuid.UUID
	Filename    string
	ContentType string
	SizeBytes   int64
	StorageKey  string
	CreatedAt   time.Time
}

func NewAttachment(articleID uuid.UUID, filename, contentType, storageKey string, size int64) Attachment {
	return Attachment{
		ID:          uuid.New(),
		ArticleID:   articleID,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   size,
		StorageKey:  storageKey,
		CreatedAt:   time.Now().UTC(),
	}
}
