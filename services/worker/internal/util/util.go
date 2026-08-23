package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"
)

func ConstructDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"))
}

func CreateUserSession() (*string, error) {
	tokenBytes, err := strconv.Atoi(os.Getenv("USER_SESSION_TOKEN_BYTE"))
	if err != nil {
		return nil, err
	}

	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("Failed to generate session id: %w", err)
	}

	s := base64.RawURLEncoding.EncodeToString(b)
	return &s, nil
}
