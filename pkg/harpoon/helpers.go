package harpoon

import (
	"buf.build/go/protovalidate"

	harpoonv1 "github.com/walteh/ec1/gen/proto/golang/harpoon/v1"
)

func ptr[T any](v T) *T { return &v }

func NewBytestream(f func(*harpoonv1.Bytestream_builder)) *harpoonv1.Bytestream {
	builder := &harpoonv1.Bytestream_builder{}
	f(builder)
	return builder.Build()
}

func NewValidatedBytestream(f func(*harpoonv1.Bytestream_builder)) (*harpoonv1.Bytestream, error) {
	b := NewBytestream(f)
	err := protovalidate.Validate(b)
	return b, err
}

func NewExecResponse(f func(*harpoonv1.ExecResponse_builder)) *harpoonv1.ExecResponse {
	builder := &harpoonv1.ExecResponse_builder{}
	f(builder)
	return builder.Build()
}

func NewValidatedExecResponse(f func(*harpoonv1.ExecResponse_builder)) (*harpoonv1.ExecResponse, error) {
	b := NewExecResponse(f)
	err := protovalidate.Validate(b)
	return b, err
}

func NewExecResponse_Exit(f func(*harpoonv1.ExecResponse_Exit_builder)) *harpoonv1.ExecResponse_Exit {
	builder := &harpoonv1.ExecResponse_Exit_builder{}
	f(builder)
	return builder.Build()
}
