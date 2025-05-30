// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PatchBalloonHandlerFunc turns a function with the right signature into a patch balloon handler
type PatchBalloonHandlerFunc func(PatchBalloonParams) PatchBalloonResponder

// Handle executing the request and returning a response
func (fn PatchBalloonHandlerFunc) Handle(params PatchBalloonParams) PatchBalloonResponder {
	return fn(params)
}

// PatchBalloonHandler interface for that can handle valid patch balloon params
type PatchBalloonHandler interface {
	Handle(PatchBalloonParams) PatchBalloonResponder
}

// NewPatchBalloon creates a new http.Handler for the patch balloon operation
func NewPatchBalloon(ctx *middleware.Context, handler PatchBalloonHandler) *PatchBalloon {
	return &PatchBalloon{Context: ctx, Handler: handler}
}

/*
	PatchBalloon swagger:route PATCH /balloon patchBalloon

Updates a balloon device.

Updates an existing balloon device, before or after machine startup. Will fail if update is not possible.
*/
type PatchBalloon struct {
	Context *middleware.Context
	Handler PatchBalloonHandler
}

func (o *PatchBalloon) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewPatchBalloonParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
