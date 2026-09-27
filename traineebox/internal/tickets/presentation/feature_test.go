package presentation_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"traineebox/internal/testkit"
)

func TestStudentAttemptOnceThenTeacherGrantsRetry(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	_ = testkit.SeedUser(t, pool, "owner1", "password1", "teacher")
	_ = testkit.SeedUser(t, pool, "stud1", "password1", "student")
	ownerToken := loginToken(t, handler, "owner1", "password1")
	studentToken := loginToken(t, handler, "stud1", "password1")
	studentID := mustFindLogin(t, handler, studentToken)

	createGroup := doRequest(t, handler, http.MethodPost, "/groups", ownerToken, map[string]string{"name": "Cohort"})
	mustOK(t, createGroup)
	var group struct {
		ID string `json:"id"`
	}
	mustDecode(t, createGroup.Body, &group)
	mustOK(t, doRequest(t, handler, http.MethodPost, "/groups/"+group.ID+"/members", ownerToken, map[string]string{
		"user_id": studentID, "role": "student",
	}))

	topic := createNamed(t, handler, ownerToken, "/topics", map[string]string{"title": "Пожар"})
	mustOK(t, doRequest(t, handler, http.MethodPost, "/topics/"+topic+"/articles", ownerToken, map[string]any{
		"title": "Как принимать вызов", "body_md": "# Intro\n\nТекст",
	}))
	module := createNamed(t, handler, ownerToken, "/modules", map[string]any{"title": "Смена 1", "description": ""})
	lessonBody := doRequest(t, handler, http.MethodPost, "/modules/"+module+"/lessons", ownerToken, map[string]any{
		"title": "Занятие 1", "position": 0, "duration_seconds": 3600,
	})
	mustOK(t, lessonBody)
	var lesson struct {
		ID string `json:"id"`
	}
	mustDecode(t, lessonBody.Body, &lesson)

	variantA := createNamed(t, handler, ownerToken, "/lessons/"+lesson.ID+"/variants", map[string]any{"title": "Вариант A", "position": 0})
	variantB := createNamed(t, handler, ownerToken, "/lessons/"+lesson.ID+"/variants", map[string]any{"title": "Вариант B", "position": 1})

	ticketA := createNamed(t, handler, ownerToken, "/variants/"+variantA+"/tickets", map[string]any{
		"topic_id": topic, "title": "Ticket A", "body": "Call about fire",
	})
	getTicket := doRequest(t, handler, http.MethodGet, "/tickets/"+ticketA, ownerToken, nil)
	mustOK(t, getTicket)
	var ticketAudio struct {
		AudioStatus string `json:"audio_status"`
	}
	mustDecode(t, getTicket.Body, &ticketAudio)
	if ticketAudio.AudioStatus != "none" {
		t.Fatalf("audio_status = %q, want none", ticketAudio.AudioStatus)
	}
	ticketB := createNamed(t, handler, ownerToken, "/variants/"+variantB+"/tickets", map[string]any{
		"topic_id": topic, "title": "Ticket B", "body": "Another fire",
	})
	setRef(t, handler, ownerToken, ticketA)
	setRef(t, handler, ownerToken, ticketB)

	assign := doRequest(t, handler, http.MethodPost, "/groups/"+group.ID+"/modules", ownerToken, map[string]any{
		"module_id": module,
		"lessons":   []map[string]string{{"lesson_id": lesson.ID, "variant_id": variantA}},
	})
	mustOK(t, assign)

	mine := doRequest(t, handler, http.MethodGet, "/me/modules", studentToken, nil)
	mustOK(t, mine)
	var assigned []struct {
		Lessons []struct {
			Variant struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			} `json:"variant"`
			AttemptID *string `json:"attempt_id"`
			Status    string  `json:"status"`
		} `json:"lessons"`
	}
	mustDecode(t, mine.Body, &assigned)
	if len(assigned) != 1 || len(assigned[0].Lessons) != 1 {
		t.Fatalf("assigned = %+v", assigned)
	}
	if assigned[0].Lessons[0].Variant.ID != variantA {
		t.Fatalf("student saw variant %s", assigned[0].Lessons[0].Variant.ID)
	}
	if assigned[0].Lessons[0].Status != "available" || assigned[0].Lessons[0].AttemptID == nil {
		t.Fatalf("grant = %+v", assigned[0].Lessons[0])
	}
	hidden := doRequest(t, handler, http.MethodGet, "/variants/"+variantB+"/tickets", studentToken, nil)
	if hidden.StatusCode != http.StatusForbidden {
		t.Fatalf("other variant tickets = %d", hidden.StatusCode)
	}

	attemptID := *assigned[0].Lessons[0].AttemptID
	startForbidden := doRequest(t, handler, http.MethodPost, "/attempts/"+attemptID+"/start", ownerToken, nil)
	if startForbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("owner start status = %d", startForbidden.StatusCode)
	}
	start := doRequest(t, handler, http.MethodPost, "/attempts/"+attemptID+"/start", studentToken, nil)
	mustOK(t, start)

	wrong := doRequest(t, handler, http.MethodPatch, "/attempts/"+attemptID+"/answers/"+ticketA, studentToken, map[string]any{
		"incident_type_code":   "101",
		"tag_codes":            []string{"where_street"},
		"service_codes":        []string{"sluzhba_101"},
		"applicant_last_name":  "Wrong",
		"applicant_first_name": "Name",
		"caller_number":        "79000000000",
		"dictated_number":      "000",
		"notes":                "draft",
	})
	mustOK(t, wrong)
	submit := doRequest(t, handler, http.MethodPost, "/attempts/"+attemptID+"/submit", studentToken, nil)
	mustOK(t, submit)
	var submitted struct {
		Status string `json:"status"`
		Score  *int   `json:"score"`
		Report struct {
			OverallScore int `json:"overall_score"`
			Items        []struct {
				Errors []struct {
					Field string `json:"field"`
				} `json:"errors"`
			} `json:"items"`
		} `json:"report"`
	}
	mustDecode(t, submit.Body, &submitted)
	if submitted.Status != "submitted" || submitted.Score == nil || *submitted.Score == 100 {
		t.Fatalf("submitted = %+v", submitted)
	}
	if submitted.Report.OverallScore == 0 || len(submitted.Report.Items) != 1 || len(submitted.Report.Items[0].Errors) == 0 {
		t.Fatalf("report = %+v", submitted.Report)
	}

	restart := doRequest(t, handler, http.MethodPost, "/attempts/"+attemptID+"/start", studentToken, nil)
	if restart.StatusCode != http.StatusConflict {
		t.Fatalf("restart = %d %s", restart.StatusCode, restart.Body)
	}

	grantA := doRequest(t, handler, http.MethodPost, "/users/"+studentID+"/attempts", ownerToken, map[string]string{"variant_id": variantA})
	mustOK(t, grantA)
	var granted struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	mustDecode(t, grantA.Body, &granted)
	if granted.Status != "available" {
		t.Fatalf("grant A = %+v", granted)
	}
	mustOK(t, doRequest(t, handler, http.MethodPost, "/attempts/"+granted.ID+"/start", studentToken, nil))
	savePerfect(t, handler, studentToken, granted.ID, ticketA)
	mustOK(t, doRequest(t, handler, http.MethodPost, "/attempts/"+granted.ID+"/submit", studentToken, nil))

	grantB := doRequest(t, handler, http.MethodPost, "/users/"+studentID+"/attempts", ownerToken, map[string]string{"variant_id": variantB})
	mustOK(t, grantB)
	mustDecode(t, grantB.Body, &granted)
	startB := doRequest(t, handler, http.MethodPost, "/attempts/"+granted.ID+"/start", studentToken, nil)
	mustOK(t, startB)
	var startedB struct {
		VariantID string `json:"variant_id"`
		Status    string `json:"status"`
	}
	mustDecode(t, startB.Body, &startedB)
	if startedB.VariantID != variantB || startedB.Status != "in_progress" {
		t.Fatalf("start B = %+v", startedB)
	}
}

