package data

import (
	"strings"
	"testing"
)

func TestRedactEntDebugMessageHidesArguments(t *testing.T) {
	message := `driver.Query: query=SELECT * FROM "sessions" WHERE "token_hash" = $1 args=[secret-session-hash]`
	redacted := redactEntDebugMessage(message)
	if strings.Contains(redacted, "secret-session-hash") {
		t.Fatalf("Ent 调试日志仍包含查询参数: %s", redacted)
	}
	if !strings.Contains(redacted, `SELECT * FROM "sessions"`) || !strings.HasSuffix(redacted, "args=[REDACTED]") {
		t.Fatalf("Ent 调试日志脱敏结果不正确: %s", redacted)
	}
}
