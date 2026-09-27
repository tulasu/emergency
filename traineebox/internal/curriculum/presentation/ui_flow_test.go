package presentation_test

import (
	"net/http"
	"testing"
	"time"

	"traineebox/internal/testkit"
)

func TestCurriculumUILibraryAndOpen(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	_ = testkit.SeedUser(t, pool, "uteach", "password1", "teacher")
	_ = testkit.SeedUser(t, pool, "ustud", "password1", "student")
	teacher := login(t, handler, "uteach", "password1")
	student := login(t, handler, "ustud", "password1")
	studentID := me(t, handler, student)

	topic := doJSON(t, handler, http.MethodPost, "/topics", teacher, map[string]string{"title": "Тема UI"})
	mustCode(t, topic, http.StatusOK)
	var topicBody struct {
		ID string `json:"id"`
	}
	mustJSON(t, topic.Body, &topicBody)

	module := doJSON(t, handler, http.MethodPost, "/modules", teacher, map[string]any{"title": "Модуль UI", "description": "d"})
	mustCode(t, module, http.StatusOK)
	var moduleBody struct {
		ID               string `json:"id"`
		Status           string `json:"status"`
		SuccessThreshold int    `json:"success_threshold"`
	}
	mustJSON(t, module.Body, &moduleBody)
	if moduleBody.Status != "draft" || moduleBody.SuccessThreshold != 70 {
		t.Fatalf("module defaults = %+v", moduleBody)
	}

	patched := doJSON(t, handler, http.MethodPatch, "/modules/"+moduleBody.ID, teacher, map[string]any{
		"title": "Модуль UI", "description": "d", "status": "active", "success_threshold": 80,
	})
	mustCode(t, patched, http.StatusOK)

	lesson := doJSON(t, handler, http.MethodPost, "/modules/"+moduleBody.ID+"/lessons", teacher, map[string]any{
		"title": "Урок UI", "position": 0,
	})
	mustCode(t, lesson, http.StatusOK)
	var lessonBody struct {
		ID string `json:"id"`
	}
	mustJSON(t, lesson.Body, &lessonBody)

	variant := doJSON(t, handler, http.MethodPost, "/lessons/"+lessonBody.ID+"/variants", teacher, map[string]any{
		"title": "Вариант A", "position": 0,
	})
	mustCode(t, variant, http.StatusOK)
	var variantBody struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	mustJSON(t, variant.Body, &variantBody)

	clone := doJSON(t, handler, http.MethodPost, "/variants/"+variantBody.ID+"/clone", teacher, map[string]any{
		"title": "Вариант B",
	})
	mustCode(t, clone, http.StatusOK)

	approve := doJSON(t, handler, http.MethodPatch, "/variants/"+variantBody.ID, teacher, map[string]any{
		"title": "Вариант A", "position": 0, "status": "approved", "is_primary": true,
	})
	mustCode(t, approve, http.StatusOK)

	summary := doJSON(t, handler, http.MethodGet, "/modules/"+moduleBody.ID+"/summary", teacher, nil)
	mustCode(t, summary, http.StatusOK)

	poolLessons := doJSON(t, handler, http.MethodGet, "/lessons?q=Урок", teacher, nil)
	mustCode(t, poolLessons, http.StatusOK)

	lib := doJSON(t, handler, http.MethodPost, "/tickets", teacher, map[string]any{
		"topic_id": topicBody.ID, "title": "Карточка пула", "body": "текст",
	})
	mustCode(t, lib, http.StatusOK)
	var libBody struct {
		ID        string  `json:"id"`
		VariantID *string `json:"variant_id"`
	}
	mustJSON(t, lib.Body, &libBody)
	if libBody.VariantID != nil {
		t.Fatalf("library ticket must have null variant_id")
	}

	listed := doJSON(t, handler, http.MethodGet, "/tickets?q=пула", teacher, nil)
	mustCode(t, listed, http.StatusOK)

	copied := doJSON(t, handler, http.MethodPost, "/variants/"+variantBody.ID+"/tickets/from-pool", teacher, map[string]any{
		"ticket_id": libBody.ID,
	})
	mustCode(t, copied, http.StatusOK)

	group := doJSON(t, handler, http.MethodPost, "/groups", teacher, map[string]string{"name": "GU"})
	mustCode(t, group, http.StatusOK)
	var groupBody struct {
		ID string `json:"id"`
	}
	mustJSON(t, group.Body, &groupBody)
	mustCode(t, doJSON(t, handler, http.MethodPost, "/groups/"+groupBody.ID+"/members", teacher, map[string]string{
		"user_id": studentID, "role": "student",
	}), http.StatusOK)
	mustCode(t, doJSON(t, handler, http.MethodPost, "/groups/"+groupBody.ID+"/modules", teacher, map[string]any{
		"module_id": moduleBody.ID,
		"lessons":   []map[string]string{{"lesson_id": lessonBody.ID, "variant_id": variantBody.ID}},
	}), http.StatusNoContent)

	from := time.Now().UTC().Add(-time.Hour).Format(time.RFC3339)
	until := time.Now().UTC().Add(48 * time.Hour).Format(time.RFC3339)
	opened := doJSON(t, handler, http.MethodPost, "/variants/"+variantBody.ID+"/open", teacher, map[string]any{
		"mode": "users", "user_ids": []string{studentID},
		"available_from": from, "deadline_at": until,
	})
	mustCode(t, opened, http.StatusOK)

	mine := doJSON(t, handler, http.MethodGet, "/me/modules", student, nil)
	mustCode(t, mine, http.StatusOK)

	archived := doJSON(t, handler, http.MethodPost, "/lessons/"+lessonBody.ID+"/archive", teacher, nil)
	mustCode(t, archived, http.StatusOK)
}
