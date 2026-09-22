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
)

func TestHealth(t *testing.T) {
	pool := testkit.StartPostgres(t)
	handler := testkit.NewAPI(t, pool)

	res := doRequest(t, handler, http.MethodGet, "/health", "", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status = %d", res.StatusCode)
	}
}

func TestAuthFlow(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	admin := testkit.SeedUser(t, pool, "admin", "password1", value_objects.RoleAdmin)

	loginRes := doRequest(t, handler, http.MethodPost, "/auth/login", "", map[string]string{
		"login": "admin", "password": "password1",
	})
	if loginRes.StatusCode != http.StatusOK {
		t.Fatalf("login status = %d body=%s", loginRes.StatusCode, loginRes.Body)
	}
	var loginOut struct {
		Token string `json:"token"`
		User  struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	mustDecode(t, loginRes.Body, &loginOut)
	if loginOut.Token == "" || loginOut.User.ID != admin.ID.String() {
		t.Fatalf("login out = %+v", loginOut)
	}

	meRes := doRequest(t, handler, http.MethodGet, "/auth/me", loginOut.Token, nil)
	if meRes.StatusCode != http.StatusOK {
		t.Fatalf("me status = %d body=%s", meRes.StatusCode, meRes.Body)
	}

	logoutRes := doRequest(t, handler, http.MethodPost, "/auth/logout", loginOut.Token, nil)
	if logoutRes.StatusCode != http.StatusOK {
		t.Fatalf("logout status = %d", logoutRes.StatusCode)
	}

	meAfter := doRequest(t, handler, http.MethodGet, "/auth/me", loginOut.Token, nil)
	if meAfter.StatusCode != http.StatusUnauthorized {
		t.Fatalf("me after logout status = %d", meAfter.StatusCode)
	}
}

func TestBlockedUser(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	u := testkit.SeedUser(t, pool, "blocked", "password1", value_objects.RoleStudent)
	testkit.SetBlocked(t, pool, u.ID, true)

	loginRes := doRequest(t, handler, http.MethodPost, "/auth/login", "", map[string]string{
		"login": "blocked", "password": "password1",
	})
	if loginRes.StatusCode != http.StatusForbidden {
		t.Fatalf("login status = %d body=%s", loginRes.StatusCode, loginRes.Body)
	}
}

func TestAdminRBAC(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	_ = testkit.SeedUser(t, pool, "admin", "password1", value_objects.RoleAdmin)
	teacher := testkit.SeedUser(t, pool, "teacher", "password1", value_objects.RoleTeacher)

	adminToken := loginToken(t, handler, "admin", "password1")
	teacherToken := loginToken(t, handler, "teacher", "password1")

	createBody := map[string]string{
		"login": "newbie", "password": "password1", "role": "student",
	}

	unauth := doRequest(t, handler, http.MethodPost, "/auth/users", "", createBody)
	if unauth.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unauth create status = %d", unauth.StatusCode)
	}

	forbidden := doRequest(t, handler, http.MethodPost, "/auth/users", teacherToken, createBody)
	if forbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("teacher create status = %d body=%s", forbidden.StatusCode, forbidden.Body)
	}

	created := doRequest(t, handler, http.MethodPost, "/auth/users", adminToken, createBody)
	if created.StatusCode != http.StatusOK {
		t.Fatalf("admin create status = %d body=%s", created.StatusCode, created.Body)
	}
	var createdUser struct {
		ID string `json:"id"`
	}
	mustDecode(t, created.Body, &createdUser)

	dup := doRequest(t, handler, http.MethodPost, "/auth/users", adminToken, createBody)
	if dup.StatusCode != http.StatusConflict {
		t.Fatalf("dup status = %d body=%s", dup.StatusCode, dup.Body)
	}

	blockRes := doRequest(t, handler, http.MethodPost, "/auth/users/"+teacher.ID.String()+"/block", adminToken, map[string]bool{
		"blocked": true,
	})
	if blockRes.StatusCode != http.StatusOK {
		t.Fatalf("block status = %d body=%s", blockRes.StatusCode, blockRes.Body)
	}

	roleRes := doRequest(t, handler, http.MethodPost, "/auth/users/"+createdUser.ID+"/role", adminToken, map[string]string{
		"role": "teacher",
	})
	if roleRes.StatusCode != http.StatusOK {
		t.Fatalf("role status = %d body=%s", roleRes.StatusCode, roleRes.Body)
	}
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