func TestTicketsAutoExpire(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	_ = testkit.SeedUser(t, pool, "owner2", "password1", "teacher")
	_ = testkit.SeedUser(t, pool, "stud2", "password1", "student")
	ownerToken := loginToken(t, handler, "owner2", "password1")
	studentToken := loginToken(t, handler, "stud2", "password1")
	studentID := mustFindLogin(t, handler, studentToken)

	topic := createNamed(t, handler, ownerToken, "/topics", map[string]string{"title": "Timed topic"})
	module := createNamed(t, handler, ownerToken, "/modules", map[string]any{"title": "Timed module", "description": ""})
	lessonRes := doRequest(t, handler, http.MethodPost, "/modules/"+module+"/lessons", ownerToken, map[string]any{
		"title": "Timed lesson", "position": 0, "duration_seconds": 1,
	})
	mustOK(t, lessonRes)
	var lesson struct {
		ID string `json:"id"`
	}
	mustDecode(t, lessonRes.Body, &lesson)
	variant := createNamed(t, handler, ownerToken, "/lessons/"+lesson.ID+"/variants", map[string]any{"title": "V", "position": 0})
	ticket := createNamed(t, handler, ownerToken, "/variants/"+variant+"/tickets", map[string]any{
		"topic_id": topic, "title": "Timed", "body": "x",
	})
	mustOK(t, doRequest(t, handler, http.MethodPut, "/tickets/"+ticket+"/reference", ownerToken, map[string]any{
		"incident_type_code": "gratitude",
		"tag_codes":          []string{},
		"service_codes":      []string{},
	}))
	grant := doRequest(t, handler, http.MethodPost, "/users/"+studentID+"/attempts", ownerToken, map[string]string{"variant_id": variant})
	mustOK(t, grant)
	var attempt struct {
		ID string `json:"id"`
	}
	mustDecode(t, grant.Body, &attempt)
	mustOK(t, doRequest(t, handler, http.MethodPost, "/attempts/"+attempt.ID+"/start", studentToken, nil))

	time.Sleep(1100 * time.Millisecond)

	get := doRequest(t, handler, http.MethodGet, "/attempts/"+attempt.ID, studentToken, nil)
	mustOK(t, get)
	var expired struct {
		Status string `json:"status"`
		Score  *int   `json:"score"`
	}
	mustDecode(t, get.Body, &expired)
	if expired.Status != "timed_out" || expired.Score == nil {
		t.Fatalf("expired = %+v", expired)
	}
}

