package workitem

import (
	"database/sql/driver"
	"encoding/json"
	"reflect"

	"github.com/almighty/almighty-core/convert"
	"github.com/pkg/errors"
)

type Fields map[string]interface{}

// Ensure Fields implements the Equaler interface
var _ convert.Equaler = Fields{}
var _ convert.Equaler = (*Fields)(nil)

// Equal returns true if two Fields objects are equal; otherwise false is returned.
// TODO: (kwk) think about a better comparison for Fields map.
func (f Fields) Equal(u convert.Equaler) bool {
	other, ok := u.(Fields)
	if !ok {
		return false
	}
	return reflect.DeepEqual(f, other)
}

func (f *Fields) Value() (driver.Value, error) {
	return toBytes(f)
}

func (f *Fields) Scan(src interface{}) error {
	return fromBytes(src, f)
}

type FieldDefinitions map[string]FieldDefinition

func (fd FieldDefinitions) Value() (driver.Value, error) {
	return toBytes(fd)
}

func (fd *FieldDefinitions) Scan(src interface{}) error {
	return fromBytes(src, fd)
}

func toBytes(f interface{}) (driver.Value, error) {
	if f == nil {
		// log.Trace("returning null")
		return nil, nil
	}

	res, error := json.Marshal(f)
	return res, error
}

func fromBytes(src interface{}, target interface{}) error {
	if src == nil {
		target = nil
		return nil
	}
	s, ok := src.([]byte)
	if !ok {
		return errors.New("Scan source was not string")
	}
	return json.Unmarshal(s, target)
}
