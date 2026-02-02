package httpx

var (
	MsgErrGeneric            = "Internal server error"
	MsgErrNotFound           = "Not found"
	MsgErrBadRequest         = "Invalid request"
	MsgErrRequired           = "Value is required"
	MsgErrDuplicate          = "Value already exists"
	MsgErrTooShort           = "Value must be at least %s characters"
	MsgErrTooLong            = "Value must be less than %s characters"
	MsgErrInvalid            = "Invalid value"
	MsgErrMismatch           = "Values do not match"
	MsgErrBadCredentials     = "Invalid email or password"
	MsgErrJobQueueFull       = "Job queue is full"
	MsgErrWorkersUnavailable = "No available workers"

	MsgSuccessGeneric     = "Action completed successfully"
	MsgSuccessMessageSent = "Message sent"
)
