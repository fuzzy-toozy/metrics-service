package common

import (
	"errors"
	"net"
	"time"
)

type RetryExecutor interface {
	RetryOnError(callback func() error) error
}

type CommonRetryExecutor struct {
	retryDelta   time.Duration
	retriesCount uint
	errs         []error
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

func (r *CommonRetryExecutor) RetryOnError(callback func() error) error {
	waitTime := r.retryDelta
	var err error
	for retry := uint(0); retry <= r.retriesCount; retry++ {
		err = callback()
		if !errorIsOneOf(err, r.errs) && !isNetworkError(err) {
			break
		}
		time.Sleep(waitTime)

		waitTime += r.retryDelta
	}

	return err
}

func NewCommonRetryExecutor(retryDelta time.Duration, retriesCount uint, allowedErrors []error) *CommonRetryExecutor {
	return &CommonRetryExecutor{retryDelta: retryDelta, retriesCount: retriesCount, errs: allowedErrors}
}
