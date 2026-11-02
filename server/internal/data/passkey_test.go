package data

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

func passkeyTestSettings() *conf.Security {
	return &conf.Security{Passkey: &conf.Security_Passkey{Enabled: true, RpId: "roncin.com.cn", RpName: "Roncin 后台", Origins: []string{"https://ops.roncin.com.cn"}}}
}

func TestPasskeyConfigurationRequiresExplicitSafeRPAndOrigins(t *testing.T) {
	for _, tc := range []struct {
		name, rp, origin string
		valid            bool
	}{
		{"主域子域", "roncin.com.cn", "https://ops.roncin.com.cn", true},
		{"同RP域", "roncin.com.cn", "https://roncin.com.cn", true},
		{"localhost开发", "localhost", "http://localhost:8001", true},
		{"HTTP IP", "127.0.0.1", "http://127.0.0.1:8001", false},
		{"HTTP业务域", "roncin.com.cn", "http://ops.roncin.com.cn", false},
		{"错误域后缀", "roncin.com.cn", "https://evilroncin.com.cn", false},
		{"站外来源", "roncin.com.cn", "https://example.com", false},
		{"来源含路径", "roncin.com.cn", "https://ops.roncin.com.cn/path", false},
		{"来源含用户信息", "roncin.com.cn", "https://user@ops.roncin.com.cn", false},
		{"来源含query", "roncin.com.cn", "https://ops.roncin.com.cn?other=true", false},
		{"公共后缀", "com.cn", "https://ops.com.cn", false},
		{"RP含端口", "roncin.com.cn:443", "https://ops.roncin.com.cn", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := NewPasskeyRepo(nil, &conf.Security{Passkey: &conf.Security_Passkey{Enabled: true, RpId: tc.rp, RpName: "Roncin", Origins: []string{tc.origin}}})
			if (err == nil) != tc.valid {
				t.Fatalf("配置校验错误: %v", err)
			}
		})
	}
	if _, err := NewPasskeyRepo(nil, &conf.Security{Passkey: &conf.Security_Passkey{Enabled: false, RpId: "invalid value"}}); err != nil {
		t.Fatal("禁用功能阻止了普通登录启动")
	}
}

func TestPasskeyCeremoniesRequireUVAndAreOneTimePurposeBound(t *testing.T) {
	source, err := NewPasskeyRepo(nil, passkeyTestSettings())
	if err != nil {
		t.Fatal(err)
	}
	repo := source.(*passkeyRepo)
	ceremony, err := repo.BeginLogin(t.Context(), "https://ops.roncin.com.cn")
	if err != nil {
		t.Fatal(err)
	}
	var options map[string]any
	if err = json.Unmarshal([]byte(ceremony.OptionsJSON), &options); err != nil {
		t.Fatal(err)
	}
	if options["userVerification"] != "required" || options["rpId"] != "roncin.com.cn" || options["challenge"] == nil || options["publicKey"] != nil {
		t.Fatal("发现式认证选项不正确")
	}
	if _, err = repo.takeCeremony(ceremony.ID, "registration", uuid.New()); err != biz.ErrPasskeyInvalid {
		t.Fatal("登录challenge可跨用途消费")
	}
	if _, err = repo.takeCeremony(ceremony.ID, "login", uuid.Nil); err != biz.ErrPasskeyInvalid {
		t.Fatal("失败消费后challenge可重放")
	}
	ceremony, err = repo.BeginLogin(t.Context(), "https://ops.roncin.com.cn")
	if err != nil {
		t.Fatal(err)
	}
	repo.mu.Lock()
	item := repo.sessions[ceremony.ID]
	item.expires = time.Now().Add(-time.Second)
	repo.sessions[ceremony.ID] = item
	repo.mu.Unlock()
	if _, err = repo.takeCeremony(ceremony.ID, "login", uuid.Nil); err != biz.ErrPasskeyInvalid {
		t.Fatal("过期challenge仍有效")
	}
	if len(repo.sessions) != 0 {
		t.Fatal("终态challenge没有释放")
	}
}

func TestPasskeyCeremonyCacheIsBoundedAndConsumesOnce(t *testing.T) {
	source, err := NewPasskeyRepo(nil, passkeyTestSettings())
	if err != nil {
		t.Fatal(err)
	}
	repo := source.(*passkeyRepo)
	for i := range passkeyCeremonyCapacity {
		repo.sessions[fmt.Sprint(i)] = passkeySession{expires: time.Now().Add(time.Minute)}
	}
	if _, err = repo.BeginLogin(t.Context(), "https://ops.roncin.com.cn"); err != biz.ErrPasskeyBusy {
		t.Fatal("仪式缓存超过固定容量")
	}
	for key := range repo.sessions {
		repo.sessions[key] = passkeySession{expires: time.Now().Add(-time.Second)}
	}
	ceremony, err := repo.BeginLogin(t.Context(), "https://ops.roncin.com.cn")
	if err != nil {
		t.Fatal(err)
	}
	if len(repo.sessions) != 1 {
		t.Fatal("过期仪式没有清理")
	}
	results := make(chan error, 16)
	for range 16 {
		go func() { _, err := repo.takeCeremony(ceremony.ID, "login", uuid.Nil); results <- err }()
	}
	successes := 0
	for range 16 {
		if err := <-results; err == nil {
			successes++
		} else if err != biz.ErrPasskeyInvalid {
			t.Fatal(err)
		}
	}
	if successes != 1 {
		t.Fatalf("同challenge并发消费成功%d次", successes)
	}
}
