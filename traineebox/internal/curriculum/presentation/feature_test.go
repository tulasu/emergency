package presentation_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"traineebox/internal/testkit"
)

func TestCurriculumCRUDAndAssign(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	_ = testkit.SeedUser(t, pool, "cteach", "password1", "teacher")
	_ = testkit.SeedUser(t, pool, "cstud", "password1", "student")
	teacher := login(t, handler, "cteach", "password1")
	student := login(t, handler, "cstud", "password1")
	studentID := me(t, handler, student)

	forbidden := doJSON(t, handler, http.MethodPost, "/topics", student, map[string]string{"title": "No"})
	if forbidden.Code != http.StatusForbidden {
		t.Fatalf("student create topic = %d", forbidden.Code)
	}

	topic := doJSON(t, handler, http.MethodPost, "/topics", teacher, map[string]string{"title": "Тема"})
	mustCode(t, topic, http.StatusOK)
	var topicBody struct {
		ID string `json:"id"`
	}
	mustJSON(t, topic.Body, &topicBody)

	article := doJSON(t, handler, http.MethodPost, "/topics/"+topicBody.ID+"/articles", teacher, map[string]any{
		"title": "Статья", "body_md": "# H1\ntext",
	})
	mustCode(t, article, http.StatusOK)

	module := doJSON(t, handler, http.MethodPost, "/modules", teacher, map[string]any{"title": "Модуль", "description": "d"})
	mustCode(t, module, http.StatusOK)
	var moduleBody struct {
		ID string `json:"id"`
	}
	mustJSON(t, module.Body, &moduleBody)

	lesson := doJSON(t, handler, http.MethodPost, "/modules/"+moduleBody.ID+"/lessons", teacher, map[string]any{
		"title": "Урок", "position": 1, "duration_seconds": 120,
	})
	mustCode(t, lesson, http.StatusOK)
	var lessonBody struct {
		ID string `json:"id"`
	}
	mustJSON(t, lesson.Body, &lessonBody)

	variant := doJSON(t, handler, http.MethodPost, "/lessons/"+lessonBody.ID+"/variants", teacher, map[string]any{
		"title": "Вар 1", "position": 0,
	})
	mustCode(t, variant, http.StatusOK)
	var variantBody struct {
		ID string `json:"id"`
	}
	mustJSON(t, variant.Body, &variantBody)

	group := doJSON(t, handler, http.MethodPost, "/groups", teacher, map[string]string{"name": "G"})
	mustCode(t, group, http.StatusOK)
	var groupBody struct {
		ID string `json:"id"`
	}
	mustJSON(t, group.Body, &groupBody)
	mustCode(t, doJSON(t, handler, http.MethodPost, "/groups/"+groupBody.ID+"/members", teacher, map[string]string{
		"user_id": studentID, "role": "student",
	}), http.StatusOK)

	assign := doJSON(t, handler, http.MethodPost, "/groups/"+groupBody.ID+"/modules", teacher, map[string]any{
		"module_id": moduleBody.ID,
		"lessons":   []map[string]string{{"lesson_id": lessonBody.ID, "variant_id": variantBody.ID}},
	})
	mustCode(t, assign, http.StatusNoContent)

	mine := doJSON(t, handler, http.MethodGet, "/me/modules", student, nil)
	mustCode(t, mine, http.StatusOK)
	var assigned []map[string]any
	mustJSON(t, mine.Body, &assigned)
	if len(assigned) != 1 {
		t.Fatalf("me modules = %s", mine.Body)
	}

	articles := doJSON(t, handler, http.MethodGet, "/topics/"+topicBody.ID+"/articles", student, nil)
	if articles.Code != http.StatusForbidden && articles.Code != http.StatusOK {
		t.Fatalf("student articles = %d", articles.Code)
	}
}

type jsonRes struct {
	Code int
	Body []byte
}

func doJSON(t *testing.T, handler http.Handler, method, path, token string, body any) jsonRes {
	t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	return jsonRes{Code: rec.Code, Body: rec.Body.Bytes()}
}

func login(t *testing.T, handler http.Handler, loginName, password string) string {
	t.Helper()
	res := doJSON(t, handler, http.MethodPost, "/auth/login", "", map[string]string{"login": loginName, "password": password})
	mustCode(t, res, http.StatusOK)
	var out struct {
		Token string `json:"token"`
	}
	mustJSON(t, res.Body, &out)
	return out.Token
}

func me(t *testing.T, handler http.Handler, token string) string {
	t.Helper()
	res := doJSON(t, handler, http.MethodGet, "/auth/me", token, nil)
	mustCode(t, res, http.StatusOK)
	var out struct {
		ID string `json:"id"`
	}
	mustJSON(t, res.Body, &out)
	return out.ID
}

func mustCode(t *testing.T, res jsonRes, want int) {
	t.Helper()
	if res.Code != want {
		t.Fatalf("status = %d want %d body=%s", res.Code, want, res.Body)
	}
}

func mustJSON(t *testing.T, raw []byte, dest any) {
	t.Helper()
	if err := json.Unmarshal(raw, dest); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}
