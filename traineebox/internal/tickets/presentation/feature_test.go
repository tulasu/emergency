package presentation_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"traineebox/internal/auth/domain/value_objects"
	"traineebox/internal/testkit"
	ticketsinfra "traineebox/internal/tickets/infrastructure"
)

func TestTicketsAttemptFlow(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	catalog := ticketsinfra.NewCatalogRepository(pool)
	ctx := t.Context()
	fire, err := catalog.UpsertIncidentType(ctx, "fire", "Пожар")
	if err != nil {
		t.Fatal(err)
	}
	tag, err := catalog.UpsertIncidentTag(ctx, fire.ID, "building", "Здание")
	if err != nil {
		t.Fatal(err)
	}
	svc, err := catalog.UpsertService(ctx, "fire_service", "Пожарная")
	if err != nil {
		t.Fatal(err)
	}

	_ = testkit.SeedUser(t, pool, "owner1", "password1", value_objects.RoleTeacher)
	_ = testkit.SeedUser(t, pool, "stud1", "password1", value_objects.RoleStudent)
	ownerToken := loginToken(t, handler, "owner1", "password1")
	studentToken := loginToken(t, handler, "stud1", "password1")

	createGroup := doRequest(t, handler, http.MethodPost, "/groups", ownerToken, map[string]string{"name": "Cohort"})
	if createGroup.StatusCode != http.StatusOK {
		t.Fatalf("create group %d %s", createGroup.StatusCode, createGroup.Body)
	}
	var group struct {
		ID string `json:"id"`
	}
	mustDecode(t, createGroup.Body, &group)

	student := mustFindLogin(t, handler, studentToken)
	addStud := doRequest(t, handler, http.MethodPost, "/groups/"+group.ID+"/members", ownerToken, map[string]string{
		"user_id": student, "role": "student",
	})
	if addStud.StatusCode != http.StatusOK {
		t.Fatalf("add student %d %s", addStud.StatusCode, addStud.Body)
	}

	dur := 3600
	createTicket := doRequest(t, handler, http.MethodPost, "/groups/"+group.ID+"/tickets", ownerToken, map[string]any{
		"title": "Ticket 1", "body": "Call about fire", "max_attempts": 2, "duration_seconds": dur,
	})
	if createTicket.StatusCode != http.StatusOK {
		t.Fatalf("create ticket %d %s", createTicket.StatusCode, createTicket.Body)
	}
	var ticket struct {
		ID string `json:"id"`
	}
	mustDecode(t, createTicket.Body, &ticket)

	ref := doRequest(t, handler, http.MethodPut, "/tickets/"+ticket.ID+"/reference", ownerToken, map[string]any{
		"incident_type_id":     fire.ID.String(),
		"tag_ids":              []string{tag.ID.String()},
		"service_ids":          []string{svc.ID.String()},
		"applicant_last_name":  "Ivanov",
		"applicant_first_name": "Ivan",
		"caller_number":        "79001112233",
		"dictated_number":      "101",
	})
	if ref.StatusCode != http.StatusOK {
		t.Fatalf("set reference %d %s", ref.StatusCode, ref.Body)
	}

	startForbidden := doRequest(t, handler, http.MethodPost, "/tickets/"+ticket.ID+"/attempts", ownerToken, nil)
	if startForbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("owner start status = %d", startForbidden.StatusCode)
	}

	start := doRequest(t, handler, http.MethodPost, "/tickets/"+ticket.ID+"/attempts", studentToken, nil)
	if start.StatusCode != http.StatusOK {
		t.Fatalf("start %d %s", start.StatusCode, start.Body)
	}
	var attempt struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	mustDecode(t, start.Body, &attempt)
	if attempt.Status != "in_progress" {
		t.Fatalf("status = %s", attempt.Status)
	}

	save := doRequest(t, handler, http.MethodPatch, "/attempts/"+attempt.ID+"/answer", studentToken, map[string]any{
		"incident_type_id":     fire.ID.String(),
		"tag_ids":              []string{tag.ID.String()},
		"service_ids":          []string{svc.ID.String()},
		"applicant_last_name":  "Ivanov",
		"applicant_first_name": "Ivan",
		"caller_number":        "79001112233",
		"dictated_number":      "101",
		"notes":                "draft note",
	})
	if save.StatusCode != http.StatusOK {
		t.Fatalf("save %d %s", save.StatusCode, save.Body)
	}

	resume := doRequest(t, handler, http.MethodGet, "/attempts/"+attempt.ID, studentToken, nil)
	if resume.StatusCode != http.StatusOK {
		t.Fatalf("resume %d %s", resume.StatusCode, resume.Body)
	}
	var resumed struct {
		Answer struct {
			Notes string `json:"notes"`
		} `json:"answer"`
	}
	mustDecode(t, resume.Body, &resumed)
	if resumed.Answer.Notes != "draft note" {
		t.Fatalf("notes = %q", resumed.Answer.Notes)
	}

	submit := doRequest(t, handler, http.MethodPost, "/attempts/"+attempt.ID+"/submit", studentToken, nil)
	if submit.StatusCode != http.StatusOK {
		t.Fatalf("submit %d %s", submit.StatusCode, submit.Body)
	}
	var submitted struct {
		Status string `json:"status"`
		Score  *int   `json:"score"`
	}
	mustDecode(t, submit.Body, &submitted)
	if submitted.Status != "submitted" || submitted.Score == nil || *submitted.Score != 100 {
		t.Fatalf("submitted = %+v", submitted)
	}
}

func TestTicketsAutoExpire(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	catalog := ticketsinfra.NewCatalogRepository(pool)
	ctx := t.Context()
	fire, err := catalog.UpsertIncidentType(ctx, "fire", "Пожар")
	if err != nil {
		t.Fatal(err)
	}

	_ = testkit.SeedUser(t, pool, "owner2", "password1", value_objects.RoleTeacher)
	_ = testkit.SeedUser(t, pool, "stud2", "password1", value_objects.RoleStudent)
	ownerToken := loginToken(t, handler, "owner2", "password1")
	studentToken := loginToken(t, handler, "stud2", "password1")

	createGroup := doRequest(t, handler, http.MethodPost, "/groups", ownerToken, map[string]string{"name": "Short"})
	mustOK(t, createGroup)
	var group struct {
		ID string `json:"id"`
	}
	mustDecode(t, createGroup.Body, &group)
	studentID := mustFindLogin(t, handler, studentToken)
	mustOK(t, doRequest(t, handler, http.MethodPost, "/groups/"+group.ID+"/members", ownerToken, map[string]string{
		"user_id": studentID, "role": "student",
	}))

	createTicket := doRequest(t, handler, http.MethodPost, "/groups/"+group.ID+"/tickets", ownerToken, map[string]any{
		"title": "Timed", "body": "x", "duration_seconds": 1,
	})
	mustOK(t, createTicket)
	var ticket struct {
		ID string `json:"id"`
	}
	mustDecode(t, createTicket.Body, &ticket)

	mustOK(t, doRequest(t, handler, http.MethodPut, "/tickets/"+ticket.ID+"/reference", ownerToken, map[string]any{
		"incident_type_id": fire.ID.String(),
		"tag_ids":          []string{},
		"service_ids":      []string{},
	}))

	start := doRequest(t, handler, http.MethodPost, "/tickets/"+ticket.ID+"/attempts", studentToken, nil)
	mustOK(t, start)
	var attempt struct {
		ID string `json:"id"`
	}
	mustDecode(t, start.Body, &attempt)

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

func mustOK(t *testing.T, res httpResult) {
	t.Helper()
	if res.StatusCode != http.StatusOK {
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