func setRef(t *testing.T, handler http.Handler, token, ticketID string) {
	t.Helper()
	mustOK(t, doRequest(t, handler, http.MethodPut, "/tickets/"+ticketID+"/reference", token, map[string]any{
		"incident_type_code":   "101",
		"tag_codes":            []string{"where_street", "street_flame_smoke", "burn_trash"},
		"service_codes":        []string{"sluzhba_101"},
		"applicant_last_name":  "Ivanov",
		"applicant_first_name": "Ivan",
		"caller_number":        "79001112233",
		"dictated_number":      "101",
	}))
}

func savePerfect(t *testing.T, handler http.Handler, token, attemptID, ticketID string) {
	t.Helper()
	mustOK(t, doRequest(t, handler, http.MethodPatch, "/attempts/"+attemptID+"/answers/"+ticketID, token, map[string]any{
		"incident_type_code":   "101",
		"tag_codes":            []string{"where_street", "street_flame_smoke", "burn_trash"},
		"service_codes":        []string{"sluzhba_101"},
		"applicant_last_name":  "Ivanov",
		"applicant_first_name": "Ivan",
		"caller_number":        "79001112233",
		"dictated_number":      "101",
		"notes":                "",
	}))
}

func createNamed(t *testing.T, handler http.Handler, token, path string, body any) string {
	t.Helper()
	res := doRequest(t, handler, http.MethodPost, path, token, body)
	mustOK(t, res)
	var out struct {
		ID string `json:"id"`
	}
	mustDecode(t, res.Body, &out)
	return out.ID
}

func mustOK(t *testing.T, res httpResult) {
	t.Helper()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		t.Fatalf("status = %d body=%s", res.StatusCode, res.Body)
	}
}

func mustFindLogin(t *testing.T, handler http.Handler, token string) string {
	t.Helper()
	res := doRequest(t, handler, http.MethodGet, "/auth/me", token, nil)
	mustOK(t, res)
	var me struct {
		ID string `json:"id"`
	}
	mustDecode(t, res.Body, &me)
	return me.ID
}

type httpResult struct {
	StatusCode int
	Body       []byte
}

func doRequest(t *testing.T, handler http.Handler, method, path, token string, body any) httpResult {
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
	return httpResult{StatusCode: rec.Code, Body: rec.Body.Bytes()}
}

func mustDecode(t *testing.T, raw []byte, dest any) {
	t.Helper()
	if err := json.Unmarshal(raw, dest); err != nil {
		t.Fatalf("decode %s: %v", raw, err)
	}
}

func loginToken(t *testing.T, handler http.Handler, login, password string) string {
	t.Helper()
	res := doRequest(t, handler, http.MethodPost, "/auth/login", "", map[string]string{
		"login": login, "password": password,
	})
	if res.StatusCode != http.StatusOK {
		t.Fatalf("login %s status = %d body=%s", login, res.StatusCode, res.Body)
	}
	var out struct {
		Token string `json:"token"`
	}
	mustDecode(t, res.Body, &out)
	return out.Token
}
