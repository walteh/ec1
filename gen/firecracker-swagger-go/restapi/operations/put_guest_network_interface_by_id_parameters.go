// Code generated by go-swagger; DO NOT EDIT.

package operations

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"io"
	"net/http"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/runtime"
	"github.com/go-openapi/runtime/middleware"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/validate"

	"github.com/walteh/ec1/gen/firecracker-swagger-go/models"
)

// NewPutGuestNetworkInterfaceByIDParams creates a new PutGuestNetworkInterfaceByIDParams object
//
// There are no default values defined in the spec.
func NewPutGuestNetworkInterfaceByIDParams() PutGuestNetworkInterfaceByIDParams {

	return PutGuestNetworkInterfaceByIDParams{}
}

// PutGuestNetworkInterfaceByIDParams contains all the bound params for the put guest network interface by ID operation
// typically these are obtained from a http.Request
//
// swagger:parameters putGuestNetworkInterfaceByID
type PutGuestNetworkInterfaceByIDParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*Guest network interface properties
	  Required: true
	  In: body
	*/
	Body *models.NetworkInterface
	/*The id of the guest network interface
	  Required: true
	  In: path
	*/
	IfaceID string
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewPutGuestNetworkInterfaceByIDParams() beforehand.
func (o *PutGuestNetworkInterfaceByIDParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	if runtime.HasBody(r) {
		defer r.Body.Close()
		var body models.NetworkInterface
		if err := route.Consumer.Consume(r.Body, &body); err != nil {
			if err == io.EOF {
				res = append(res, errors.Required("body", "body", ""))
			} else {
				res = append(res, errors.NewParseError("body", "body", "", err))
			}
		} else {
			// validate body object
			if err := body.Validate(route.Formats); err != nil {
				res = append(res, err)
			}

			ctx := validate.WithOperationRequest(r.Context())
			if err := body.ContextValidate(ctx, route.Formats); err != nil {
				res = append(res, err)
			}

			if len(res) == 0 {
				o.Body = &body
			}
		}
	} else {
		res = append(res, errors.Required("body", "body", ""))
	}

	rIfaceID, rhkIfaceID, _ := route.Params.GetOK("iface_id")
	if err := o.bindIfaceID(rIfaceID, rhkIfaceID, route.Formats); err != nil {
		res = append(res, err)
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

// bindIfaceID binds and validates parameter IfaceID from path.
func (o *PutGuestNetworkInterfaceByIDParams) bindIfaceID(rawData []string, hasKey bool, formats strfmt.Registry) error {
	var raw string
	if len(rawData) > 0 {
		raw = rawData[len(rawData)-1]
	}

	// Required: true
	// Parameter is provided by construction from the route
	o.IfaceID = raw

	return nil
}
