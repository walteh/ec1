// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"net/http"

	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"

	"github.com/walteh/ec1/gen/firecracker-swagger-go/models"
)

// PatchVMNoContentCode is the HTTP code returned for type PatchVMNoContent
const PatchVMNoContentCode int = 204

/*
PatchVMNoContent Vm state updated

swagger:response patchVmNoContent
*/
type PatchVMNoContent struct {
}

// NewPatchVMNoContent creates PatchVMNoContent with default headers values
func NewPatchVMNoContent() *PatchVMNoContent {

	return &PatchVMNoContent{}
}

// WriteResponse to the client
func (o *PatchVMNoContent) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(204)
}

func (o *PatchVMNoContent) PatchVMResponder() {}

// PatchVMBadRequestCode is the HTTP code returned for type PatchVMBadRequest
const PatchVMBadRequestCode int = 400

/*
PatchVMBadRequest Vm state cannot be updated due to bad input

swagger:response patchVmBadRequest
*/
type PatchVMBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewPatchVMBadRequest creates PatchVMBadRequest with default headers values
func NewPatchVMBadRequest() *PatchVMBadRequest {

	return &PatchVMBadRequest{}
}

// WithPayload adds the payload to the patch Vm bad request response
func (o *PatchVMBadRequest) WithPayload(payload *models.Error) *PatchVMBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the patch Vm bad request response
func (o *PatchVMBadRequest) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PatchVMBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

func (o *PatchVMBadRequest) PatchVMResponder() {}

/*
PatchVMDefault Internal server error

swagger:response patchVmDefault
*/
type PatchVMDefault struct {
	_statusCode int

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewPatchVMDefault creates PatchVMDefault with default headers values
func NewPatchVMDefault(code int) *PatchVMDefault {
	if code <= 0 {
		code = 500
	}

	return &PatchVMDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the patch Vm default response
func (o *PatchVMDefault) WithStatusCode(code int) *PatchVMDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the patch Vm default response
func (o *PatchVMDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the patch Vm default response
func (o *PatchVMDefault) WithPayload(payload *models.Error) *PatchVMDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the patch Vm default response
func (o *PatchVMDefault) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PatchVMDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

func (o *PatchVMDefault) PatchVMResponder() {}

type PatchVMNotImplementedResponder struct {
	middleware.Responder
}

func (*PatchVMNotImplementedResponder) PatchVMResponder() {}

func PatchVMNotImplemented() PatchVMResponder {
	return &PatchVMNotImplementedResponder{
		middleware.NotImplemented(
			"operation authentication.PatchVM has not yet been implemented",
		),
	}
}

type PatchVMResponder interface {
	middleware.Responder
	PatchVMResponder()
}
