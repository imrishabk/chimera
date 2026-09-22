package repo

const (
	errUniqueViolation     = "23505"
	errForeignKeyViolation = "23503"
	errNotNullViolation    = "23502"
	errCheckViolation      = "23514"
	errExclusionViolation  = "23P01"

	errInvalidTextRepresentation = "22P02"
	errStringDataTruncation      = "22001"
	errDivisionByZero            = "22012"
	errNumericValueOutOfRange    = "22003"

	errSerializationFailure = "40001"
	errDeadlockDetected     = "40P01"

	errSyntaxError           = "42601"
	errUndefinedTable        = "42P01"
	errUndefinedColumn       = "42703"
	errInsufficientPrivilege = "42501"
	errDuplicateTable        = "42P07"

	errConnectionFailure      = "08006"
	errConnectionDoesNotExist = "08003"

	errTooManyConnections = "53300"
	errOutOfMemory        = "53200"

	errQueryCanceled = "57014"
	errAdminShutdown = "57P01"
)
