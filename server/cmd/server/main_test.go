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

const dingtalkConfig = `security:
  dingtalk:
    enabled: ${DINGTALK_ENABLED:false}
    client_id: ${DINGTALK_CLIENT_ID:}
    client_secret: ${DINGTALK_CLIENT_SECRET:}
    redirect_uri: ${DINGTALK_REDIRECT_URI:}
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

func TestNewRuntimeConfigResolvesDingTalkEnvironment(t *testing.T) {
	t.Setenv("DINGTALK_ENABLED", "true")
	t.Setenv("DINGTALK_CLIENT_ID", "test-client")
	t.Setenv("DINGTALK_CLIENT_SECRET", "test-secret")
	t.Setenv("DINGTALK_REDIRECT_URI", "http://127.0.0.1:8001/user/login/dingtalk/callback")

	dingtalk := loadDingTalkConfig(t)
	if !dingtalk.Enabled {
		t.Fatal("DINGTALK_ENABLED=true 未解析为启用状态")
	}
	if dingtalk.ClientId != "test-client" || dingtalk.ClientSecret != "test-secret" {
		t.Fatal("钉钉环境变量未完整注入")
	}
}

func TestProductionConfigUsesSafeDefaults(t *testing.T) {
	t.Setenv("DATABASE_SOURCE", "postgres://example.invalid/roncin")
	c := newRuntimeConfig("../../configs/config.production.yaml")
	t.Cleanup(func() { _ = c.Close() })
	if err := c.Load(); err != nil {
		t.Fatalf("加载生产配置失败: %v", err)
	}
	var bootstrap conf.Bootstrap
	if err := c.Scan(&bootstrap); err != nil {
		t.Fatalf("解析生产配置失败: %v", err)
	}
	if bootstrap.GetData().GetDatabase().GetDebug() {
		t.Fatal("生产配置不得启用数据库调试日志")
	}
	if !bootstrap.GetSecurity().GetSession().GetSecure() {
		t.Fatal("生产配置必须启用 Secure 会话 Cookie")
	}
	if bootstrap.GetTelemetry().GetInsecure() {
		t.Fatal("生产配置不得默认使用不安全的 OTLP 连接")
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

func loadDingTalkConfig(t *testing.T) *conf.Security_DingTalk {
	t.Helper()
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(configPath, []byte(dingtalkConfig), 0o600); err != nil {
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
	return bootstrap.Security.Dingtalk
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
