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

// PutMmdsNoContentCode is the HTTP code returned for type PutMmdsNoContent
const PutMmdsNoContentCode int = 204

/*
PutMmdsNoContent MMDS data store created/updated.

swagger:response putMmdsNoContent
*/
type PutMmdsNoContent struct {
}

// NewPutMmdsNoContent creates PutMmdsNoContent with default headers values
func NewPutMmdsNoContent() *PutMmdsNoContent {

	return &PutMmdsNoContent{}
}

// WriteResponse to the client
func (o *PutMmdsNoContent) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.Header().Del(runtime.HeaderContentType) //Remove Content-Type on empty responses

	rw.WriteHeader(204)
}

func (o *PutMmdsNoContent) PutMmdsResponder() {}

// PutMmdsBadRequestCode is the HTTP code returned for type PutMmdsBadRequest
const PutMmdsBadRequestCode int = 400

/*
PutMmdsBadRequest MMDS data store cannot be created due to bad input.

swagger:response putMmdsBadRequest
*/
type PutMmdsBadRequest struct {

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewPutMmdsBadRequest creates PutMmdsBadRequest with default headers values
func NewPutMmdsBadRequest() *PutMmdsBadRequest {

	return &PutMmdsBadRequest{}
}

// WithPayload adds the payload to the put mmds bad request response
func (o *PutMmdsBadRequest) WithPayload(payload *models.Error) *PutMmdsBadRequest {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the put mmds bad request response
func (o *PutMmdsBadRequest) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PutMmdsBadRequest) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(400)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

func (o *PutMmdsBadRequest) PutMmdsResponder() {}

/*
PutMmdsDefault Internal server error

swagger:response putMmdsDefault
*/
type PutMmdsDefault struct {
	_statusCode int

	/*
	  In: Body
	*/
	Payload *models.Error `json:"body,omitempty"`
}

// NewPutMmdsDefault creates PutMmdsDefault with default headers values
func NewPutMmdsDefault(code int) *PutMmdsDefault {
	if code <= 0 {
		code = 500
	}

	return &PutMmdsDefault{
		_statusCode: code,
	}
}

// WithStatusCode adds the status to the put mmds default response
func (o *PutMmdsDefault) WithStatusCode(code int) *PutMmdsDefault {
	o._statusCode = code
	return o
}

// SetStatusCode sets the status to the put mmds default response
func (o *PutMmdsDefault) SetStatusCode(code int) {
	o._statusCode = code
}

// WithPayload adds the payload to the put mmds default response
func (o *PutMmdsDefault) WithPayload(payload *models.Error) *PutMmdsDefault {
	o.Payload = payload
	return o
}

// SetPayload sets the payload to the put mmds default response
func (o *PutMmdsDefault) SetPayload(payload *models.Error) {
	o.Payload = payload
}

// WriteResponse to the client
func (o *PutMmdsDefault) WriteResponse(rw http.ResponseWriter, producer runtime.Producer) {

	rw.WriteHeader(o._statusCode)
	if o.Payload != nil {
		payload := o.Payload
		if err := producer.Produce(rw, payload); err != nil {
			panic(err) // let the recovery middleware deal with this
		}
	}
}

func (o *PutMmdsDefault) PutMmdsResponder() {}

type PutMmdsNotImplementedResponder struct {
	middleware.Responder
}

func (*PutMmdsNotImplementedResponder) PutMmdsResponder() {}

func PutMmdsNotImplemented() PutMmdsResponder {
	return &PutMmdsNotImplementedResponder{
		middleware.NotImplemented(
			"operation authentication.PutMmds has not yet been implemented",
		),
	}
}

type PutMmdsResponder interface {
	middleware.Responder
	PutMmdsResponder()
}
