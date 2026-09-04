package errors

var (
	ErrUserNotFound      = &NotFoundError{"user not found"}
	ErrInvalidUserID     = &NotFoundError{"invalid user id"}
	ErrDuplicateUsername = &ConflictError{"user already taken"}
	ErrDuplicateEmail    = &ConflictError{"email already registered"}
	ErrDuplicateUser     = &ConflictError{"user with that login already exists"}
	ErrIncorrectPassword = &UnauthorizedError{"incorrect password"}
)
