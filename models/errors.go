package models

import (
	"fmt"

	"github.com/almighty/almighty-core/app"
)

const (
	stBadParameterErrorMsg = "Bad value for parameter '%s': '%v'"
	stNotFoundErrorMsg     = "%s with id '%s' not found"
)

type simpleError struct {
	message string
}

func (err simpleError) Error() string {
	return err.message
}

// NewInternalError returns the custom defined error of type NewInternalError.
func NewInternalError(msg string) InternalError {
	return InternalError{simpleError{msg}}
}

// InternalError means that the operation failed for some internal, unexpected reason
type InternalError struct {
	simpleError
}

// VersionConflictError means that the version was not as expected in an update operation
type VersionConflictError struct {
	simpleError
}

// BadParameterError means that a parameter was not as required
type BadParameterError struct {
	parameter string
	value     interface{}
}

// Error implements the error interface
func (err BadParameterError) Error() string {
	return fmt.Sprintf(stBadParameterErrorMsg, err.parameter, err.value)
}

// NewBadParameterError returns the custom defined error of type NewBadParameterError.
func NewBadParameterError(param string, value interface{}) BadParameterError {
	return BadParameterError{parameter: param, value: value}
}

// NewConversionError returns the custom defined error of type NewConversionError.
func NewConversionError(msg string) ConversionError {
	return ConversionError{simpleError{msg}}
}

// ConversionError error means something went wrong converting between different representations
type ConversionError struct {
	simpleError
}

// NotFoundError means the object specified for the operation does not exist
type NotFoundError struct {
	entity string
	ID     string
}

func (err NotFoundError) Error() string {
	return fmt.Sprintf(stNotFoundErrorMsg, err.entity, err.ID)
}

// NewNotFoundError returns the custom defined error of type NewNotFoundError.
func NewNotFoundError(entity string, id string) NotFoundError {
	return NotFoundError{entity: entity, ID: id}
}

// JSONAPIErrors is defined here http://jsonapi.org/format/#error-objects
type JSONAPIErrors struct {
	app.JSONAPIErrors
}

// Error returns the JSONAPI errors as a string
func (err JSONAPIErrors) Error() string {
	if err.JSONAPIErrors.Errors != nil {
		res := "ERRORS(S): "
		for _, e := range err.JSONAPIErrors.Errors {
			res += fmt.Sprintf("%s: %s\n", e.Title, e.Detail)
		}
		return res
	}
	return "NO ERROR"
}

//func AppendJSONAPIError(jerr *JSONAPIErrors, code int, message string) JSONAPIErrors {
//	if jerr == nil {
//		jerr := &JSONAPIErrors {
//			app.JSONAPIErrors {},
//		}
//	}
//	if jerr.JSONAPIErrors.Errors == nil {
//		jerr.JSONAPIErrors.Errors = []*app.JSONAPIError
//	}
//	toAppend:=app.JSONAPIError {
//
//	}
//	append(jerr.JSONAPIErrors.Errors,)
//	return
//}
