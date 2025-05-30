// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the generate command

import (
	"net/http"

	"github.com/go-openapi/runtime/middleware"
)

// DescribeBalloonStatsHandlerFunc turns a function with the right signature into a describe balloon stats handler
type DescribeBalloonStatsHandlerFunc func(DescribeBalloonStatsParams) DescribeBalloonStatsResponder

// Handle executing the request and returning a response
func (fn DescribeBalloonStatsHandlerFunc) Handle(params DescribeBalloonStatsParams) DescribeBalloonStatsResponder {
	return fn(params)
}

// DescribeBalloonStatsHandler interface for that can handle valid describe balloon stats params
type DescribeBalloonStatsHandler interface {
	Handle(DescribeBalloonStatsParams) DescribeBalloonStatsResponder
}

// NewDescribeBalloonStats creates a new http.Handler for the describe balloon stats operation
func NewDescribeBalloonStats(ctx *middleware.Context, handler DescribeBalloonStatsHandler) *DescribeBalloonStats {
	return &DescribeBalloonStats{Context: ctx, Handler: handler}
}

/*
	DescribeBalloonStats swagger:route GET /balloon/statistics describeBalloonStats

Returns the latest balloon device statistics, only if enabled pre-boot.
*/
type DescribeBalloonStats struct {
	Context *middleware.Context
	Handler DescribeBalloonStatsHandler
}

func (o *DescribeBalloonStats) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	route, rCtx, _ := o.Context.RouteInfo(r)
	if rCtx != nil {
		*r = *rCtx
	}
	var Params = NewDescribeBalloonStatsParams()
	if err := o.Context.BindValidRequest(r, route, &Params); err != nil { // bind params
		o.Context.Respond(rw, r, route.Produces, route, err)
		return
	}

	res := o.Handler.Handle(Params) // actually handle the request
	o.Context.Respond(rw, r, route.Produces, route, res)

}
