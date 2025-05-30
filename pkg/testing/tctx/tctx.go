package tctx

import (
	"context"
	"errors"
	"testing"
	"time"
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

func WithinTimeout[T any](t *testing.T, timeout time.Duration, f func() (T, error)) (T, bool, error) {
	t.Helper()
	chr := make(chan T, 1)
	che := make(chan error, 1)
	defer close(chr)
	defer close(che)
	go func() {
		res, err := f()
		if err != nil {
			che <- err
		} else {
			chr <- res
		}
	}()
	select {
	case res := <-chr:
		return res, false, nil
	case err := <-che:
		return *new(T), false, err
	case <-time.After(timeout):
		return *new(T), true, errors.New("timeout")
	}
}
