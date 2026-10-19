package server

import (
	"context"
	"github.com/go-kratos/kratos/v3/errors"
	"net/http"
	"testing"
	"time"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/service"
	"google.golang.org/protobuf/encoding/protojson"
	"net/http/httptest"
	"strings"
)

type passwordResetServiceRepo struct{}

func (passwordResetServiceRepo) ReservePasswordReset(context.Context, string, *biz.PasswordResetRecord) error {
	return nil
}
func (passwordResetServiceRepo) CompletePasswordResetSend(context.Context, *biz.PasswordResetRecord, bool) (bool, error) {
	return true, nil
}
func (passwordResetServiceRepo) ConsumePasswordReset(context.Context, string, string, func() (string, error)) (bool, error) {
	return true, nil
}
func (passwordResetServiceRepo) RecordPasswordResetSendIP(context.Context, string, time.Time) (bool, error) {
	return false, nil
}

type passwordResetServiceSender struct{}

func (passwordResetServiceSender) Enabled() bool { return true }
func (passwordResetServiceSender) SendMessage(context.Context, string, biz.DingTalkMessage) error {
	return nil
}

// 真实公开路由、camelCase DTO、响应头与 Cookie 边界。
func TestPasswordResetServiceNoStoreAndClearsCookie(t *testing.T) {
	policy := &biz.SessionPolicy{CookieName: "roncin_session", Secure: true, SameSite: "lax"}
	authService := service.NewAuthService(nil, policy, biz.NewPasswordResetUsecase(passwordResetServiceRepo{}, passwordResetServiceSender{}))
	server := kratoshttp.NewServer(kratoshttp.ResponseEncoder(encodeResponse), kratoshttp.ErrorEncoder(encodeError), kratoshttp.Middleware(Authorization(nil, policy, nil, nil), RequiredFieldsValidator()))
	v1.RegisterAuthServiceHTTPServer(server, authService)
	response := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset/code", strings.NewReader(`{"mobile":"13800138000"}`))
	request.Header.Set("Content-Type", "application/json")
	server.ServeHTTP(response, request)
	var reply v1.RequestPasswordResetCodeResponse
	if err := protojson.Unmarshal(response.Body.Bytes(), &reply); err != nil || !reply.Success || reply.Data.ChallengeToken == "" {
		t.Fatalf("发送边界 %v", err)
	}
	if reply.Message != "若该手机号对应已开通的钉钉账号，验证码将发送到本人钉钉。" {
		t.Fatal("成功信封缺少统一的中文发送提示")
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("未禁用缓存")
	}
	confirmed := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset/confirm", strings.NewReader(`{"challengeToken":"`+reply.Data.ChallengeToken+`","code":"012345","newPassword":"long-enough-password"}`))
	request.Header.Set("Content-Type", "application/json")
	server.ServeHTTP(confirmed, request)
	if confirmed.Code != 200 {
		t.Fatalf("确认边界 %d %s", confirmed.Code, confirmed.Body.String())
	}
	cookies := confirmed.Result().Cookies()
	if len(cookies) != 1 || cookies[0].Name != policy.CookieName || cookies[0].Value != "" || cookies[0].MaxAge != -1 || !cookies[0].HttpOnly || !cookies[0].Secure {
		t.Fatal("未正确清除 Cookie")
	}
}
func TestPasswordResetCooldownOnlyExposesRetryTimes(t *testing.T) {
	now := time.Now().UTC()
	writer := httptest.NewRecorder()
	encodeError(writer, httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset/code", nil), biz.PasswordResetCooldown(now, now.Add(time.Minute)))
	if writer.Code != 429 || !strings.Contains(writer.Body.String(), `"serverTime"`) || !strings.Contains(writer.Body.String(), `"resendAvailableAt"`) {
		t.Fatal("缺少倒计时信封")
	}
	writer = httptest.NewRecorder()
	encodeError(writer, httptest.NewRequest(http.MethodPost, "/api/v1/auth/password-reset/code", nil), errors.BadRequest("OTHER", "其他错误").WithMetadata(map[string]string{"secret": "不得输出"}))
	if strings.Contains(writer.Body.String(), "metadata") || strings.Contains(writer.Body.String(), "不得输出") {
		t.Fatal("泄漏任意 metadata")
	}
}
func TestPasswordResetRequiredFieldFailureIsPublicAndNotCached(t *testing.T) {
	policy := &biz.SessionPolicy{CookieName: "roncin_session"}
	authService := service.NewAuthService(nil, policy, biz.NewPasswordResetUsecase(passwordResetServiceRepo{}, passwordResetServiceSender{}))
	server := kratoshttp.NewServer(kratoshttp.ErrorEncoder(encodeError), kratoshttp.Middleware(Authorization(nil, policy, nil, nil), RequiredFieldsValidator()))
	v1.RegisterAuthServiceHTTPServer(server, authService)
	for _, path := range []string{"/api/v1/auth/password-reset/code", "/api/v1/auth/password-reset/confirm"} {
		response := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(`{}`))
		request.Header.Set("Content-Type", "application/json")
		server.ServeHTTP(response, request)
		if response.Code != 400 || response.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("公开校验边界 %d %v", response.Code, response.Header())
		}
	}
}
