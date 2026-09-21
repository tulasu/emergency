package application

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"traineebox/internal/auth/domain/errs"
	"traineebox/internal/auth/domain/value_objects"

	"golang.org/x/crypto/argon2"
)

type PasswordHasher struct{}

func (PasswordHasher) Hash(password string) (value_objects.PasswordHash, error) {
	if len(password) < 8 {
		return "", errs.ErrInvalidInput
	}
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("salt: %w", err)
	}
	hash := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	payload := append(salt, hash...)
	return value_objects.PasswordHash(base64.RawStdEncoding.EncodeToString(payload)), nil
}

func (PasswordHasher) Verify(password string, stored value_objects.PasswordHash) bool {
	raw, err := base64.RawStdEncoding.DecodeString(stored.String())
	if err != nil || len(raw) != 48 {
		return false
	}
	salt, want := raw[:16], raw[16:]
	got := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)
	if len(got) != len(want) {
		return false
	}
	var v byte
	for i := range got {
		v |= got[i] ^ want[i]
	}
	return v == 0
}
