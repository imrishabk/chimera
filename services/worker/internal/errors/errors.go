package errors

import (
	"errors"
	"fmt"
	"net/http"

	"github.com/go-playground/validator/v10"
)

type HTTPError interface {
	error
	StatusCode() int
	Messages() []string
}

type NotFoundError struct {
	msg string
}

func (e *NotFoundError) Error() string      { return "not found" }
func (e *NotFoundError) StatusCode() int    { return http.StatusNotFound }
func (e *NotFoundError) Messages() []string { return []string{e.msg} }

type ConflictError struct {
	msg string
}

func (e *ConflictError) Error() string      { return "conflicting resource" }
func (e *ConflictError) StatusCode() int    { return http.StatusConflict }
func (e *ConflictError) Messages() []string { return []string{e.msg} }

type UnauthorizedError struct {
	msg string
}

func (e *UnauthorizedError) Error() string      { return e.msg }
func (e *UnauthorizedError) StatusCode() int    { return http.StatusUnauthorized }
func (e *UnauthorizedError) Messages() []string { return []string{e.msg} }

type ValidationError struct {
	Fields validator.ValidationErrors
}

func (e *ValidationError) Error() string   { return "validation failed" }
func (e *ValidationError) StatusCode() int { return http.StatusBadRequest }
func (e *ValidationError) Messages() []string {
	msgs := make([]string, len(e.Fields))
	for i, fe := range e.Fields {
		msgs[i] = fmt.Sprintf("%s: %s", fe.Field(), fieldMessage(fe))
	}
	return msgs
}

// DatabaseError wraps a lower-level DB error with the operation that failed.
// Repo layer returns this; it's for logging, not shown to the client directly.
type DatabaseError struct {
	Operation string
	Err       error
}

func (e *DatabaseError) Error() string      { return "something went wrong with database" }
func (e *DatabaseError) StatusCode() int    { return http.StatusInternalServerError }
func (e *DatabaseError) Messages() []string { return []string{"internal server error"} }

type BadRequestError struct {
	msg string
}

func (e *BadRequestError) Error() string      { return "bad request" }
func (e *BadRequestError) StatusCode() int    { return http.StatusBadRequest }
func (e *BadRequestError) Messages() []string { return []string{e.msg} }

type BadGatewayError struct {
	msg string
}

func (e *BadGatewayError) Error() string      { return "bad gateway" }
func (e *BadGatewayError) StatusCode() int    { return http.StatusBadGateway }
func (e *BadGatewayError) Messages() []string { return []string{e.msg} }

type ServiceUnavailableError struct {
	msg string
}

func (e *ServiceUnavailableError) Error() string      { return "service unavailable" }
func (e *ServiceUnavailableError) StatusCode() int    { return http.StatusServiceUnavailable }
func (e *ServiceUnavailableError) Messages() []string { return []string{e.msg} }

// StatusAndBody is the single central place that decides how any error
// gets turned into an HTTP status + client-facing message + optional field errors.
//
//	func StatusAndErrorMsg(err error) (statusCode int, messages []string) {
//		var (
//			valErrs     validator.ValidationErrors
//			notFoundErr *NotFoundError
//			handlerErr  *HandlerError
//		)
//
//		switch {
//		case errors.Is(err, ErrUserNotFound), errors.Is(err, ErrTokenNotFound):
//			return http.StatusNotFound, []string{err.Error()}
//		case errors.Is(err, ErrDuplicateEmail),
//			errors.Is(err, ErrDuplicateUsername),
//			errors.Is(err, ErrDuplicateUser):
//			return http.StatusConflict, []string{err.Error()}
//		case errors.Is(err, ErrInvalidToken), errors.Is(err, ErrExpiredToken):
//			return http.StatusUnauthorized, []string{err.Error()}
//		case errors.As(err, &handlerErr):
//			return handlerErr.Status, []string{handlerErr.Message}
//		case errors.As(err, &valErrs):
//			messages = make([]string, len(valErrs))
//			for i, fe := range valErrs {
//				messages[i] = fmt.Sprintf("%s: %s", fe.Field(), fieldMessage(fe))
//			}
//			return http.StatusBadRequest, messages
//		case errors.As(err, &notFoundErr):
//			return http.StatusNotFound, []string{notFoundErr.Error()}
//		default:
//			return http.StatusInternalServerError, []string{"internal server error"}
//		}
//	}
//
// Following Open-Close Principle for Error Handlers
func StatusAndErrorMsg(err error) (int, []string) {
	var httpErr HTTPError
	if errors.As(err, &httpErr) {
		return httpErr.StatusCode(), httpErr.Messages()
	}
	return http.StatusInternalServerError, []string{"internal server error"}
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
