package errors

import "fmt"

// Custom error types for the application

type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("validation error on field %s: %s", e.Field, e.Message)
}

type DatabaseError struct {
	Operation string
	Err       error
}

func (e DatabaseError) Error() string {
	return fmt.Sprintf("database error during %s: %v", e.Operation, e.Err)
}

func (e DatabaseError) Unwrap() error {
	return e.Err
}

type GRPCError struct {
	Method string
	Err    error
}

func (e GRPCError) Error() string {
	return fmt.Sprintf("grpc error calling %s: %v", e.Method, e.Err)
}

func (e GRPCError) Unwrap() error {
	return e.Err
}

type HandlerError struct {
	Status  int
	Message string
}

func (e HandlerError) Error() string {
	return fmt.Sprintf("handler error (status %d): %s", e.Status, e.Message)
}
