package presentation_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"traineebox/internal/auth/domain/value_objects"
	"traineebox/internal/testkit"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGenerationJobApproveFlow(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	_ = testkit.SeedUser(t, pool, "genowner", "password1", value_objects.RoleTeacher)
	_ = testkit.SeedUser(t, pool, "genstud", "password1", value_objects.RoleStudent)
	ownerToken := genLogin(t, handler, "genowner", "password1")
	studentToken := genLogin(t, handler, "genstud", "password1")

	createGroup := genDo(t, handler, http.MethodPost, "/groups", ownerToken, map[string]string{"name": "Gen"})
	mustGenOK(t, createGroup)
	var group struct {
		ID string `json:"id"`
	}
	mustGenDecode(t, createGroup.Body, &group)

	studentID := mustGenFindLogin(t, handler, studentToken)
	mustGenOK(t, genDo(t, handler, http.MethodPost, "/groups/"+group.ID+"/members", ownerToken, map[string]string{
		"user_id": studentID, "role": "student",
	}))

	createJob := genDo(t, handler, http.MethodPost, "/groups/"+group.ID+"/generation-jobs", ownerToken, map[string]string{
		"prompt": "пожар",
	})
	mustGenOK(t, createJob)
	var job struct {
		ID     string `json:"id"`
		Status string `json:"status"`
	}
	mustGenDecode(t, createJob.Body, &job)
	if job.Status != "queued" {
		t.Fatalf("status=%s", job.Status)
	}

	// student cannot create
	forbidden := genDo(t, handler, http.MethodPost, "/groups/"+group.ID+"/generation-jobs", studentToken, map[string]string{
		"prompt": "x",
	})
	if forbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("student create = %d", forbidden.StatusCode)
	}

	// simulate worker filling draft to ready
	seedReadyJob(t, pool, job.ID)

	getJob := genDo(t, handler, http.MethodGet, "/generation-jobs/"+job.ID, ownerToken, nil)
	mustGenOK(t, getJob)

	patch := genDo(t, handler, http.MethodPatch, "/generation-jobs/"+job.ID, ownerToken, map[string]any{
		"draft_title":   "Пожар на улице",
		"scenario_text": "Горит мусорный бак у дома 5.",
		"draft_reference": map[string]any{
			"incident_type_code":   "101",
			"tag_codes":            []string{"where_street", "street_flame_smoke", "burn_trash"},
			"service_codes":        []string{"sluzhba_101"},
			"applicant_last_name":  "Petrov",
			"applicant_first_name": "Petr",
			"caller_number":        "79001112233",
			"dictated_number":      "79001112233",
		},
	})
	mustGenOK(t, patch)

	approve := genDo(t, handler, http.MethodPost, "/generation-jobs/"+job.ID+"/approve", ownerToken, nil)
	mustGenOK(t, approve)
	var published struct {
		Status            string  `json:"status"`
		PublishedTicketID *string `json:"published_ticket_id"`
	}
	mustGenDecode(t, approve.Body, &published)
	if published.Status != "published" || published.PublishedTicketID == nil {
		t.Fatalf("approve = %+v", published)
	}

	start := genDo(t, handler, http.MethodPost, "/tickets/"+*published.PublishedTicketID+"/attempts", studentToken, nil)
	mustGenOK(t, start)
	var attempt struct {
		ID string `json:"id"`
	}
	mustGenDecode(t, start.Body, &attempt)

	save := genDo(t, handler, http.MethodPatch, "/attempts/"+attempt.ID+"/answer", studentToken, map[string]any{
		"incident_type_code":   "101",
		"tag_codes":            []string{"where_street", "street_flame_smoke", "burn_trash"},
		"service_codes":        []string{"sluzhba_101"},
		"applicant_last_name":  "Petrov",
		"applicant_first_name": "Petr",
		"caller_number":        "79001112233",
		"dictated_number":      "79001112233",
		"notes":                "",
	})
	mustGenOK(t, save)
	submit := genDo(t, handler, http.MethodPost, "/attempts/"+attempt.ID+"/submit", studentToken, nil)
	mustGenOK(t, submit)
	var scored struct {
		Score *int `json:"score"`
	}
	mustGenDecode(t, submit.Body, &scored)
	if scored.Score == nil || *scored.Score != 100 {
		t.Fatalf("score=%v", scored.Score)
	}
}

