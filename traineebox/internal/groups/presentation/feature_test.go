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

func TestGroupsFlow(t *testing.T) {
	pool := testkit.StartPostgres(t)
	testkit.Truncate(t, pool)
	handler := testkit.NewAPI(t, pool)

	_ = testkit.SeedUser(t, pool, "admin1", "password1", value_objects.RoleAdmin)
	owner := testkit.SeedUser(t, pool, "owner1", "password1", value_objects.RoleTeacher)
	coTeacher := testkit.SeedUser(t, pool, "coteach1", "password1", value_objects.RoleTeacher)
	student := testkit.SeedUser(t, pool, "stud1", "password1", value_objects.RoleStudent)
	outsider := testkit.SeedUser(t, pool, "stud2", "password1", value_objects.RoleStudent)

	adminToken := loginToken(t, handler, "admin1", "password1")
	ownerToken := loginToken(t, handler, "owner1", "password1")
	coToken := loginToken(t, handler, "coteach1", "password1")
	studentToken := loginToken(t, handler, "stud1", "password1")
	outsiderToken := loginToken(t, handler, "stud2", "password1")

	forbiddenCreate := doRequest(t, handler, http.MethodPost, "/groups", studentToken, map[string]string{"name": "X"})
	if forbiddenCreate.StatusCode != http.StatusForbidden {
		t.Fatalf("student create status = %d body=%s", forbiddenCreate.StatusCode, forbiddenCreate.Body)
	}

	createRes := doRequest(t, handler, http.MethodPost, "/groups", ownerToken, map[string]string{"name": " Cohort A "})
	if createRes.StatusCode != http.StatusOK {
		t.Fatalf("create status = %d body=%s", createRes.StatusCode, createRes.Body)
	}
	var group struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Members []struct {
			UserID string `json:"user_id"`
			Role   string `json:"role"`
		} `json:"members"`
	}
	mustDecode(t, createRes.Body, &group)
	if group.Name != "Cohort A" || len(group.Members) != 1 || group.Members[0].Role != "owner" {
		t.Fatalf("group = %+v", group)
	}

	addTeacher := doRequest(t, handler, http.MethodPost, "/groups/"+group.ID+"/members", ownerToken, map[string]string{
		"user_id": coTeacher.ID.String(), "role": "teacher",
	})
	if addTeacher.StatusCode != http.StatusOK {
		t.Fatalf("add teacher status = %d body=%s", addTeacher.StatusCode, addTeacher.Body)
	}

	addStudentForbidden := doRequest(t, handler, http.MethodPost, "/groups/"+group.ID+"/members", coToken, map[string]string{
		"user_id": outsider.ID.String(), "role": "teacher",
	})
	if addStudentForbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("co-teacher add teacher status = %d body=%s", addStudentForbidden.StatusCode, addStudentForbidden.Body)
	}

	addStudentAsTeacherRole := doRequest(t, handler, http.MethodPost, "/groups/"+group.ID+"/members", ownerToken, map[string]string{
		"user_id": student.ID.String(), "role": "teacher",
	})
	if addStudentAsTeacherRole.StatusCode != http.StatusForbidden {
		t.Fatalf("student as teacher status = %d body=%s", addStudentAsTeacherRole.StatusCode, addStudentAsTeacherRole.Body)
	}

	addStudent := doRequest(t, handler, http.MethodPost, "/groups/"+group.ID+"/members", coToken, map[string]string{
		"user_id": student.ID.String(), "role": "student",
	})
	if addStudent.StatusCode != http.StatusOK {
		t.Fatalf("add student status = %d body=%s", addStudent.StatusCode, addStudent.Body)
	}

	getOk := doRequest(t, handler, http.MethodGet, "/groups/"+group.ID, studentToken, nil)
	if getOk.StatusCode != http.StatusOK {
		t.Fatalf("student get status = %d", getOk.StatusCode)
	}
	getForbidden := doRequest(t, handler, http.MethodGet, "/groups/"+group.ID, outsiderToken, nil)
	if getForbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("outsider get status = %d", getForbidden.StatusCode)
	}

	removeOwner := doRequest(t, handler, http.MethodDelete, "/groups/"+group.ID+"/members/"+owner.ID.String(), ownerToken, nil)
	if removeOwner.StatusCode != http.StatusForbidden {
		t.Fatalf("remove last owner status = %d body=%s", removeOwner.StatusCode, removeOwner.Body)
	}

	rename := doRequest(t, handler, http.MethodPatch, "/groups/"+group.ID, ownerToken, map[string]string{"name": "Cohort B"})
	if rename.StatusCode != http.StatusOK {
		t.Fatalf("rename status = %d body=%s", rename.StatusCode, rename.Body)
	}

	adminList := doRequest(t, handler, http.MethodGet, "/groups", adminToken, nil)
	if adminList.StatusCode != http.StatusOK {
		t.Fatalf("admin list status = %d", adminList.StatusCode)
	}

	delForbidden := doRequest(t, handler, http.MethodDelete, "/groups/"+group.ID, coToken, nil)
	if delForbidden.StatusCode != http.StatusForbidden {
		t.Fatalf("co-teacher delete status = %d", delForbidden.StatusCode)
	}

	del := doRequest(t, handler, http.MethodDelete, "/groups/"+group.ID, ownerToken, nil)
	if del.StatusCode != http.StatusOK {
		t.Fatalf("delete status = %d body=%s", del.StatusCode, del.Body)
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
