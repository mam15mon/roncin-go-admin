package data

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestDingTalkDirectoryGatewayScopeAndPagination(t *testing.T) {
	calls := 0
	provider := &dingTalkIdentityProvider{enabled: true, corpID: "test-corp", appToken: "test-token", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		var content string
		switch r.URL.Path {
		case "/auth/scopes":
			content = `{"errcode":0,"auth_user_field":["name","avatar","department","jobnumber"],"auth_org_scopes":{"authed_dept":[1],"authed_user":[]}}`
		case "/topapi/v2/department/listsub":
			content = `{"errcode":0,"result":[]}`
		case "/topapi/v2/user/list":
			calls++
			if calls == 1 {
				content = `{"errcode":0,"result":{"has_more":true,"next_cursor":1,"list":[{"userid":"u1","unionid":"union1","name":"员工甲","mobile":"13800000000","job_number":"001","avatar":""}]}}`
			} else {
				content = `{"errcode":0,"result":{"has_more":false,"list":[{"userid":"u2","unionid":"union2","name":"员工乙","avatar":""}]}}`
			}
		default:
			t.Fatalf("意外请求路径 %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(content)), Header: make(http.Header)}, nil
	})}}
	snapshot, err := provider.ReadDirectory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !snapshot.FullScope || len(snapshot.People) != 2 || calls != 2 {
		t.Fatalf("授权/分页结果不正确: %+v", snapshot)
	}
	fields := strings.Join(snapshot.AvailableFields, ",")
	if strings.Contains(fields, "mobile") || strings.Contains(fields, "email") || strings.Contains(fields, "job_number") || !strings.Contains(fields, "avatar_url") {
		t.Fatalf("字段授权错误 %s", fields)
	}
}
func TestDingTalkDirectoryGatewayDeniedIsNotSuccessfulEmptySync(t *testing.T) {
	provider := &dingTalkIdentityProvider{enabled: true, corpID: "test-corp", appToken: "test-token", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"errcode":88}`
		if r.URL.Path == "/auth/scopes" {
			body = `{"errcode":0,"auth_user_field":["name"],"auth_org_scopes":{"authed_dept":[1]}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
	snapshot, err := provider.ReadDirectory(context.Background())
	if err == nil || snapshot != nil {
		t.Fatal("缺少部门读取权限必须整体失败")
	}
}

func TestDingTalkDirectoryGatewayEmptyScopeDoesNotQueryRoot(t *testing.T) {
	calls := 0
	provider := &dingTalkIdentityProvider{enabled: true, corpID: "test-corp", appToken: "test-token", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"errcode":0,"auth_org_scopes":{"authed_dept":[],"authed_user":[]}}`)), Header: make(http.Header)}, nil
	})}}
	if _, err := provider.ReadDirectory(context.Background()); err == nil || calls != 1 {
		t.Fatal("无授权范围不得自动查询根部门")
	}
}

func TestDingTalkDirectoryGatewayMultipleDepartmentsAndPartialFailure(t *testing.T) {
	provider := &dingTalkIdentityProvider{enabled: true, corpID: "test-corp", appToken: "test-token", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := ""
		switch r.URL.Path {
		case "/auth/scopes":
			body = `{"errcode":0,"auth_user_field":["name","department"],"auth_org_scopes":{"authed_dept":[1]}}`
		case "/topapi/v2/department/listsub":
			payload, _ := io.ReadAll(r.Body)
			if strings.Contains(string(payload), `"dept_id":1`) {
				body = `{"errcode":0,"result":[{"dept_id":2,"name":"部门乙"}]}`
			} else {
				body = `{"errcode":88}`
			}
		case "/topapi/v2/user/list":
			payload, _ := io.ReadAll(r.Body)
			if strings.Contains(string(payload), `"dept_id":1`) {
				body = `{"errcode":0,"result":{"has_more":false,"list":[{"userid":"u1","unionid":"union1","name":"跨部门人员"}]}}`
			} else {
				body = `{"errcode":0,"result":{"has_more":false,"list":[{"userid":"u1","unionid":"union1","name":"跨部门人员"}]}}`
			}
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
	snapshot, err := provider.ReadDirectory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.People) != 2 || snapshot.People[1].Departments[0].Name != "部门乙" || len(snapshot.Failures) != 1 || snapshot.Failures[0].DepartmentID != 2 || snapshot.Failures[0].DepartmentName != "部门乙" {
		t.Fatalf("多级部门/部分失败未明确: %+v", snapshot)
	}
}
func TestDingTalkDirectoryGatewayIndependentUserScopeHasDepartmentName(t *testing.T) {
	paths := []string{}
	provider := &dingTalkIdentityProvider{enabled: true, corpID: "test-corp", appToken: "test-token", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		paths = append(paths, r.URL.Path)
		body := ""
		switch r.URL.Path {
		case "/auth/scopes":
			body = `{"errcode":0,"auth_user_field":["name","department","jobnumber"],"auth_org_scopes":{"authed_dept":[],"authed_user":["u1"]}}`
		case "/topapi/v2/user/get":
			body = `{"errcode":0,"result":{"userid":"u1","unionid":"union1","name":"独立授权人员","dept_id_list":[10]}}`
		case "/topapi/v2/department/get":
			body = `{"errcode":0,"result":{"name":"来源部门"}}`
		default:
			t.Fatalf("独立人员授权不应遍历全企业: %s", r.URL.Path)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
	snapshot, err := provider.ReadDirectory(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.FullScope || len(snapshot.People) != 1 || snapshot.People[0].Departments[0].Name != "来源部门" || len(paths) != 3 {
		t.Fatalf("独立人员来源部门缺失: %+v", snapshot)
	}
}

func TestDingTalkDirectoryGatewayIndependentUserReadFailure(t *testing.T) {
	provider := &dingTalkIdentityProvider{enabled: true, corpID: "test-corp", appToken: "test-token", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"errcode":88}`
		if r.URL.Path == "/auth/scopes" {
			body = `{"errcode":0,"auth_user_field":["email","mobile","avatar"],"auth_org_scopes":{"authed_user":["u1"]}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
	snapshot, err := provider.ReadDirectory(context.Background())
	if err != nil || len(snapshot.People) != 0 || len(snapshot.Failures) != 1 {
		t.Fatalf("真实人员接口错误不得当作空资料同步: err=%v", err)
	}
	if snapshot.Failures[0].UserID != "u1" || snapshot.Failures[0].Reason != "DINGTALK_PERSON_READ_FAILED" {
		t.Fatal("真实人员接口错误未保留")
	}
}
