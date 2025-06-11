package valuelog

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/k0kubun/pp/v3"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type PrettyValue interface {
	PrettyString() string
}

type PrettyRawJSONValue interface {
	RawJSON() []byte
}

type PrettyAnyValue interface {
	Any() any
}

type PrettyStructValue struct {
	v any
}

type PrettyValueHandler interface {
	HandleJSON([]byte) string
	HandleStruct(any) string
}

func (h *PrettyStructValue) PrettyString() string {
	p := pp.New()
	p.SetExportedOnly(true)
	return p.Sprint(h.v)
}

func (h *PrettyStructValue) Any() any {
	return h.v
}

type PrettyJSONValue struct {
	v any
}

func (h *PrettyStructValue) RawJSON() []byte {
	json, err := json.MarshalIndent(h.v, "", "\t")
	if err != nil {
		return []byte(fmt.Sprintf("!PANIC: %v", err))
	}
	return json
}
func (h *PrettyJSONValue) PrettyString() string {
	json, err := json.MarshalIndent(h.v, "", "\t")
	if err != nil {
		return fmt.Sprintf("!PANIC: %v", err)
	}
	return string(json)
}

type PrettyProtobufValue struct {
	v proto.Message
}

func (h *PrettyProtobufValue) RawJSON() []byte {
	opts := protojson.MarshalOptions{
		Indent:            "\t",
		Multiline:         true,
		UseProtoNames:     true,
		UseEnumNumbers:    false,
		EmitUnpopulated:   false,
		EmitDefaultValues: false,
		AllowPartial:      true,
	}
	json, err := opts.Marshal(h.v)
	if err != nil {
		return []byte(fmt.Sprintf("!PANIC: %v", err))
	}
	return json
}

func (h *PrettyProtobufValue) PrettyString() string {
	return string(h.RawJSON())
}

func NewPrettyValue(v any) slog.Value {
	switch v := v.(type) {
	case proto.Message:
		return slog.AnyValue(&PrettyProtobufValue{v})
	case json.Marshaler:
		return slog.AnyValue(&PrettyJSONValue{v})
	default:
		return slog.AnyValue(&PrettyStructValue{v})
	}
}

// func HandlePrettyValue(value slog.Value, handler PrettyValueHandler) string {
// 	switch v := value.Any().(type) {
// 	case PrettyProtobufValue:
// 		return handler.HandleJSON([]byte(v.format()))
// 	case PrettyJSONValue:
// 		return handler.HandleJSON([]byte(v.append()))
// 	default:
// 		return handler.HandleStruct(v)
// 	}
// }
