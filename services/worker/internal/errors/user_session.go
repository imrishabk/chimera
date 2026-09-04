package errors

import "errors"

var (
	ErrTokenNotFound = errors.New("session token not found")
	ErrInvalidToken  = errors.New("invalid token")
	ErrExpiredToken  = errors.New("token expired")
)
