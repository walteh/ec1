package proc

import "context"

type Interface interface {
	ID() string
	Start(ctx context.Context) error
	Stop(ctx context.Context) error
}

func CleanupAfter(ctx context.Context, proc Interface) error {

	return nil
}

// func WaitFor
