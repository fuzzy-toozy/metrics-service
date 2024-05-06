// Package errtypes Error types to return from services methods
package errtypes

import (
	"context"
	"errors"
	"net/http"
)

type genericErrorWrapper struct {
	err error
}

func (e genericErrorWrapper) Error() string {
	return e.err.Error()
}

func (e genericErrorWrapper) Unwrap() error {
	return e.err
}

type ServerError struct {
	genericErrorWrapper
}

type BadDataError struct {
	genericErrorWrapper
}

type NotFoundError struct {
	genericErrorWrapper
}

func MakeServerError(err error) ServerError {
	return ServerError{genericErrorWrapper: genericErrorWrapper{err: err}}
}

func MakeBadDataError(err error) BadDataError {
	return BadDataError{genericErrorWrapper: genericErrorWrapper{err: err}}
}

func MakeNotFoundError(err error) NotFoundError {
	return NotFoundError{genericErrorWrapper: genericErrorWrapper{err: err}}
}

func ErrorToStatus(err error) int {
	status := http.StatusOK

	var serverError ServerError
	var notFoundError NotFoundError
	var requestError BadDataError

	if errors.As(err, &serverError) {
		status = http.StatusInternalServerError
	} else if errors.As(err, &notFoundError) {
		status = http.StatusNotFound
	} else if errors.As(err, &requestError) {
		status = http.StatusBadRequest
	} else if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusRequestTimeout
	}

	return status
}
