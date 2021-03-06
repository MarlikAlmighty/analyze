// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/errors"
	"github.com/go-openapi/swag"
)

// Obj obj
// swagger:model Obj
type Obj struct {

	// array
	Array Array `json:"Array,omitempty"`

	// timestamp
	Timestamp string `json:"Timestamp,omitempty"`
}

// Validate validates this obj
func (m *Obj) Validate(formats strfmt.Registry) error {
	var res []error

	if err := m.validateArray(formats); err != nil {
		res = append(res, err)
	}

	if len(res) > 0 {
		return errors.CompositeValidationError(res...)
	}
	return nil
}

func (m *Obj) validateArray(formats strfmt.Registry) error {

	if swag.IsZero(m.Array) { // not required
		return nil
	}

	if err := m.Array.Validate(formats); err != nil {
		if ve, ok := err.(*errors.Validation); ok {
			return ve.ValidateName("Array")
		}
		return err
	}

	return nil
}

// MarshalBinary interface implementation
func (m *Obj) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Obj) UnmarshalBinary(b []byte) error {
	var res Obj
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
