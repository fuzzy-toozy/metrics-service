package common

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func Test_Retrier(t *testing.T) {
	ctx, cancel := context.WithCancel(context.TODO())
	dummyErr := errors.New("Dummy")
	const retriesCnt = 5

	cancel()
	retrier := NewCommonRetryExecutor(ctx, time.Second*10, retriesCnt, []error{dummyErr})
	err := retrier.RetryOnError(func() error {
		return dummyErr
	})

	require.ErrorIs(t, err, context.Canceled)

	retrier = NewCommonRetryExecutor(context.TODO(), time.Second*10, retriesCnt, nil)
	err = retrier.RetryOnError(func() error {
		return dummyErr
	})

	require.ErrorIs(t, err, dummyErr)

	retrier = NewCommonRetryExecutor(context.TODO(), time.Millisecond*10, retriesCnt, []error{dummyErr})

	ctr := new(int)

	err = retrier.RetryOnError(func() error {
		*ctr += 1
		return dummyErr
	})

	require.Equal(t, *ctr, retriesCnt+1)
	require.ErrorIs(t, err, dummyErr)

	err = retrier.RetryOnError(func() error {
		return nil
	})

	require.NoError(t, err)
}
