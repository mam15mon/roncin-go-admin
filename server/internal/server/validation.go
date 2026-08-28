package server

import (
	"github.com/go-kratos/kratos/v3/middleware"
	"github.com/go-kratos/kratos/v3/middleware/validate"
	"go.einride.tech/aip/fieldbehavior"
	"google.golang.org/protobuf/proto"
)

// RequiredFieldsValidator 统一校验 HTTP 与 gRPC 请求中的 Proto 必填字段。
func RequiredFieldsValidator() middleware.Middleware {
	return validate.Validator(func(request any) error {
		message, ok := request.(proto.Message)
		if !ok {
			return nil
		}
		return fieldbehavior.ValidateRequiredFields(message)
	})
}
