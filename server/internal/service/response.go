package service

import (
	"context"

	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type successfulResponse interface {
	proto.Message
	GetSuccess() bool
	GetMessage() string
	GetTraceId() string
}

// ok 统一写入所有成功响应共有的封套字段，具体响应仍由调用方保持静态类型。
func ok[T successfulResponse](ctx context.Context, response T) T {
	message := response.ProtoReflect()
	fields := message.Descriptor().Fields()
	message.Set(fields.ByName(protoreflect.Name("success")), protoreflect.ValueOfBool(true))
	message.Set(fields.ByName(protoreflect.Name("message")), protoreflect.ValueOfString("OK"))
	message.Set(fields.ByName(protoreflect.Name("trace_id")), protoreflect.ValueOfString(requestmeta.TraceID(ctx)))
	return response
}

// okList 标记列表响应；各接口不一致的 total/page/page_size 字段由具体类型赋值。
func okList[T successfulResponse](ctx context.Context, response T) T {
	return ok(ctx, response)
}
