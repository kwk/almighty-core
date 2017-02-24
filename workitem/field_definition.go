package workitem

import (
	"encoding/json"
	"fmt"
	"reflect"

	"strings"

	"github.com/almighty/almighty-core/convert"
	"github.com/pkg/errors"
)

// constants for describing possible field types
const (
	KindString            Kind = "string"
	KindInteger           Kind = "integer"
	KindFloat             Kind = "float"
	KindInstant           Kind = "instant"
	KindDuration          Kind = "duration"
	KindURL               Kind = "url"
	KindIteration         Kind = "iteration"
	KindWorkitemReference Kind = "workitem"
	KindUser              Kind = "user"
	KindEnum              Kind = "enum"
	KindList              Kind = "list"
	KindMarkup            Kind = "markup"
	KindArea              Kind = "area"
)

// Kind is the kind of field type
type Kind string

// FieldType describes the possible values of a FieldDefinition
func (k Kind) isSimpleType() bool {
	return k != KindEnum && k != KindList
}

// FieldType describes the possible values of a FieldDefinition
type FieldType interface {
	GetKind() Kind
	// ConvertToModel converts a field value for use in the persistence layer
	ConvertToModel(value interface{}) (interface{}, error)
	// ConvertFromModel converts a field value for use in the REST API layer
	ConvertFromModel(value interface{}) (interface{}, error)
	// Implement the Equaler interface
	Equal(u convert.Equaler) bool
}

// FieldDefinition describes type & other restrictions of a field
type FieldDefinition struct {
	Required bool
	Type     FieldType
}

// Ensure FieldDefinition implements the Equaler interface
var _ convert.Equaler = FieldDefinition{}
var _ convert.Equaler = (*FieldDefinition)(nil)

// Equal returns true if two FieldDefinition objects are equal; otherwise false is returned.
func (fd FieldDefinition) Equal(u convert.Equaler) bool {
	other, ok := u.(FieldDefinition)
	if !ok {
		return false
	}
	if fd.Required != other.Required {
		return false
	}
	return fd.Type.Equal(other.Type)
}

// ConvertToModel converts a field value for use in the persistence layer
func (fd FieldDefinition) ConvertToModel(name string, value interface{}) (interface{}, error) {
	if fd.Required && (value == nil || (fd.Type.GetKind() == KindString && strings.TrimSpace(value.(string)) == "")) {
		return nil, fmt.Errorf("Value %s is required", name)
	}
	return fd.Type.ConvertToModel(value)
}

// ConvertFromModel converts a field value for use in the REST API layer
func (fd FieldDefinition) ConvertFromModel(name string, value interface{}) (interface{}, error) {
	if fd.Required && value == nil {
		return nil, fmt.Errorf("Value %s is required", name)
	}
	return fd.Type.ConvertFromModel(value)
}

type rawFieldDef struct {
	Required bool
	Type     *json.RawMessage
}

// Ensure rawFieldDef implements the Equaler interface
var _ convert.Equaler = rawFieldDef{}
var _ convert.Equaler = (*rawFieldDef)(nil)

// Equal returns true if two rawFieldDef objects are equal; otherwise false is returned.
func (rfd rawFieldDef) Equal(u convert.Equaler) bool {
	other, ok := u.(rawFieldDef)
	if !ok {
		return false
	}
	if rfd.Required != other.Required {
		return false
	}
	if rfd.Type == nil && other.Type == nil {
		return true
	}
	if rfd.Type != nil && other.Type != nil {
		return reflect.DeepEqual(rfd.Type, other.Type)
	}
	return false
}

// UnmarshalJSON implements encoding/json.Unmarshaler
func (fd *FieldDefinition) UnmarshalJSON(bytes []byte) error {
	temp := rawFieldDef{}

	err := json.Unmarshal(bytes, &temp)
	if err != nil {
		return errors.WithStack(err)
	}
	rawType := map[string]interface{}{}
	json.Unmarshal(*temp.Type, &rawType)

	kind, err := convertAnyToKind(rawType["Kind"])

	if err != nil {
		return errors.WithStack(err)
	}
	switch *kind {
	case KindList:
		theType := ListType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return errors.WithStack(err)
		}
		*fd = FieldDefinition{Type: theType, Required: temp.Required}
	case KindEnum:
		theType := EnumType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return errors.WithStack(err)
		}
		*fd = FieldDefinition{Type: theType, Required: temp.Required}
	default:
		theType := SimpleType{}
		err = json.Unmarshal(*temp.Type, &theType)
		if err != nil {
			return errors.WithStack(err)
		}
		*fd = FieldDefinition{Type: theType, Required: temp.Required}
	}
	return nil
}
