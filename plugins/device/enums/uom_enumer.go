// Code generated by "enumer -type=UOM -transform=snake -trimprefix=UOM -json -text -yaml"; DO NOT EDIT.

package enums

import (
	"encoding/json"
	"fmt"
)

const _UOMName = "imperialmetric"

var _UOMIndex = [...]uint8{0, 8, 14}

func (i UOM) String() string {
	if i < 0 || i >= UOM(len(_UOMIndex)-1) {
		return fmt.Sprintf("UOM(%d)", i)
	}
	return _UOMName[_UOMIndex[i]:_UOMIndex[i+1]]
}

var _UOMValues = []UOM{0, 1}

var _UOMNameToValueMap = map[string]UOM{
	_UOMName[0:8]:  0,
	_UOMName[8:14]: 1,
}

// UOMString retrieves an enum value from the enum constants string name.
// Throws an error if the param is not part of the enum.
func UOMString(s string) (UOM, error) {
	if val, ok := _UOMNameToValueMap[s]; ok {
		return val, nil
	}
	return 0, fmt.Errorf("%s does not belong to UOM values", s)
}

// UOMValues returns all values of the enum
func UOMValues() []UOM {
	return _UOMValues
}

// IsAUOM returns "true" if the value is listed in the enum definition. "false" otherwise
func (i UOM) IsAUOM() bool {
	for _, v := range _UOMValues {
		if i == v {
			return true
		}
	}
	return false
}

// MarshalJSON implements the json.Marshaler interface for UOM
func (i UOM) MarshalJSON() ([]byte, error) {
	return json.Marshal(i.String())
}

// UnmarshalJSON implements the json.Unmarshaler interface for UOM
func (i *UOM) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return fmt.Errorf("UOM should be a string, got %s", data)
	}

	var err error
	*i, err = UOMString(s)
	return err
}

// MarshalText implements the encoding.TextMarshaler interface for UOM
func (i UOM) MarshalText() ([]byte, error) {
	return []byte(i.String()), nil
}

// UnmarshalText implements the encoding.TextUnmarshaler interface for UOM
func (i *UOM) UnmarshalText(text []byte) error {
	var err error
	*i, err = UOMString(string(text))
	return err
}

// MarshalYAML implements a YAML Marshaler for UOM
func (i UOM) MarshalYAML() (interface{}, error) {
	return i.String(), nil
}

// UnmarshalYAML implements a YAML Unmarshaler for UOM
func (i *UOM) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}

	var err error
	*i, err = UOMString(s)
	return err
}
