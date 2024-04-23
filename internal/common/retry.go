// Package common Compression/Decompresion algorithms for handlers
package common

import (
	"context"
	"errors"
	"net"
	"time"
)

// RetryExecutor retry executor base interface
type RetryExecutor interface {
	RetryOnError(callback func() error) error
}

// CommonRetryExecutor default retry executor implementation.
type CommonRetryExecutor struct {
	// stopCtx context to stop retry executor by calling cancel.
	stopCtx context.Context
	// errs in case any of these errors returned by callback - retry callback execution
	errs []error
	// retryDelta this amount of time is added to wait time between retries each retry attempt.
	retryDelta time.Duration
	// retriesCount number of retry attempts
	retriesCount uint
}

func errorIsOneOf(target error, expected []error) bool {
	for _, err := range expected {
		if errors.Is(err, target) {
			return true
		}
	}

	return false
}

func isNetworkError(target error) bool {
	var netErr net.Error
	return errors.As(target, &netErr)
}

// RetryOnError executes callback and retries if it returns network error or
// any error passed to NewCommonRetryExecutor retriesCount times.
// For each retry interval between retries increases by retryDelta.
// Simply returns if returned error doesn't match or no error occured.
func (r *CommonRetryExecutor) RetryOnError(callback func() error) error {
	waitTime := r.retryDelta
	var err error
	for retry := uint(0); retry <= r.retriesCount; retry++ {
		err = callback()
		if !errorIsOneOf(err, r.errs) && !isNetworkError(err) {
			break
		}

		select {
		case <-time.After(waitTime):
		case <-r.stopCtx.Done():
			return r.stopCtx.Err()
		}

		waitTime += r.retryDelta
	}

	return err
}

func NewCommonRetryExecutor(
	stopCtx context.Context,
	retryDelta time.Duration,
	retriesCount uint,
	allowedErrors []error) *CommonRetryExecutor {
	return &CommonRetryExecutor{
		stopCtx:      stopCtx,
		retryDelta:   retryDelta,
		retriesCount: retriesCount,
		errs:         allowedErrors}
}
