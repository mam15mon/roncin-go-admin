package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/conf"
)

const wecomConfig = `security:
  wecom:
    enabled: ${WECOM_ENABLED:false}
    corp_id: ${WECOM_CORP_ID:}
    agent_id: ${WECOM_AGENT_ID:0}
    secret: ${WECOM_SECRET:}
    redirect_uri: ${WECOM_REDIRECT_URI:}
`

func TestNewRuntimeConfigResolvesWeComEnvironment(t *testing.T) {
	t.Setenv("WECOM_ENABLED", "true")
	t.Setenv("WECOM_CORP_ID", "test-corp")
	t.Setenv("WECOM_AGENT_ID", "1000001")
	t.Setenv("WECOM_SECRET", "test-secret")
	t.Setenv("WECOM_REDIRECT_URI", "http://127.0.0.1:8001/user/login/wecom/callback")

	wecom := loadWeComConfig(t)
	if !wecom.Enabled {
		t.Fatal("WECOM_ENABLED=true 未解析为启用状态")
	}
	if wecom.CorpId != "test-corp" || wecom.AgentId != 1000001 || wecom.Secret != "test-secret" {
		t.Fatal("企业微信环境变量未完整注入")
	}
}

func TestNewRuntimeConfigUsesDisabledDefault(t *testing.T) {
	unsetEnv(t, "WECOM_ENABLED")

	wecom := loadWeComConfig(t)
	if wecom.Enabled {
		t.Fatal("缺少 WECOM_ENABLED 时应保持关闭")
	}
}

func loadWeComConfig(t *testing.T) *conf.Security_WeCom {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(wecomConfig), 0o600); err != nil {
		t.Fatalf("写入测试配置失败: %v", err)
	}

	c := newRuntimeConfig(configPath)
	t.Cleanup(func() { _ = c.Close() })
	if err := c.Load(); err != nil {
		t.Fatalf("加载测试配置失败: %v", err)
	}

	var bootstrap conf.Bootstrap
	if err := c.Scan(&bootstrap); err != nil {
		t.Fatalf("解析测试配置失败: %v", err)
	}
	return bootstrap.Security.Wecom
}

func unsetEnv(t *testing.T, key string) {
	t.Helper()
	value, exists := os.LookupEnv(key)
	if err := os.Unsetenv(key); err != nil {
		t.Fatalf("清理测试环境变量失败: %v", err)
	}
	t.Cleanup(func() {
		if exists {
			_ = os.Setenv(key, value)
			return
		}
		_ = os.Unsetenv(key)
	})
}
