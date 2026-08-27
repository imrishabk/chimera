package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func ConstructDSN() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		os.Getenv("DB_USERNAME"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_HOSTNAME"),
		os.Getenv("DB_PORT"),
		os.Getenv("DB_DATABASE"))
}

func CreateUserSession() (string, error) {
	tokenBytes, err := strconv.Atoi(os.Getenv("USER_SESSION_TOKEN_BYTES"))
	if err != nil {
		return "", err
	}

	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("Failed to generate session id: %w", err)
	}

	s := base64.RawURLEncoding.EncodeToString(b)
	return s, nil
}

func HashPassword(p string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(p), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(b), err
}

func VerifyPassword(password, hash string) bool {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return false
	}
	return true
}
