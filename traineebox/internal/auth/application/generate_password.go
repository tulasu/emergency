package application

import (
	"crypto/rand"
	"fmt"
)

const generatedPasswordLen = 10

func generatePassword() (string, error) {
	const letters = "abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	buf := make([]byte, generatedPasswordLen)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("password: %w", err)
	}
	for i := range buf {
		buf[i] = letters[int(buf[i])%len(letters)]
	}
	return string(buf), nil
}
