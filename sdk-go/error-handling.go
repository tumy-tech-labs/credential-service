package sdk

import (
	"fmt"
)

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API Error: %d - %s", e.StatusCode, e.Message)
}

func HandleError(statusCode int, body []byte) error {
	return &APIError{
		StatusCode: statusCode,
		Message:    string(body),
	}
}
