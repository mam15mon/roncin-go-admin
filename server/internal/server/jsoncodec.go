package server

import (
	"encoding/json"

	"github.com/go-kratos/kratos/v3/encoding"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// 注册 protojson 版 JSON 编解码器，覆盖 Kratos 内置的 encoding/json 实现。
// 内置实现按 Go 结构体 json tag（snake_case）解码请求体，而 OpenAPI 契约与
// 响应编码均使用 ProtoJSON camelCase 字段，导致前端写操作请求体字段全部
// 解码为空值。protojson 解码同时接受 camelCase 与原始 snake_case 字段名，
// 对既有调用方保持兼容。
func init() {
	encoding.RegisterCodec(protoJSONCodec{})
}

type protoJSONCodec struct{}

func (protoJSONCodec) Name() string { return "json" }

func (protoJSONCodec) Marshal(v any) ([]byte, error) {
	switch m := v.(type) {
	case json.Marshaler:
		return m.MarshalJSON()
	case proto.Message:
		return marshalProtoJSON(m)
	default:
		return json.Marshal(v)
	}
}

func marshalProtoJSON(message proto.Message) ([]byte, error) {
	return (protojson.MarshalOptions{UseEnumNumbers: true}).Marshal(message)
}

func (protoJSONCodec) Unmarshal(data []byte, v any) error {
	if len(data) == 0 {
		return nil
	}
	switch m := v.(type) {
	case json.Unmarshaler:
		return m.UnmarshalJSON(data)
	case proto.Message:
		return protojson.UnmarshalOptions{DiscardUnknown: true}.Unmarshal(data, m)
	default:
		return json.Unmarshal(data, v)
	}
}
