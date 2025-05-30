// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// CreateSyncActionHandlerFunc turns a function with the right signature into a create sync action handler
type CreateSyncActionHandlerFunc func(CreateSyncActionParams) CreateSyncActionResponder

// Handle executing the request and returning a response
func (fn CreateSyncActionHandlerFunc) Handle(params CreateSyncActionParams) CreateSyncActionResponder {
	return fn(params)
}

// CreateSyncActionHandler interface for that can handle valid create sync action params
type CreateSyncActionHandler interface {
	Handle(CreateSyncActionParams) CreateSyncActionResponder
}

// NewCreateSyncAction creates a new http.Handler for the create sync action operation
func NewCreateSyncAction(ctx *middleware.Context, handler CreateSyncActionHandler) *CreateSyncAction {
	return &CreateSyncAction{Context: ctx, Handler: handler}
}

/*
	CreateSyncAction swagger:route PUT /actions createSyncAction

Creates a synchronous action.
*/
type CreateSyncAction struct {
	Context *middleware.Context
	Handler CreateSyncActionHandler
}

func (o *CreateSyncAction) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewCreateSyncActionParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
