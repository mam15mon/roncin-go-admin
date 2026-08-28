package server

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	authv1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
)

func TestEncodeResponseUsesProtoJSONFieldNames(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	writer := httptest.NewRecorder()
	reply := &authv1.MeResponse{
		Success: true,
		Data: &authv1.CurrentUser{
			CurrentOrganization: &authv1.Organization{Code: "HQ"},
			RoleScopes:          []*authv1.RoleScope{{RoleCode: "administrator", DataScope: "all"}},
		},
	}

	if err := encodeResponse(writer, request, reply); err != nil {
		t.Fatalf("编码响应失败: %v", err)
	}
	body := writer.Body.String()
	for _, field := range []string{`"currentOrganization"`, `"roleScopes"`, `"roleCode"`, `"dataScope"`} {
		if !strings.Contains(body, field) {
			t.Fatalf("响应缺少 ProtoJSON 字段 %s: %s", field, body)
		}
	}
	for _, field := range []string{`"current_organization"`, `"role_scopes"`, `"role_code"`, `"data_scope"`} {
		if strings.Contains(body, field) {
			t.Fatalf("响应包含 snake_case 字段 %s: %s", field, body)
		}
	}
}

func TestEncodeResponseUsesEnumNumbers(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	writer := httptest.NewRecorder()
	reply := &adminv1.ListUsersResponse{
		Success: true,
		Data: []*adminv1.AdminUser{{
			Status: adminv1.AdminUserStatus_ADMIN_USER_STATUS_PENDING_AUTHORIZATION,
		}},
	}

	if err := encodeResponse(writer, request, reply); err != nil {
		t.Fatalf("编码响应失败: %v", err)
	}
	if !strings.Contains(writer.Body.String(), `"status":2`) {
		t.Fatalf("枚举响应未使用 OpenAPI 数字枚举: %s", writer.Body.String())
	}
}

func TestEncodeErrorHidesUnknownInternalError(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/admin/users", nil)
	writer := httptest.NewRecorder()

	encodeError(writer, request, errors.New(`pq: relation "users" does not exist`))

	if writer.Code != 500 {
		t.Fatalf("状态码 = %d，期望 500", writer.Code)
	}
	var response errorEnvelope
	if err := json.Unmarshal(writer.Body.Bytes(), &response); err != nil {
		t.Fatalf("解析错误响应失败: %v", err)
	}
	if response.Message != internalErrorMessage {
		t.Fatalf("错误消息 = %q，期望 %q", response.Message, internalErrorMessage)
	}
	if strings.Contains(writer.Body.String(), "users") {
		t.Fatalf("错误响应泄露底层错误: %s", writer.Body.String())
	}
}
