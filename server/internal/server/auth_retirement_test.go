package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	authv1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
)

type retirementAuthService struct {
	authv1.UnimplementedAuthServiceServer
	calls []string
}

func (s *retirementAuthService) Login(context.Context, *authv1.LoginRequest) (*authv1.LoginResponse, error) {
	s.calls = append(s.calls, "local")
	return &authv1.LoginResponse{Success: true}, nil
}

func (s *retirementAuthService) GetDingTalkLoginConfig(context.Context, *authv1.GetDingTalkLoginConfigRequest) (*authv1.GetDingTalkLoginConfigResponse, error) {
	s.calls = append(s.calls, "dingtalk-config")
	return &authv1.GetDingTalkLoginConfigResponse{Success: true}, nil
}

type retirementAdminService struct {
	adminv1.UnimplementedAdminServiceServer
	called bool
}

func (s *retirementAdminService) AuthorizeDingTalkUser(context.Context, *adminv1.AuthorizeDingTalkUserRequest) (*adminv1.AuthorizeDingTalkUserResponse, error) {
	s.called = true
	return &adminv1.AuthorizeDingTalkUserResponse{Success: true}, nil
}

func TestRetiredIdentityRoutesAreAbsentAndAdjacentRoutesRemain(t *testing.T) {
	server := kratoshttp.NewServer()
	authService, adminService := &retirementAuthService{}, &retirementAdminService{}
	authv1.RegisterAuthServiceHTTPServer(server, authService)
	adminv1.RegisterAdminServiceHTTPServer(server, adminService)
	for _, item := range []struct {
		method string
		path   string
		status int
	}{
		{http.MethodGet, "/api/v1/auth/wecom/login-config", http.StatusNotFound},
		{http.MethodPost, "/api/v1/auth/wecom/login", http.StatusNotFound},
		{http.MethodPost, "/api/v1/admin/users/retired/wecom-authorization", http.StatusNotFound},
		{http.MethodPost, "/api/v1/auth/login", http.StatusOK},
		{http.MethodGet, "/api/v1/auth/dingtalk/login-config", http.StatusOK},
		{http.MethodPost, "/api/v1/admin/users/retained/dingtalk-authorization", http.StatusOK},
	} {
		t.Run(item.path, func(t *testing.T) {
			request := httptest.NewRequest(item.method, item.path, strings.NewReader("{}"))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			server.ServeHTTP(response, request)
			if response.Code != item.status {
				t.Fatalf("请求 %s %s 状态=%d，期望=%d", item.method, item.path, response.Code, item.status)
			}
		})
	}
	if strings.Join(authService.calls, ",") != "local,dingtalk-config" || !adminService.called {
		t.Fatalf("相邻认证与钉钉授权路由未命中对应处理器: auth=%v admin=%v", authService.calls, adminService.called)
	}
}
