package errors

var (
	// Default Errors
	ErrInvalidBody  = &BadRequestError{"invalid body"}
	ErrInvalidParam = &BadRequestError{"invalid parameter"}
	ErrInvalidQuery = &BadRequestError{"invalid query"}

	// Users
	ErrUserNotFound      = &NotFoundError{"user not found"}
	ErrInvalidUser       = &BadRequestError{"invalid user"}
	ErrDuplicateUsername = &ConflictError{"user already taken"}
	ErrDuplicateEmail    = &ConflictError{"email already registered"}
	ErrDuplicateUser     = &ConflictError{"user with that login already exists"}
	ErrIncorrectPassword = &UnauthorizedError{"incorrect password"}
	ErrSamePassword      = &BadRequestError{"old and new password cannot be same"}

	// Sessions
	ErrInvalidSession  = &BadRequestError{"invalid session"}
	ErrSessionNotFound = &NotFoundError{"session not found"}

	// Token
	ErrTokenNotFound = &UnauthorizedError{"session token not found"}
	ErrInvalidToken  = &UnauthorizedError{"invalid token"}
	ErrExpiredToken  = &UnauthorizedError{"token expired"}

	// Ingestion
	ErrFailedIngestRequest        = &BadGatewayError{"failed to send request to ingestion service"}
	ErrInvalidJobParam            = &BadRequestError{"invalid job parameter"}
	ErrIngestionSeviceUnavailable = &ServiceUnavailableError{"ingestion job tracking not available"}

	// IngestionJob
	ErrIngestionJobNotFound = &NotFoundError{"ingestion job not found"}
)
