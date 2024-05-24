// Package errtypes Error types to return from services methods
package errtypes

import (
	"context"
	"errors"
	"net/http"

	"google.golang.org/grpc/codes"
)

type GenericErrorWrapper struct {
	err error
}

func (e GenericErrorWrapper) Error() string {
	return e.err.Error()
}

func (e GenericErrorWrapper) Unwrap() error {
	return e.err
}

type ServerError struct {
	GenericErrorWrapper
}

type BadDataError struct {
	GenericErrorWrapper
}

type NotFoundError struct {
	GenericErrorWrapper
}

func MakeGenericErrorWrapper(err error) GenericErrorWrapper {
	return GenericErrorWrapper{err: err}
}

func MakeServerError(err error) ServerError {
	return ServerError{GenericErrorWrapper: GenericErrorWrapper{err: err}}
}

func MakeBadDataError(err error) BadDataError {
	return BadDataError{GenericErrorWrapper: GenericErrorWrapper{err: err}}
}

func MakeNotFoundError(err error) NotFoundError {
	return NotFoundError{GenericErrorWrapper: GenericErrorWrapper{err: err}}
}

func ErrorToStatusHTTP(err error) int {
	if err == nil {
		return http.StatusOK
	}

	status := http.StatusInternalServerError
	var notFoundError NotFoundError
	var requestError BadDataError

	if errors.As(err, &notFoundError) {
		status = http.StatusNotFound
	} else if errors.As(err, &requestError) {
		status = http.StatusBadRequest
	} else if errors.Is(err, context.DeadlineExceeded) {
		status = http.StatusRequestTimeout
	}

	return status
}

func ErrorToStatusGRPC(err error) int {
	if err == nil {
		return int(codes.OK)
	}

	status := codes.Internal
	var notFoundError NotFoundError
	var requestError BadDataError

	if errors.As(err, &notFoundError) {
		status = codes.NotFound
	} else if errors.As(err, &requestError) {
		status = codes.InvalidArgument
	} else if errors.Is(err, context.DeadlineExceeded) {
		status = codes.DeadlineExceeded
	}

	return int(status)
}
