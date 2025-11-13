package errors

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) WriteHTTPResponse(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.Code)
	json.NewEncoder(w).Encode(map[string]string{
		"error": e.Message,
	})
}

func New(code int, message string, err error) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

func BadRequest(message string, err error) *AppError {
	return New(http.StatusBadRequest, message, err)
}

func InternalServerError(message string, err error) *AppError {
	return New(http.StatusInternalServerError, message, err)
}

func NotFound(message string, err error) *AppError {
	return New(http.StatusNotFound, message, err)
}
