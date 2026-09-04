package server

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

const dingTalkApprovalCallbackPath = "/api/integrations/dingtalk/order-unlock/events"

type dingTalkEncryptedCallbackRequest struct {
	Encrypt string `json:"encrypt"`
}

type dingTalkEncryptedCallbackResponse struct {
	MessageSignature string `json:"msg_signature"`
	Timestamp        string `json:"timeStamp"`
	Nonce            string `json:"nonce"`
	Encrypt          string `json:"encrypt"`
}

func registerDingTalkApprovalCallback(srv interface {
	HandleFunc(string, http.HandlerFunc)
}, usecase *biz.DingTalkApprovalUsecase, logger *slog.Logger) {
	srv.HandleFunc(dingTalkApprovalCallbackPath, func(response http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			response.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if !usecase.CallbackEnabled() {
			response.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		body, err := io.ReadAll(io.LimitReader(request.Body, 1<<20))
		if err != nil {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		var payload dingTalkEncryptedCallbackRequest
		if err := json.Unmarshal(body, &payload); err != nil || strings.TrimSpace(payload.Encrypt) == "" {
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		query := request.URL.Query()
		signature := query.Get("msg_signature")
		if signature == "" {
			signature = query.Get("signature")
		}
		timestamp := query.Get("timestamp")
		nonce := query.Get("nonce")
		encrypted, responseSignature, err := usecase.ReceiveCallback(request.Context(), signature, timestamp, nonce, payload.Encrypt)
		if err != nil {
			logger.Warn("reject dingtalk approval callback", slog.Any("error", err))
			response.WriteHeader(http.StatusBadRequest)
			return
		}
		response.Header().Set("Content-Type", "application/json; charset=utf-8")
		response.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(response).Encode(dingTalkEncryptedCallbackResponse{
			MessageSignature: responseSignature,
			Timestamp:        timestamp,
			Nonce:            nonce,
			Encrypt:          encrypted,
		})
	})
}
