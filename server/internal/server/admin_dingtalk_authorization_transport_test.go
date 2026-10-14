package server

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	kratoshttp "github.com/go-kratos/kratos/v3/transport/http"
	adminv1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
)

type dingTalkAuthorizationTransportService struct {
	adminv1.UnimplementedAdminServiceServer
	single *adminv1.AuthorizeDingTalkUserRequest
	batch  *adminv1.BatchAuthorizeDingTalkUsersRequest
}

func (s *dingTalkAuthorizationTransportService) AuthorizeDingTalkUser(_ context.Context, request *adminv1.AuthorizeDingTalkUserRequest) (*adminv1.AuthorizeDingTalkUserResponse, error) {
	s.single = request
	return &adminv1.AuthorizeDingTalkUserResponse{Success: true, Data: &adminv1.AdminUser{Id: request.GetId()}}, nil
}

func (s *dingTalkAuthorizationTransportService) BatchAuthorizeDingTalkUsers(_ context.Context, request *adminv1.BatchAuthorizeDingTalkUsersRequest) (*adminv1.BatchAuthorizeDingTalkUsersResponse, error) {
	s.batch = request
	return &adminv1.BatchAuthorizeDingTalkUsersResponse{
		Success: true,
		Data: []*adminv1.DingTalkAssignmentResult{
			{UserId: request.GetUserIds()[0], Success: true, Message: "开通成功"},
			{UserId: request.GetUserIds()[1], Reason: "DINGTALK_ASSIGNMENT_CONFLICT", Message: "已被其他管理员更新"},
		},
	}, nil
}

// 使用真实路由、必填字段校验与响应编码，验证前端 items 载荷和省略 false 的逐人结果。
func TestDingTalkMultiOrganizationAuthorizationHTTPContract(t *testing.T) {
	service := &dingTalkAuthorizationTransportService{}
	server := kratoshttp.NewServer(kratoshttp.ResponseEncoder(encodeResponse), kratoshttp.Middleware(RequiredFieldsValidator()))
	adminv1.RegisterAdminServiceHTTPServer(server, service)
	const person1 = "018f0718-9baf-70fd-97eb-749611f7dbe4"
	const person2 = "018f0718-9baf-70fd-97eb-749611f7dbe5"
	const items = `"items":[{"organizationId":"018f0718-9baf-70fd-97eb-749611f7dbe6","roleIds":["018f0718-9baf-70fd-97eb-749611f7dbe7"]},{"organizationId":"018f0718-9baf-70fd-97eb-749611f7dbe8","roleIds":[]}]`
	requests := []struct {
		path string
		body string
	}{
		{"/api/v1/admin/users/" + person1 + "/dingtalk-authorization", "{" + items + "}"},
		{"/api/v1/admin/dingtalk/authorizations", `{"userIds":["` + person1 + `","` + person2 + `"],` + items + "}"},
	}
	for _, input := range requests {
		request := httptest.NewRequest(http.MethodPost, input.path, strings.NewReader(input.body))
		request.Header.Set("Content-Type", "application/json")
		response := httptest.NewRecorder()
		server.ServeHTTP(response, request)
		if response.Code != http.StatusOK {
			t.Fatalf("多组织开通请求被传输层拒绝: %d %s", response.Code, response.Body.String())
		}
		if input.path == "/api/v1/admin/dingtalk/authorizations" {
			var body struct {
				Success bool `json:"success"`
				Data    []struct {
					UserID  string `json:"userId"`
					Success *bool  `json:"success"`
					Reason  string `json:"reason"`
				} `json:"data"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if !body.Success || len(body.Data) != 2 || body.Data[0].Success == nil || !*body.Data[0].Success {
				t.Fatalf("逐人成功结果丢失: %s", response.Body.String())
			}
			if body.Data[1].UserID != person2 || body.Data[1].Success != nil || body.Data[1].Reason != "DINGTALK_ASSIGNMENT_CONFLICT" {
				t.Fatalf("合法失败应省略 false，保留人员与原因供前端重试: %s", response.Body.String())
			}
		}
	}
	if service.single.GetId() != person1 || len(service.batch.GetUserIds()) != 2 {
		t.Fatal("单人与批量人员 ID 未按新契约解码")
	}
	for _, input := range [][]*adminv1.UserMembershipCreation{service.single.GetItems(), service.batch.GetItems()} {
		if len(input) != 2 || len(input[0].GetRoleIds()) != 1 || len(input[1].GetRoleIds()) != 0 || input[0].GetOrganizationId() == input[1].GetOrganizationId() {
			t.Fatalf("逐组织角色未解码或空角色项被必填校验拒绝: %v", input)
		}
	}
}
