package server

import (
	"encoding/json"
	nethttp "net/http"

	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/go-kratos/kratos/v3/errors"
)

type errorEnvelope struct {
	Metadata map[string]string `json:"metadata,omitempty"`
	Success  bool              `json:"success"`
	Code     int32             `json:"code"`
	Message  string            `json:"message"`
	Reason   string            `json:"reason"`
	TraceID  string            `json:"traceId,omitempty"`
}

const internalErrorMessage = "服务器内部错误"

func encodeError(writer nethttp.ResponseWriter, request *nethttp.Request, err error) {
	switch request.URL.Path {
	case "/api/v1/auth/password-reset/code", "/api/v1/auth/password-reset/confirm",
		"/api/v1/auth/dingtalk/embedded-config", "/api/v1/auth/dingtalk/embedded-login":
		writer.Header().Set("Cache-Control", "no-store")
		writer.Header().Set("Pragma", "no-cache")
	}
	serviceError := errors.FromError(err)
	message := serviceError.Message
	if serviceError.Code >= nethttp.StatusInternalServerError && serviceError.Reason == errors.UnknownReason {
		message = internalErrorMessage
	}
	var metadata map[string]string
	if serviceError.Reason == "AUTH_PASSWORD_RESET_COOLDOWN" {
		metadata = map[string]string{"serverTime": serviceError.Metadata["serverTime"], "resendAvailableAt": serviceError.Metadata["resendAvailableAt"]}
	} else {
		metadata = publicErrorMetadata(serviceError.Metadata)
	}
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(int(serviceError.Code))
	_ = json.NewEncoder(writer).Encode(errorEnvelope{Metadata: metadata, Success: false, Code: serviceError.Code, Message: message, Reason: serviceError.Reason, TraceID: requestmeta.TraceID(request.Context())})
}

// publicErrorMetadataKeys 是允许随错误响应返回浏览器的 metadata 键：整表保存的行级定位
// （client_row_id、field、reason、row_errors）与缺汇率提示（currency、week_start、direction）。
// 仅放行这些固定键，避免把其他错误内部携带的元数据整体暴露给前端。
var publicErrorMetadataKeys = []string{"client_row_id", "field", "reason", "row_errors", "currency", "week_start", "direction"}

func publicErrorMetadata(source map[string]string) map[string]string {
	var result map[string]string
	for _, key := range publicErrorMetadataKeys {
		if value, ok := source[key]; ok {
			if result == nil {
				result = make(map[string]string, len(publicErrorMetadataKeys))
			}
			result[key] = value
		}
	}
	return result
}
