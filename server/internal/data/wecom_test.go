package data

import (
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

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
