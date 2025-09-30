package myerrors

import (
	"fmt"
	"net/http"
)

type AppError struct {
	Message    string
	StatusCode int
}

func (r *AppError) Error() string {
	return fmt.Sprintf("error %s. status code :%d", r.Message, r.StatusCode)
}

func NewBadRequestError(msg string) *AppError {
	return &AppError{Message: msg, StatusCode: http.StatusBadRequest}
}

func NewNotFoundError(msg string) *AppError {
	return &AppError{Message: msg, StatusCode: http.StatusNotFound}
}

func NewServerError(msg string) *AppError {
	return &AppError{Message: msg, StatusCode: http.StatusInternalServerError}
}
