package server

import (
	"encoding/json"
	"errors"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
)

func TestEncodeResponseUsesProtoJSONFieldNames(t *testing.T) {
	request := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	writer := httptest.NewRecorder()
	reply := &v1.MeResponse{
		Success: true,
		Data: &v1.CurrentUser{
			CurrentOrganization: &v1.Organization{Code: "HQ"},
			RoleScopes:          []*v1.RoleScope{{RoleCode: "administrator", DataScope: "all"}},
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
