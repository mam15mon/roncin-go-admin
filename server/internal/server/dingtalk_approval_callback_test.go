package server

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

type callbackCodecStub struct {
	decodeErr error
}

type callbackRegistrarStub struct{ handler http.HandlerFunc }

func (s *callbackRegistrarStub) HandleFunc(_ string, handler http.HandlerFunc) { s.handler = handler }

func (*callbackCodecStub) Enabled() bool { return true }
func (c *callbackCodecStub) Decode(signature, timestamp, nonce, encrypted string) (*biz.DingTalkApprovalCallbackEvent, error) {
	if c.decodeErr != nil {
		return nil, c.decodeErr
	}
	if signature != "sig" || timestamp != "123" || nonce != "nonce" || encrypted != "cipher" {
		return nil, errors.New("回调参数未完整转交")
	}
	return &biz.DingTalkApprovalCallbackEvent{EventType: "check_url"}, nil
}
func (*callbackCodecStub) EncodeSuccess(string, string) (string, string, error) {
	return "response-cipher", "response-signature", nil
}

func TestDingTalkApprovalCallbackHandlerProtocol(t *testing.T) {
	registrar := &callbackRegistrarStub{}
	uc := biz.NewDingTalkApprovalUsecase(nil, nil, nil, &callbackCodecStub{})
	registerDingTalkApprovalCallback(registrar, uc, slog.New(slog.NewTextHandler(io.Discard, nil)))

	request := httptest.NewRequest(http.MethodPost, dingTalkApprovalCallbackPath+"?msg_signature=sig&timestamp=123&nonce=nonce", strings.NewReader(`{"encrypt":"cipher"}`))
	response := httptest.NewRecorder()
	registrar.handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	for _, value := range []string{`"msg_signature":"response-signature"`, `"timeStamp":"123"`, `"nonce":"nonce"`, `"encrypt":"response-cipher"`} {
		if !strings.Contains(response.Body.String(), value) {
			t.Fatalf("response missing %s: %s", value, response.Body.String())
		}
	}

	invalid := httptest.NewRecorder()
	registrar.handler.ServeHTTP(invalid, httptest.NewRequest(http.MethodPost, dingTalkApprovalCallbackPath+"?msg_signature=sig&timestamp=123&nonce=nonce", strings.NewReader(`{}`)))
	if invalid.Code != http.StatusBadRequest {
		t.Fatalf("invalid status = %d", invalid.Code)
	}

	wrongMethod := httptest.NewRecorder()
	registrar.handler.ServeHTTP(wrongMethod, httptest.NewRequest(http.MethodGet, dingTalkApprovalCallbackPath, nil))
	if wrongMethod.Code != http.StatusMethodNotAllowed {
		t.Fatalf("wrong method status = %d", wrongMethod.Code)
	}
}
