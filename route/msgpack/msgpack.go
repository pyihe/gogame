package msgpackc

import (
	"github.com/pyihe/gogame/route"
	"github.com/vmihailenco/msgpack/v5"
)

const Name = "msgpack"

func init() {
	route.RegisterCodec(&msgpackCodec{})
}

type msgpackCodec struct {
}

func (m *msgpackCodec) Name() string {
	return Name
}

func (m *msgpackCodec) Encode(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (m *msgpackCodec) Decode(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}
