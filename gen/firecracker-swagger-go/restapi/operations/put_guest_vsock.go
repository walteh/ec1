// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PutGuestVsockHandlerFunc turns a function with the right signature into a put guest vsock handler
type PutGuestVsockHandlerFunc func(PutGuestVsockParams) PutGuestVsockResponder

// Handle executing the request and returning a response
func (fn PutGuestVsockHandlerFunc) Handle(params PutGuestVsockParams) PutGuestVsockResponder {
	return fn(params)
}

// PutGuestVsockHandler interface for that can handle valid put guest vsock params
type PutGuestVsockHandler interface {
	Handle(PutGuestVsockParams) PutGuestVsockResponder
}

// NewPutGuestVsock creates a new http.Handler for the put guest vsock operation
func NewPutGuestVsock(ctx *middleware.Context, handler PutGuestVsockHandler) *PutGuestVsock {
	return &PutGuestVsock{Context: ctx, Handler: handler}
}

/*
	PutGuestVsock swagger:route PUT /vsock putGuestVsock

Creates/updates a vsock device. Pre-boot only.

The first call creates the device with the configuration specified in body. Subsequent calls will update the device configuration. May fail if update is not possible.
*/
type PutGuestVsock struct {
	Context *middleware.Context
	Handler PutGuestVsockHandler
}

func (o *PutGuestVsock) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewPutGuestVsockParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
