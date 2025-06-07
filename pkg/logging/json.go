package logging

import (
	"encoding/json"
	"log/slog"

	"google.golang.org/protobuf/proto"

	"github.com/k0kubun/pp/v3"
	"google.golang.org/protobuf/encoding/protojson"
)

type SlogRawJSONValue struct {
	rawJson json.RawMessage
}

func NewSlogRawJSONValue(v any) SlogRawJSONValue {
	json, err := json.Marshal(v)
	if err != nil {
		return SlogRawJSONValue{rawJson: nil}
	}
	return SlogRawJSONValue{rawJson: json}
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

var _ slog.LogValuer = &PP{}

type PP struct {
	v any
}

func NewPP(v any) PP {
	return PP{
		v: v,
	}
}

func (p PP) LogValue() slog.Value {
	data := pp.Sprint(p.v)
	return slog.StringValue(data)
}

type Protobuf struct {
	v proto.Message
}

func NewProtobuf(v proto.Message) Protobuf {
	return Protobuf{
		v: v,
	}
}

func (p Protobuf) LogValue() slog.Value {

	return slog.StringValue(protojson.Format(p.v))
}
