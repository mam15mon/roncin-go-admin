package server

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/go-kratos/kratos/v3/errors"
)

type errorEnvelope struct {
	Success bool   `json:"success"`
	Code    int32  `json:"code"`
	Message string `json:"message"`
	Reason  string `json:"reason"`
	TraceID string `json:"traceId,omitempty"`
}

const internalErrorMessage = "服务器内部错误"

func encodeError(writer nethttp.ResponseWriter, request *nethttp.Request, err error) {
	serviceError := errors.FromError(err)
	message := serviceError.Message
	if serviceError.Code >= nethttp.StatusInternalServerError && serviceError.Reason == errors.UnknownReason {
		message = internalErrorMessage
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(int(serviceError.Code))
	_ = json.NewEncoder(writer).Encode(errorEnvelope{Success: false, Code: serviceError.Code, Message: message, Reason: serviceError.Reason, TraceID: requestmeta.TraceID(request.Context())})
}
