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
	ID               string   `json:"id"`
	Title            string   `json:"title"`
	Description      string   `json:"description"`
	Status           string   `json:"status"`
	SuccessThreshold int      `json:"success_threshold"`
	CreatedBy        string   `json:"created_by"`
	CreatedAt        string   `json:"created_at"`
	LessonCount      *int     `json:"lesson_count,omitempty"`
	AssignedCount    *int     `json:"assigned_count,omitempty"`
	AssignedGroups   *int     `json:"assigned_groups,omitempty"`
	AssignedUsers    *int     `json:"assigned_users,omitempty"`
	AssignedLabel    string   `json:"assigned_label,omitempty"`
	OpenedDone       *int     `json:"opened_done,omitempty"`
	OpenedTotal      *int     `json:"opened_total,omitempty"`
	SuccessRate      *float64 `json:"success_rate,omitempty"`
}

type lessonDTO struct {
	ID              string   `json:"id"`
	ModuleID        string   `json:"module_id"`
	Title           string   `json:"title"`
	Position        int      `json:"position"`
	DurationSeconds *int     `json:"duration_seconds,omitempty"`
	ArchivedAt      *string  `json:"archived_at,omitempty"`
	CreatedAt       string   `json:"created_at"`
	VariantCount    *int     `json:"variant_count,omitempty"`
	TicketCount     *int     `json:"ticket_count,omitempty"`
	VariantsLabel   string   `json:"variants_label,omitempty"`
	OpenedFor       *int     `json:"opened_for,omitempty"`
	OpenedTotal     *int     `json:"opened_total,omitempty"`
	PassedRate      *float64 `json:"passed_rate,omitempty"`
	Attention       string   `json:"attention,omitempty"`
}

type variantDTO struct {
	ID        string `json:"id"`
	LessonID  string `json:"lesson_id"`
	Title     string `json:"title"`
	Position  int    `json:"position"`
	Status    string `json:"status"`
	IsPrimary bool   `json:"is_primary"`
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

type assignmentGroupDTO struct {
	Label string `json:"label"`
	Count int    `json:"count"`
}

type assignmentSummaryDTO struct {
	TotalUsers  int                  `json:"total_users"`
	Groups      []assignmentGroupDTO `json:"groups"`
	Individuals int                  `json:"individuals"`
}

type moduleSummaryDTO struct {
	Module     moduleDTO             `json:"module"`
	Lessons    []lessonDTO           `json:"lessons"`
	Assignment *assignmentSummaryDTO `json:"assignment,omitempty"`
	Attention  []string              `json:"attention,omitempty"`
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
		Status: m.Status.String(), SuccessThreshold: m.SuccessThreshold,
		CreatedBy: m.CreatedBy.String(), CreatedAt: m.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toModuleListDTO(item models.ModuleListItem) moduleDTO {
	dto := toModuleDTO(item.Module)
	n := item.LessonCount
	dto.LessonCount = &n
	ac := item.AssignedCount
	dto.AssignedCount = &ac
	ag := item.AssignedGroups
	dto.AssignedGroups = &ag
	au := item.AssignedUsers
	dto.AssignedUsers = &au
	dto.AssignedLabel = item.AssignedLabel
	od, ot := item.OpenedDone, item.OpenedTotal
	dto.OpenedDone = &od
	dto.OpenedTotal = &ot
	dto.SuccessRate = item.SuccessRate
	return dto
}

func toLessonDTO(l models.Lesson) lessonDTO {
	dto := lessonDTO{
		ID: l.ID.String(), ModuleID: l.ModuleID.String(), Title: l.Title.String(),
		Position: l.Position, DurationSeconds: l.DurationSeconds,
		CreatedAt: l.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
	if l.ArchivedAt != nil {
		s := l.ArchivedAt.UTC().Format(time.RFC3339Nano)
		dto.ArchivedAt = &s
	}
	return dto
}

func toVariantDTO(v models.Variant) variantDTO {
	return variantDTO{
		ID: v.ID.String(), LessonID: v.LessonID.String(), Title: v.Title.String(),
		Position: v.Position, Status: v.Status.String(), IsPrimary: v.IsPrimary,
		CreatedAt: v.CreatedAt.UTC().Format(time.RFC3339Nano),
	}
}

func toAssignedModuleDTO(m application.AssignedModule) assignedModuleDTO {
	lessons := make([]assignedLessonDTO, 0, len(m.Lessons))
	for _, l := range m.Lessons {
		item := assignedLessonDTO{
			Lesson: toLessonDTO(l.Lesson), Variant: toVariantDTO(l.Variant), Status: l.Status,
		}
		if l.AttemptID != nil {
			s := l.AttemptID.String()
			item.AttemptID = &s
		}
		lessons = append(lessons, item)
	}
	return assignedModuleDTO{Module: toModuleDTO(m.Module), Lessons: lessons}
}

func toModuleSummaryDTO(s models.ModuleSummary) moduleSummaryDTO {
	lessons := make([]lessonDTO, 0, len(s.Lessons))
	for _, l := range s.Lessons {
		dto := toLessonDTO(l.Lesson)
		vc, tc := l.VariantCount, l.TicketCount
		dto.VariantCount = &vc
		dto.TicketCount = &tc
		dto.VariantsLabel = l.VariantsLabel
		of, ot := l.OpenedFor, l.OpenedTotal
		dto.OpenedFor = &of
		dto.OpenedTotal = &ot
		dto.PassedRate = l.PassedRate
		dto.Attention = l.Attention
		lessons = append(lessons, dto)
	}
	groups := make([]assignmentGroupDTO, 0, len(s.Assignment.Groups))
	for _, g := range s.Assignment.Groups {
		groups = append(groups, assignmentGroupDTO{Label: g.Label, Count: g.Count})
	}
	asg := assignmentSummaryDTO{
		TotalUsers:  s.Assignment.TotalUsers,
		Groups:      groups,
		Individuals: s.Assignment.Individuals,
	}
	return moduleSummaryDTO{
		Module:     toModuleDTO(s.Module),
		Lessons:    lessons,
		Assignment: &asg,
		Attention:  s.Attention,
	}
}
