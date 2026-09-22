package application

import (
	"testing"

	"traineebox/internal/auth/domain/errs"
)

func TestPasswordHasher(t *testing.T) {
	t.Parallel()
	h := PasswordHasher{}

	t.Run("too short", func(t *testing.T) {
		t.Parallel()
		_, err := h.Hash("short")
		if err != errs.ErrInvalidInput {
			t.Fatalf("err = %v, want %v", err, errs.ErrInvalidInput)
		}
	})

	t.Run("roundtrip", func(t *testing.T) {
		t.Parallel()
		hash, err := h.Hash("password1")
		if err != nil {
			t.Fatal(err)
		}
		if !h.Verify("password1", hash) {
			t.Fatal("expected verify success")
		}
		if h.Verify("password2", hash) {
			t.Fatal("expected verify failure for wrong password")
		}
	})

	t.Run("corrupt hash", func(t *testing.T) {
		t.Parallel()
		if h.Verify("password1", "not-valid-base64!!!") {
			t.Fatal("expected false for corrupt hash")
		}
		if h.Verify("password1", "YWJj") {
			t.Fatal("expected false for truncated hash")
		}
	})
}
