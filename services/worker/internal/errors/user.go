package errors

import "errors"

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrInvalidUserID     = errors.New("invalid user id")
	ErrDuplicateUsername = errors.New("user already taken")
	ErrDuplicateEmail    = errors.New("email already registered")
	ErrDuplicateUser     = errors.New("user with that login already exists")
	ErrIncorrectPassword = errors.New("incorrect password")
)
