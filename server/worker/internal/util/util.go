package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"strconv"

	"golang.org/x/crypto/bcrypt"
)

func CreateUserSession() (string, error) {
	tokenBytes, err := strconv.Atoi(os.Getenv("USER_SESSION_TOKEN_BYTES"))
	if err != nil {
		return "", err
	}

	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate session id: %w", err)
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
