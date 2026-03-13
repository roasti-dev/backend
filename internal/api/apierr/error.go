package apierr

import "fmt"

type ApiError struct {
	Status  int
	Message string
}

func NewApiError(status int, msg string) *ApiError {
	return &ApiError{
		Status:  status,
		Message: msg,
	}
}

func (e *ApiError) Error() string {
	return fmt.Sprintf("%d: %s", e.Status, e.Message)
}
