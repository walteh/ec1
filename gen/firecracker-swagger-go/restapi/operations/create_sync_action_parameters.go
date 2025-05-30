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
	"github.com/go-openapi/validate"

	"github.com/walteh/ec1/gen/firecracker-swagger-go/models"
)

// NewCreateSyncActionParams creates a new CreateSyncActionParams object
//
// There are no default values defined in the spec.
func NewCreateSyncActionParams() CreateSyncActionParams {

	return CreateSyncActionParams{}
}

// CreateSyncActionParams contains all the bound params for the create sync action operation
// typically these are obtained from a http.Request
//
// swagger:parameters createSyncAction
type CreateSyncActionParams struct {

	// HTTP Request Object
	HTTPRequest *http.Request `json:"-"`

	/*
	  Required: true
	  In: body
	*/
	Info *models.InstanceActionInfo
}

// BindRequest both binds and validates a request, it assumes that complex things implement a Validatable(strfmt.Registry) error interface
// for simple values it will use straight method calls.
//
// To ensure default values, the struct must have been initialized with NewCreateSyncActionParams() beforehand.
func (o *CreateSyncActionParams) BindRequest(r *http.Request, route *middleware.MatchedRoute) error {
	var res []error

	o.HTTPRequest = r

	if runtime.HasBody(r) {
		defer r.Body.Close()
		var body models.InstanceActionInfo
		if err := route.Consumer.Consume(r.Body, &body); err != nil {
			if err == io.EOF {
				res = append(res, errors.Required("info", "body", ""))
			} else {
				res = append(res, errors.NewParseError("info", "body", "", err))
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
				o.Info = &body
			}
		}
	} else {
		res = append(res, errors.Required("info", "body", ""))
	}
	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}