func TestGenerationJobRetryCancel(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	_ = testkit.SeedUser(t, pool, "genowner2", "password1", value_objects.RoleTeacher)
	ownerToken := genLogin(t, handler, "genowner2", "password1")

	createGroup := genDo(t, handler, http.MethodPost, "/groups", ownerToken, map[string]string{"name": "Gen2"})
	mustGenOK(t, createGroup)
	var group struct {
		ID string `json:"id"`
	}
	mustGenDecode(t, createGroup.Body, &group)

	createJob := genDo(t, handler, http.MethodPost, "/groups/"+group.ID+"/generation-jobs", ownerToken, map[string]string{
		"prompt": "дтп",
	})
	mustGenOK(t, createJob)
	var job struct {
		ID string `json:"id"`
	}
	mustGenDecode(t, createJob.Body, &job)

	seedFailedJob(t, pool, job.ID)

	retry := genDo(t, handler, http.MethodPost, "/generation-jobs/"+job.ID+"/retry", ownerToken, nil)
	mustGenOK(t, retry)
	var retried struct {
		Status string `json:"status"`
	}
	mustGenDecode(t, retry.Body, &retried)
	if retried.Status != "queued" {
		t.Fatalf("retry status=%s", retried.Status)
	}

	cancel := genDo(t, handler, http.MethodPost, "/generation-jobs/"+job.ID+"/cancel", ownerToken, nil)
	mustGenOK(t, cancel)
	var cancelled struct {
		Status string `json:"status"`
	}
	mustGenDecode(t, cancel.Body, &cancelled)
	if cancelled.Status != "cancelled" {
		t.Fatalf("cancel status=%s", cancelled.Status)
	}

	del := genDo(t, handler, http.MethodDelete, "/generation-jobs/"+job.ID, ownerToken, nil)
	mustGenOK(t, del)
}

func seedReadyJob(t *testing.T, pool *pgxpool.Pool, jobID string) {
	t.Helper()
	id, err := uuid.Parse(jobID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(t.Context(), `
		UPDATE ticket_generation_jobs SET
			status = 'ready',
			version = version + 1,
			draft_title = 'Пожар',
			scenario_text = 'Горит бак',
			draft_reference = $2::jsonb,
			updated_at = now()
		WHERE id = $1`, id, `{
			"incident_type_code":"101",
			"tag_codes":["where_street","street_flame_smoke","burn_trash"],
			"service_codes":["sluzhba_101"],
			"applicant_last_name":"Petrov",
			"applicant_first_name":"Petr",
			"caller_number":"79001112233",
			"dictated_number":"79001112233"
		}`)
	if err != nil {
		t.Fatal(err)
	}
}

func seedFailedJob(t *testing.T, pool *pgxpool.Pool, jobID string) {
	t.Helper()
	id, err := uuid.Parse(jobID)
	if err != nil {
		t.Fatal(err)
	}
	_, err = pool.Exec(t.Context(), `
		UPDATE ticket_generation_jobs SET status='failed', error_message='llm down', version=version+1, updated_at=now()
		WHERE id=$1`, id)
	if err != nil {
		t.Fatal(err)
	}
}

type genResp struct {
	StatusCode int
	Body       string
}

func genDo(t *testing.T, handler http.Handler, method, path, token string, body any) genResp {
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
	return genResp{StatusCode: rec.Code, Body: rec.Body.String()}
}

func genLogin(t *testing.T, handler http.Handler, login, password string) string {
	t.Helper()
	res := genDo(t, handler, http.MethodPost, "/auth/login", "", map[string]string{
		"login": login, "password": password,
	})
	mustGenOK(t, res)
	var out struct {
		Token string `json:"token"`
	}
	mustGenDecode(t, res.Body, &out)
	return out.Token
}

func mustGenOK(t *testing.T, res genResp) {
	t.Helper()
	if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusNoContent {
		t.Fatalf("status=%d body=%s", res.StatusCode, res.Body)
	}
}

func mustGenDecode(t *testing.T, body string, v any) {
	t.Helper()
	if err := json.Unmarshal([]byte(body), v); err != nil {
		t.Fatalf("decode %v body=%s", err, body)
	}
}

func mustGenFindLogin(t *testing.T, handler http.Handler, token string) string {
	t.Helper()
	res := genDo(t, handler, http.MethodGet, "/auth/me", token, nil)
	mustGenOK(t, res)
	var me struct {
		ID string `json:"id"`
	}
	mustGenDecode(t, res.Body, &me)
	return me.ID
}
