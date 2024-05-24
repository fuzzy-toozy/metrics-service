package mutator

import (
	"bytes"
	"context"
)

type MutationFunc func(ctx context.Context, data *bytes.Buffer) (context.Context, error)

type DataMutator struct {
	mutationChain []MutationFunc
	ctxAppendFunc ContextAppendFunc
	buff          *bytes.Buffer
}

type ContextKey string

func NewDataMutator(ctxAppend ContextAppendFunc) *DataMutator {
	return &DataMutator{ctxAppendFunc: ctxAppend}
}

func (m *DataMutator) AddFunc(f MutationFunc) {
	m.mutationChain = append(m.mutationChain, f)
}

func (m *DataMutator) Reset() {
	m.buff.Reset()
}

func (m *DataMutator) Run(ctx context.Context, data []byte) (context.Context, error) {
	var err error

	m.buff = bytes.NewBuffer(data)

	for _, f := range m.mutationChain {
		ctx, err = f(ctx, m.buff)
		if err != nil {
			return ctx, err
		}
	}

	return ctx, nil
}

func (m *DataMutator) GetData() *bytes.Buffer {
	return m.buff
}

func (m *DataMutator) AppendCtx(ctx context.Context, key ContextKey, val string) context.Context {
	return m.ctxAppendFunc(ctx, key, val)
}

type ContextAppendFunc = func(ctx context.Context, key ContextKey, val string) context.Context
type MutationOption = func(m *DataMutator) MutationFunc
