package valuelog

import "log/slog"

type FormattedError struct {
	err error
}

func NewFormattedError(err error) slog.Value {
	return slog.AnyValue(&FormattedError{err})
}

func (e *FormattedError) Ref() *error {
	return &e.err
}
