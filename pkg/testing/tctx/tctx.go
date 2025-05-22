package tctx

import (
	"context"
	"testing"
)

type tCtx struct {
	t testing.TB
}

var tCtxKey = &tCtx{}

func FromContext(ctx context.Context) (testing.TB, bool) {
	t, ok := ctx.Value(tCtxKey).(*tCtx)
	if !ok {
		return nil, false
	}
	return t.t, true
}

func WithContext(ctx context.Context, t testing.TB) context.Context {
	return context.WithValue(ctx, tCtxKey, &tCtx{t: t})
}
