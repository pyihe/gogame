package pbc

import (
	"fmt"

	"github.com/golang/protobuf/proto"
	"github.com/pyihe/gogame/route"
)

const Name = "protocol"

func init() {
	route.RegisterCodec(&protoCodec{})
}

type protoCodec struct {
}

func (p *protoCodec) Name() string {
	return Name
}

func (p *protoCodec) Encode(v interface{}) ([]byte, error) {
	vv, ok := v.(proto.Message)
	if !ok {
		return nil, fmt.Errorf("faild to marshal: message is %T, want protocol.Message", v)
	}
	return proto.Marshal(vv)
}

func (p *protoCodec) Decode(data []byte, v interface{}) error {
	vv, ok := v.(proto.Message)
	if !ok {
		return fmt.Errorf("failed to unmarshal: message is %T, want protocol.Message", v)
	}
	return proto.Unmarshal(data, vv)
}
