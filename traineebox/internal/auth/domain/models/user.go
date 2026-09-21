package models

import (
	"time"

	"traineebox/internal/auth/domain/value_objects"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Login        value_objects.Login
	PasswordHash value_objects.PasswordHash
	Role         value_objects.Role
	BlockedAt    *time.Time
	CreatedAt    time.Time
}

func (u User) IsBlocked() bool {
	return u.BlockedAt != nil
}
