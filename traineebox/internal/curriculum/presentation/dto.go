package presentation

import (
	"time"

	"traineebox/internal/curriculum/application"
	"traineebox/internal/curriculum/domain/models"
)

type topicDTO struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
}

type articleDTO struct {
	ID        string `json:"id"`
	TopicID   string `json:"topic_id"`
	Title     string `json:"title"`
	BodyMD    string `json:"body_md"`
	CreatedBy string `json:"created_by"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type attachmentDTO struct {
	ID          string `json:"id"`
	ArticleID   string `json:"article_id"`
	Filename    string `json:"filename"`
	ContentType string `json:"content_type"`
	SizeBytes   int64  `json:"size_bytes"`
	CreatedAt   string `json:"created_at"`
}

type moduleDTO struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedBy   string `json:"created_by"`
	CreatedAt   string `json:"created_at"`
}

type lessonDTO struct {
	ID              string `json:"id"`
	ModuleID        string `json:"module_id"`
	Title           string `json:"title"`
	Position        int    `json:"position"`
	DurationSeconds *int   `json:"duration_seconds,omitempty"`
	CreatedAt       string `json:"created_at"`
}

type variantDTO struct {
	ID        string `json:"id"`
	LessonID  string `json:"lesson_id"`
	Title     string `json:"title"`
	Position  int    `json:"position"`
	CreatedAt string `json:"created_at"`
}

type assignedLessonDTO struct {
	Lesson    lessonDTO  `json:"lesson"`
	Variant   variantDTO `json:"variant"`
	AttemptID *string    `json:"attempt_id,omitempty"`
	Status    string     `json:"status,omitempty"`
}

type assignedModuleDTO struct {
	Module  moduleDTO           `json:"module"`
	Lessons []assignedLessonDTO `json:"lessons"`
}

func toTopicDTO(t models.Topic) topicDTO {
	return topicDTO{
		ID: t.ID.String(), Title: t.Title.String(), CreatedBy: t.CreatedBy.String(),
		CreatedAt: t.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toArticleDTO(a models.Article) articleDTO {
	return articleDTO{
		ID: a.ID.String(), TopicID: a.TopicID.String(), Title: a.Title.String(), BodyMD: a.BodyMD,
		CreatedBy: a.CreatedBy.String(), CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339Nano),
		UpdatedAt: a.UpdatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toAttachmentDTO(a models.Attachment) attachmentDTO {
	return attachmentDTO{
		ID: a.ID.String(), ArticleID: a.ArticleID.String(), Filename: a.Filename,
		ContentType: a.ContentType, SizeBytes: a.SizeBytes,
		CreatedAt: a.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toModuleDTO(m models.Module) moduleDTO {
	return moduleDTO{
		ID: m.ID.String(), Title: m.Title.String(), Description: m.Description,
		CreatedBy: m.CreatedBy.String(), CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toLessonDTO(l models.Lesson) lessonDTO {
	return lessonDTO{
		ID: l.ID.String(), ModuleID: l.ModuleID.String(), Title: l.Title.String(),
		Position: l.Position, DurationSeconds: l.DurationSeconds,
		CreatedAt: l.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toVariantDTO(v models.Variant) variantDTO {
	return variantDTO{
		ID: v.ID.String(), LessonID: v.LessonID.String(), Title: v.Title.String(),
		Position: v.Position, CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toAssignedModuleDTO(m application.AssignedModule) assignedModuleDTO {
	lessons := make([]assignedLessonDTO, 0, len(m.Lessons))
	for _, l := range m.Lessons {
		item := assignedLessonDTO{
			Lesson:  toLessonDTO(l.Lesson),
			Variant: toVariantDTO(l.Variant),
			Status:  l.Status,
		}
		if l.AttemptID != nil {
			s := l.AttemptID.String()
			item.AttemptID = &s
		}
		lessons = append(lessons, item)
	}
	return assignedModuleDTO{Module: toModuleDTO(m.Module), Lessons: lessons}
}
