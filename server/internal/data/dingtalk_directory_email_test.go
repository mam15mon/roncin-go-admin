package data

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// 部门列表与独立人员详情采用同样的可选邮箱语义，手机和头像仍明确返回。
func TestDingTalkDirectoryGatewayOptionalEmail(t *testing.T) {
	for _, independent := range []bool{false, true} {
		for _, email := range []struct {
			name, field string
			want        *string
		}{
			{"缺键", "", nil},
			{"null", `,"email":null`, nil},
			{"空串", `,"email":""`, directoryTestString("")},
			{"非空", `,"email":"current@example.test"`, directoryTestString("current@example.test")},
		} {
			t.Run(fmt.Sprintf("独立=%v/%s", independent, email.name), func(t *testing.T) {
				provider := &dingTalkIdentityProvider{enabled: true, corpID: "test-corp", appToken: "test-token", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
					body := ""
					person := `{"userid":"u1","unionid":"union1","name":"邮箱可选人员","mobile":"13800000000","avatar":"https://example.test/avatar","dept_id_list":[10]` + email.field + `}`
					switch r.URL.Path {
					case "/auth/scopes":
						if independent {
							body = `{"errcode":0,"auth_user_field":["email","mobile","avatar"],"auth_org_scopes":{"authed_user":["u1"]}}`
						} else {
							body = `{"errcode":0,"auth_user_field":["email","mobile","avatar"],"auth_org_scopes":{"authed_dept":[10]}}`
						}
					case "/topapi/v2/department/get":
						body = `{"errcode":0,"result":{"name":"来源部门"}}`
					case "/topapi/v2/department/listsub":
						body = `{"errcode":0,"result":[]}`
					case "/topapi/v2/user/list":
						body = `{"errcode":0,"result":{"has_more":false,"list":[` + person + `]}}`
					case "/topapi/v2/user/get":
						body = `{"errcode":0,"result":` + person + `}`
					default:
						t.Fatalf("意外请求路径 %s", r.URL.Path)
					}
					return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
				})}}
				snapshot, err := provider.ReadDirectory(context.Background())
				if err != nil || len(snapshot.People) != 1 || len(snapshot.Failures) != 0 {
					t.Fatalf("可选邮箱被误判失败: err=%v", err)
				}
				person := snapshot.People[0]
				if person.Name != "邮箱可选人员" || len(person.Departments) != 1 || person.Departments[0].Name != "来源部门" || person.Mobile == nil || person.AvatarURL == nil {
					t.Fatal("邮箱空值阻断其他正常资料")
				}
				if person.Email == nil != (email.want == nil) || person.Email != nil && *person.Email != *email.want {
					t.Fatal("邮箱来源值未保留")
				}
			})
		}
	}
}

func TestDingTalkDirectoryGatewayIndependentDepartmentReadFailureKeepsKnownUser(t *testing.T) {
	provider := &dingTalkIdentityProvider{enabled: true, corpID: "test-corp", appToken: "test-token", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := `{"errcode":88}`
		switch r.URL.Path {
		case "/auth/scopes":
			body = `{"errcode":0,"auth_user_field":["email"],"auth_org_scopes":{"authed_user":["u1"]}}`
		case "/topapi/v2/user/get":
			body = `{"errcode":0,"result":{"userid":"u1","unionid":"union1","name":"已知人员","dept_id_list":[10]}}`
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
	})}}
	snapshot, err := provider.ReadDirectory(context.Background())
	if err != nil || len(snapshot.Failures) != 1 || len(snapshot.People) != 0 {
		t.Fatalf("真实部门错误被吞掉: err=%v", err)
	}
	f := snapshot.Failures[0]
	if f.UserName != "已知人员" || f.UserID != "u1" || f.DepartmentID != 10 || f.DepartmentName != "" || f.Reason != "DINGTALK_DEPARTMENT_READ_FAILED" {
		t.Fatal("真实失败未保留已知信息或编造部门名称")
	}
}

var _ biz.DingTalkDirectoryGateway = (*dingTalkIdentityProvider)(nil)
