package pkg

type ErrorCode string

const (
	UnknownErrorCode ErrorCode = "UNKNOWN_ERROR"
)

type Error struct {
	Code    ErrorCode
	Message string
	Error   error
}

func NewError(code ErrorCode, message string, err error) *Error {
	return &Error{
		Code:    code,
		Message: message,
		Error:   err,
	}
}
