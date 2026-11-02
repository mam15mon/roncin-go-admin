package biz

import (
	"context"
	"testing"
)

type passkeyOriginRepoStub struct {
	PasskeyRepo
	settings  PasskeySettings
	discarded string
}

func (r *passkeyOriginRepoStub) Settings() PasskeySettings { return r.settings }
func (r *passkeyOriginRepoStub) DiscardCeremony(id string) { r.discarded = id }

func TestPasskeyCallerRequiresAllowlistedOriginAndMatchingHost(t *testing.T) {
	repo := &passkeyOriginRepoStub{settings: PasskeySettings{Enabled: true, RPID: "roncin.com.cn", RPName: "Roncin", Origins: []string{"https://ops.roncin.com.cn"}}}
	usecase := NewPasskeyUsecase(repo, nil, nil)
	for _, tc := range []struct {
		name, host, origin string
		configEnabled      bool
	}{
		{"允许来源", "ops.roncin.com.cn", "https://ops.roncin.com.cn", true},
		{"同源GET无Origin头", "ops.roncin.com.cn", "", true},
		{"HTTP IP不允许", "127.0.0.1:8001", "http://127.0.0.1:8001", false},
		{"HTTP业务域不允许", "ops.roncin.com.cn", "http://ops.roncin.com.cn", false},
		{"伪造Origin但Host错误", "127.0.0.1:8001", "https://ops.roncin.com.cn", false},
		{"同主域未授权子域", "other.roncin.com.cn", "https://other.roncin.com.cn", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if config := usecase.Config(PasskeyCaller{Host: tc.host, Origin: tc.origin}); config.Enabled != tc.configEnabled {
				t.Fatalf("来源可用性错误: %+v", config)
			}
		})
	}
	for _, caller := range []PasskeyCaller{{Host: "ops.roncin.com.cn"}, {Host: "127.0.0.1", Origin: "https://ops.roncin.com.cn"}, {Host: "ops.roncin.com.cn", Origin: "https://evil.example.com"}} {
		if _, err := usecase.BeginLogin(context.Background(), caller); err != ErrPasskeyOrigin {
			t.Fatal("不允许来源进入了验签网关")
		}
	}
}

func TestPasskeyFinishInvalidCallerStillDiscardsChallenge(t *testing.T) {
	repo := &passkeyOriginRepoStub{settings: PasskeySettings{Enabled: true, Origins: []string{"https://ops.roncin.com.cn"}}}
	usecase := NewPasskeyUsecase(repo, nil, nil)
	if _, err := usecase.FinishLogin(t.Context(), PasskeyCaller{Host: "127.0.0.1", Origin: "http://127.0.0.1"}, "ceremony", "{}", "agent"); err != ErrPasskeyOrigin {
		t.Fatal("非法来源未拒绝")
	}
	if repo.discarded != "ceremony" {
		t.Fatal("失败finish未消费challenge")
	}
}
