package jsonc

import (
	"encoding/json"

	"github.com/pyihe/gogame/route"
)

const Name = "json"

func init() {
	route.RegisterCodec(&jsCodec{})
}

type jsCodec struct{}

func (js *jsCodec) Name() string {
	return Name
}

func (js *jsCodec) Encode(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (js *jsCodec) Decode(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
