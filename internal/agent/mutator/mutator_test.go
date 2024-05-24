package mutator

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRun(t *testing.T) {
	r := require.New(t)

	ctx := context.Background()

	dataMutator := NewDataMutator(mockContextAppend)

	const key = "Hello"
	const value = "World"

	mockMutationFunc1 := func(ctx context.Context, data *bytes.Buffer) (context.Context, error) {
		ctx = dataMutator.AppendCtx(ctx, key, value)
		return ctx, nil
	}
	mockMutationFunc2 := func(ctx context.Context, data *bytes.Buffer) (context.Context, error) {
		data.WriteString(value)
		return ctx, nil
	}

	dataMutator.AddFunc(mockMutationFunc1)
	dataMutator.AddFunc(mockMutationFunc2)

	inputData := []byte(key)

	resultCtx, err := dataMutator.Run(ctx, inputData)
	r.NoError(err)

	ctxDataAny := resultCtx.Value(ContextKey(string(key)))
	r.NotNil(ctxDataAny)

	ctxData, ok := ctxDataAny.(string)
	r.True(ok)
	r.Equal(ctxData, value)

	r.Equal(key+value, dataMutator.GetData().String())
}

func mockContextAppend(ctx context.Context, key ContextKey, val string) context.Context {
	return context.WithValue(ctx, key, val)
}
