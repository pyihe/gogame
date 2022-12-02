package gobc

import (
	"bytes"
	"encoding/gob"

	"github.com/pyihe/gogame/route"
)

const Name = "gob"

func init() {
	route.RegisterCodec(&gobCodec{})
}

type gobCodec struct {
}

func (g *gobCodec) Name() string {
	return Name
}

func (g *gobCodec) Encode(v interface{}) ([]byte, error) {
	buff := bytes.NewBuffer([]byte{})
	err := gob.NewEncoder(buff).Encode(v)
	if err != nil {
		return nil, err
	}
	return buff.Bytes(), nil
}

func (g *gobCodec) Decode(data []byte, v interface{}) error {
	return gob.NewDecoder(bytes.NewReader(data)).Decode(v)
}
