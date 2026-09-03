package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

// DatabaseError wraps a lower-level DB error with the operation that failed.
// Repo layer returns this; it's for logging, not shown to the client directly.
type DatabaseError struct {
	Operation string
	Err       error
}

func (e *DatabaseError) Error() string {
	return fmt.Sprintf("database error during %s: %v", e.Operation, e.Err)
}

func (e *DatabaseError) Unwrap() error {
	return e.Err
}

// NotFoundError signals a missing resource — maps to 404.
type NotFoundError struct {
	Resource string
	ID       string
}

func (e *NotFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.Resource, e.ID)
}

// GRPCError wraps errors from downstream gRPC calls.
type GRPCError struct {
	Method string
	Err    error
}

func (e *GRPCError) Error() string {
	return fmt.Sprintf("grpc error calling %s: %v", e.Method, e.Err)
}

func (e *GRPCError) Unwrap() error {
	return e.Err
}

// HandlerError is for handler-level errors that already know their own
// HTTP status (bad JSON body, missing query param, etc).
type HandlerError struct {
	Status  int
	Message string
}

func (e *HandlerError) Error() string {
	return fmt.Sprintf("handler error (status %d): %s", e.Status, e.Message)
}

// StatusAndBody is the single central place that decides how any error
// gets turned into an HTTP status + client-facing message + optional field errors.
func StatusAndErrorMsg(err error) (statusCode int, messages []string) {
	var (
		valErrs     validator.ValidationErrors
		notFoundErr *NotFoundError
		handlerErr  *HandlerError
	)

	switch {
	case errors.Is(err, ErrUserNotFound):
		return http.StatusNotFound, []string{err.Error()}
	case errors.Is(err, ErrDuplicateEmail), errors.Is(err, ErrDuplicateUsername), errors.Is(err, ErrDuplicateUser):
		return http.StatusConflict, []string{err.Error()}
	case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrExpiredToken):
		return http.StatusUnauthorized, []string{err.Error()}
	case errors.As(err, &handlerErr):
		return handlerErr.Status, []string{handlerErr.Message}
	case errors.As(err, &valErrs):
		messages = make([]string, len(valErrs))
		for i, fe := range valErrs {
			messages[i] = fmt.Sprintf("%s: %s", fe.Field(), fieldMessage(fe))
		}
		return http.StatusBadRequest, messages
	case errors.As(err, &notFoundErr):
		return http.StatusNotFound, []string{notFoundErr.Error()}
	default:
		return http.StatusInternalServerError, []string{"internal server error"}
	}
}

func fieldMessage(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "this field is required"
	case "email":
		return "must be a valid email"
	case "username":
		return "must be of length 3-20 and can contain only alphabets, digits, ., _, -"
	case "password":
		return "must have length of atleast 8 and contain a uppercase, lowercase, digits and symbols"
	case "min":
		return fmt.Sprintf("must be at least %s", fe.Param())
	default:
		return "invalid value"
	}
}
