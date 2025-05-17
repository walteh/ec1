package logging

import (
	"encoding/json"
	"log/slog"
)

type SlogRawJSONValue struct {
	rawJson json.RawMessage
}

var _ slog.LogValuer = &SlogRawJSONValue{}

func (s SlogRawJSONValue) LogValue() slog.Value {
	if s.rawJson == nil {
		return slog.AnyValue(nil)
	}
	var v any
	err := json.Unmarshal(s.rawJson, &v)
	if err != nil {
		return slog.StringValue(string(s.rawJson))
	}
	return slog.AnyValue(v)
}
