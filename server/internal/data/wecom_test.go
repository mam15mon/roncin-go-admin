package data

import (
	"encoding/json"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestWeComUserIdentityResponse(t *testing.T) {
	var response wecomUserIdentityResponse
	if err := json.Unmarshal([]byte(`{"errcode":0,"errmsg":"ok","userid":"test-user"}`), &response); err != nil {
		t.Fatalf("解析企业微信身份响应失败: %v", err)
	}
	if response.ErrCode != 0 || response.UserID != "test-user" {
		t.Fatalf("企业微信身份响应解析错误: %+v", response)
	}
}

func TestMapWeComAPIError(t *testing.T) {
	tests := []struct {
		name string
		code int
		want error
	}{
		{name: "登录凭证失效", code: 40029, want: biz.ErrWeComCodeInvalid},
		{name: "可信 IP 未配置", code: 60020, want: biz.ErrWeComTrustedIPRequired},
		{name: "接口无权限", code: 48002, want: biz.ErrWeComPermissionDenied},
		{name: "其他错误", code: -1, want: biz.ErrWeComLoginFailed},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := mapWeComAPIError("test", test.code); got != test.want {
				t.Fatalf("mapWeComAPIError() = %v, want %v", got, test.want)
			}
		})
	}
}
