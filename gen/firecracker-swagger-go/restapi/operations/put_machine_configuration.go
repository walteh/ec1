// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// PutMachineConfigurationHandlerFunc turns a function with the right signature into a put machine configuration handler
type PutMachineConfigurationHandlerFunc func(PutMachineConfigurationParams) PutMachineConfigurationResponder

// Handle executing the request and returning a response
func (fn PutMachineConfigurationHandlerFunc) Handle(params PutMachineConfigurationParams) PutMachineConfigurationResponder {
	return fn(params)
}

// PutMachineConfigurationHandler interface for that can handle valid put machine configuration params
type PutMachineConfigurationHandler interface {
	Handle(PutMachineConfigurationParams) PutMachineConfigurationResponder
}

// NewPutMachineConfiguration creates a new http.Handler for the put machine configuration operation
func NewPutMachineConfiguration(ctx *middleware.Context, handler PutMachineConfigurationHandler) *PutMachineConfiguration {
	return &PutMachineConfiguration{Context: ctx, Handler: handler}
}

/*
	PutMachineConfiguration swagger:route PUT /machine-config putMachineConfiguration

Updates the Machine Configuration of the VM. Pre-boot only.

Updates the Virtual Machine Configuration with the specified input. Firecracker starts with default values for vCPU count (=1) and memory size (=128 MiB). The vCPU count is restricted to the [1, 32] range. With SMT enabled, the vCPU count is required to be either 1 or an even number in the range. otherwise there are no restrictions regarding the vCPU count. If 2M hugetlbfs pages are specified, then `mem_size_mib` must be a multiple of 2. If any of the parameters has an incorrect value, the whole update fails. All parameters that are optional and are not specified are set to their default values (smt = false, track_dirty_pages = false, cpu_template = None, huge_pages = None).
*/
type PutMachineConfiguration struct {
	Context *middleware.Context
	Handler PutMachineConfigurationHandler
}

func (o *PutMachineConfiguration) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewPutMachineConfigurationParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
