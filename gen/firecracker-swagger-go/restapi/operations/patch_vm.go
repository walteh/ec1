// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PatchVMHandlerFunc turns a function with the right signature into a patch Vm handler
type PatchVMHandlerFunc func(PatchVMParams) PatchVMResponder

// Handle executing the request and returning a response
func (fn PatchVMHandlerFunc) Handle(params PatchVMParams) PatchVMResponder {
	return fn(params)
}

// PatchVMHandler interface for that can handle valid patch Vm params
type PatchVMHandler interface {
	Handle(PatchVMParams) PatchVMResponder
}

// NewPatchVM creates a new http.Handler for the patch Vm operation
func NewPatchVM(ctx *middleware.Context, handler PatchVMHandler) *PatchVM {
	return &PatchVM{Context: ctx, Handler: handler}
}

/*
	PatchVM swagger:route PATCH /vm patchVm

Updates the microVM state.

Sets the desired state (Paused or Resumed) for the microVM.
*/
type PatchVM struct {
	Context *middleware.Context
	Handler PatchVMHandler
}

func (o *PatchVM) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewPatchVMParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
