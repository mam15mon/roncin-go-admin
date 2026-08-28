package server

import (
	nethttp "net/http"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	"google.golang.org/genproto/googleapis/api/httpbody"
	"google.golang.org/protobuf/proto"
)

func encodeResponse(writer nethttp.ResponseWriter, request *nethttp.Request, value any) error {
	if _, ok := value.(kratoshttp.Redirector); ok {
		return kratoshttp.DefaultResponseEncoder(writer, request, value)
	}
	if _, ok := value.(*httpbody.HttpBody); ok {
		return kratoshttp.DefaultResponseEncoder(writer, request, value)
	}
	message, ok := value.(proto.Message)
	if !ok {
		return kratoshttp.DefaultResponseEncoder(writer, request, value)
	}
	data, err := marshalProtoJSON(message)
	if err != nil {
		return err
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, err = writer.Write(data)
	return err
}
