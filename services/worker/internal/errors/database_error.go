package errors

const (
	ErrUniqueViolation     = "23505"
	ErrForeignKeyViolation = "23503"
	ErrNotNullViolation    = "23502"
	ErrCheckViolation      = "23514"
	ErrExclusionViolation  = "23P01"

	ErrInvalidTextRepresentation = "22P02"
	ErrStringDataTruncation      = "22001"
	ErrDivisionByZero            = "22012"
	ErrNumericValueOutOfRange    = "22003"

	ErrSerializationFailure = "40001"
	ErrDeadlockDetected     = "40P01"

	ErrSyntaxError           = "42601"
	ErrUndefinedTable        = "42P01"
	ErrUndefinedColumn       = "42703"
	ErrInsufficientPrivilege = "42501"
	ErrDuplicateTable        = "42P07"

	ErrConnectionFailure      = "08006"
	ErrConnectionDoesNotExist = "08003"

	ErrTooManyConnections = "53300"
	ErrOutOfMemory        = "53200"

	ErrQueryCanceled = "57014"
	ErrAdminShutdown = "57P01"
)
