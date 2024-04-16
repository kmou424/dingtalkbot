package dingtalkbot

import (
	"encoding/json"
	"fmt"
	"reflect"
)

type Value struct {
	data any
}

func newValue(value any) *Value {
	return &Value{data: value}
}

//goland:noinspection GoMixedReceiverTypes
func (v *Value) UnmarshalJSON(bytes []byte) error {
	return json.Unmarshal(bytes, &v.data)
}

//goland:noinspection GoMixedReceiverTypes
func (v Value) MarshalJSON() ([]byte, error) {
	return json.Marshal(v.data)
}

func cast[T any](v *Value) (T, error) {
	tValue := *new(T)
	tType := reflect.TypeOf(tValue)
	vType := reflect.ValueOf(v.data)
	target, ok := reflect.ValueOf(v.data).Interface().(T)
	if !ok {
		return tValue, fmt.Errorf("can't convert %s to %s", vType, tType)
	}
	return target, nil
}
