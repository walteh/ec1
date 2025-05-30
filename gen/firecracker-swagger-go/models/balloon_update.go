// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	"bytes"
	"context"
	"encoding/json"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/strfmt"
	"github.com/go-openapi/swag"
	"github.com/go-openapi/validate"
)

// BalloonUpdate Balloon device descriptor.
//
// swagger:model BalloonUpdate
type BalloonUpdate struct {

	// Target balloon size in MiB.
	// Required: true
	AmountMib *int64 `json:"amount_mib"`
}

// UnmarshalJSON unmarshals this object while disallowing additional properties from JSON
func (m *BalloonUpdate) UnmarshalJSON(data []byte) error {
	var props struct {

		// Target balloon size in MiB.
		// Required: true
		AmountMib *int64 `json:"amount_mib"`
	}

	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&props); err != nil {
		return err
	}

	m.AmountMib = props.AmountMib
	return nil
}

// Validate validates this balloon update
func (m *BalloonUpdate) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateAmountMib(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *BalloonUpdate) validateAmountMib(formats strfmt.Registry) error {

	if err := validate.Required("amount_mib", "body", m.AmountMib); err != nil {
		return err
	}

	return nil
}

// ContextValidate validates this balloon update based on context it is used
func (m *BalloonUpdate) ContextValidate(ctx context.Context, formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *BalloonUpdate) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *BalloonUpdate) UnmarshalBinary(b []byte) error {
	var res BalloonUpdate
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
