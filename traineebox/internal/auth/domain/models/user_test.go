package models_test

import (
	"testing"
	"time"

	"traineebox/internal/auth/domain/models"
)

func TestUserIsBlocked(t *testing.T) {
	t.Parallel()

	if (models.User{}).IsBlocked() {
		t.Fatal("nil BlockedAt should not be blocked")
	}

	now := time.Now().UTC()
	if !(models.User{BlockedAt: &now}).IsBlocked() {
		t.Fatal("non-nil BlockedAt should be blocked")
	}
}
