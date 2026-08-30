package biz

import (
	"context"
	"testing"
)

func TestRequirePrincipal(t *testing.T) {
	principal := &Principal{Username: "tester"}
	actual, err := RequirePrincipal(WithPrincipal(context.Background(), principal))
	if err != nil {
		t.Fatalf("读取登录主体失败: %v", err)
	}
	if actual != principal {
		t.Fatal("读取到的登录主体与上下文不一致")
	}
}

func TestRequirePrincipalRejectsMissingPrincipal(t *testing.T) {
	actual, err := RequirePrincipal(context.Background())
	if err != ErrSessionRequired {
		t.Fatalf("缺少登录主体时错误 = %v，期望 ErrSessionRequired", err)
	}
	if actual != nil {
		t.Fatal("缺少登录主体时不应返回主体")
	}
}
