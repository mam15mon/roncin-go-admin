package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

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
    corp_id: ${DINGTALK_CORP_ID:}
    client_id: ${DINGTALK_CLIENT_ID:}
    client_secret: ${DINGTALK_CLIENT_SECRET:}
    redirect_uri: ${DINGTALK_REDIRECT_URI:}
    registration_token_secret: ${DINGTALK_REGISTRATION_TOKEN_SECRET:}
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
	t.Setenv("DINGTALK_CORP_ID", "ding-test-corp")
	t.Setenv("DINGTALK_CLIENT_ID", "test-client")
	t.Setenv("DINGTALK_CLIENT_SECRET", "test-secret")
	t.Setenv("DINGTALK_REDIRECT_URI", "http://127.0.0.1:8001/user/login/dingtalk/callback")
	t.Setenv("DINGTALK_REGISTRATION_TOKEN_SECRET", "test-registration-token-secret-32-bytes")

	dingtalk := loadDingTalkConfig(t)
	if !dingtalk.Enabled {
		t.Fatal("DINGTALK_ENABLED=true 未解析为启用状态")
	}
	if dingtalk.CorpId != "ding-test-corp" || dingtalk.ClientId != "test-client" || dingtalk.ClientSecret != "test-secret" {
		t.Fatal("钉钉环境变量未完整注入")
	}
	if dingtalk.RegistrationTokenSecret != "test-registration-token-secret-32-bytes" {
		t.Fatal("钉钉注册凭证密钥未注入")
	}
}

func TestProductionConfigUsesSafeDefaults(t *testing.T) {
	t.Setenv("DATABASE_SOURCE", "postgres://example.invalid/roncin")
	unsetEnv(t, "HTTP_TIMEOUT")
	unsetEnv(t, "GRPC_TIMEOUT")
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
	if bootstrap.GetLogging().GetLevel() != "info" {
		t.Fatalf("生产日志级别应默认为 info，实际为 %q", bootstrap.GetLogging().GetLevel())
	}
	if !bootstrap.GetSecurity().GetSession().GetSecure() {
		t.Fatal("生产配置必须启用 Secure 会话 Cookie")
	}
	if bootstrap.GetTelemetry().GetInsecure() {
		t.Fatal("生产配置不得默认使用不安全的 OTLP 连接")
	}
	if got := bootstrap.GetServer().GetHttp().GetTimeout().AsDuration(); got != 30*time.Second {
		t.Fatalf("生产 HTTP 默认超时应为 30 秒，实际为 %s", got)
	}
	if got := bootstrap.GetServer().GetGrpc().GetTimeout().AsDuration(); got != 30*time.Second {
		t.Fatalf("生产 gRPC 默认超时应为 30 秒，实际为 %s", got)
	}
}

func TestDevelopmentConfigSupportsExternalIdentityLogin(t *testing.T) {
	t.Setenv("DATABASE_SOURCE", "postgres://example.invalid/roncin")
	c := newRuntimeConfig("../../configs/config.yaml")
	t.Cleanup(func() { _ = c.Close() })
	if err := c.Load(); err != nil {
		t.Fatalf("加载开发配置失败: %v", err)
	}
	var bootstrap conf.Bootstrap
	if err := c.Scan(&bootstrap); err != nil {
		t.Fatalf("解析开发配置失败: %v", err)
	}
	if got := bootstrap.GetServer().GetHttp().GetTimeout().AsDuration(); got != 30*time.Second {
		t.Fatalf("开发 HTTP 超时应允许外部身份认证完成，实际为 %s", got)
	}
	if bootstrap.GetLogging().GetLevel() != "debug" {
		t.Fatalf("开发日志级别应默认为 debug，实际为 %q", bootstrap.GetLogging().GetLevel())
	}
}

func TestParseLogLevel(t *testing.T) {
	tests := []struct {
		value string
		want  string
	}{
		{value: "debug", want: "DEBUG"},
		{value: "info", want: "INFO"},
		{value: "warn", want: "WARN"},
		{value: "error", want: "ERROR"},
	}
	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			level, err := parseLogLevel(tt.value)
			if err != nil {
				t.Fatalf("解析日志级别失败: %v", err)
			}
			if level.String() != tt.want {
				t.Fatalf("日志级别 = %q，期望 %q", level.String(), tt.want)
			}
		})
	}
	if _, err := parseLogLevel("verbose"); err == nil {
		t.Fatal("非法日志级别应返回错误")
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
