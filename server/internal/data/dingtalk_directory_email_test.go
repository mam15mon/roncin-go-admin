package data

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// 两个人员读取入口的三种可选资料均允许缺键、null 和空字符串。
func TestDingTalkDirectoryGatewayOptionalProfile(t *testing.T) {
	for _, independent := range []bool{false, true} {
		for _, field := range []string{"email", "mobile", "avatar"} {
			for _, value := range []struct {
				name string
				want *string
			}{
				{"缺键", nil},
				{"null", nil},
				{"空串", directoryTestString("")},
				{"非空", directoryTestString("current-source-value")},
			} {
				t.Run(fmt.Sprintf("独立=%v/%s/%s", independent, field, value.name), func(t *testing.T) {
					person := map[string]any{"userid": "u1", "unionid": "union1", "name": "资料可选人员", "email": "current@example.test", "mobile": "13800000000", "avatar": "https://example.test/avatar", "dept_id_list": []int64{10}}
					if value.name == "缺键" {
						delete(person, field)
					} else {
						person[field] = value.want
					}
					personJSON, err := json.Marshal(person)
					if err != nil {
						t.Fatal(err)
					}
					provider := &dingTalkIdentityProvider{enabled: true, corpID: "test-corp", appToken: "test-token", appTokenTill: time.Now().Add(time.Hour), client: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
						body := ""
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
							body = `{"errcode":0,"result":{"has_more":false,"list":[` + string(personJSON) + `]}}`
						case "/topapi/v2/user/get":
							body = `{"errcode":0,"result":` + string(personJSON) + `}`
						default:
							t.Fatalf("意外请求路径 %s", r.URL.Path)
						}
						return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}, nil
					})}}
					snapshot, err := provider.ReadDirectory(context.Background())
					if err != nil || len(snapshot.People) != 1 || len(snapshot.Failures) != 0 {
						t.Fatalf("可选资料被误判失败: err=%v", err)
					}
					got := snapshot.People[0]
					if got.Name != "资料可选人员" || got.UserID != "u1" || got.UnionID != "union1" || len(got.Departments) != 1 || got.Departments[0].Name != "来源部门" {
						t.Fatal("可选资料空值阻断其他正常资料")
					}
					for key, actual := range map[string]*string{"email": got.Email, "mobile": got.Mobile, "avatar": got.AvatarURL} {
						var want *string
						if key == field {
							want = value.want
						} else {
							want = directoryTestString(person[key].(string))
						}
						if !reflect.DeepEqual(actual, want) {
							t.Fatalf("%s 来源值未保留", key)
						}
					}
				})
			}
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
