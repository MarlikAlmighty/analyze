// Code generated by go-swagger; DO NOT EDIT.

package models

// This file was generated by the swagger tool.
// Editing this file might prove futile when you re-run the swagger generate command

import (
	strfmt "github.com/go-openapi/strfmt"

	"github.com/go-openapi/swag"
)

// Post post
// swagger:model Post
type Post struct {

	// body
	Body string `json:"Body,omitempty"`

	// hash
	Hash string `json:"Hash,omitempty"`

	// image
	Image string `json:"Image,omitempty"`

	// link
	Link string `json:"Link,omitempty"`

	// title
	Title string `json:"Title,omitempty"`
}

// Validate validates this post
func (m *Post) Validate(formats strfmt.Registry) error {
	return nil
}

// MarshalBinary interface implementation
func (m *Post) MarshalBinary() ([]byte, error) {
	if m == nil {
		return nil, nil
	}
	return swag.WriteJSON(m)
}

// UnmarshalBinary interface implementation
func (m *Post) UnmarshalBinary(b []byte) error {
	var res Post
	if err := swag.ReadJSON(b, &res); err != nil {
		return err
	}
	*m = res
	return nil
}
