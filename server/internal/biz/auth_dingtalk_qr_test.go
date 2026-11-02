package biz

import (
	"context"
	"strings"
	"testing"
)

type qrPollProviderStub struct {
	dingTalkProviderStub
	calls int
	state string
}

func (p *qrPollProviderStub) PollQRCode(_ context.Context, state string) (*DingTalkQRCodePollResult, error) {
	p.calls++
	p.state = state
	return &DingTalkQRCodePollResult{Status: DingTalkQRCodeWaiting}, nil
}

func TestAuthUsecasePollDingTalkQRCodeBindsBrowserBeforeUpstream(t *testing.T) {
	for _, tc := range []struct {
		name, state, cookie string
		enabled, allowed    bool
	}{
		{"匹配", "current", "current", true, true},
		{"刷新后的旧请求", "old", "current", true, false},
		{"无Cookie", "current", "", true, false},
		{"空state", "", "current", true, false},
		{"过长state", strings.Repeat("a", 513), strings.Repeat("a", 513), true, false},
		{"禁用", "current", "current", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			provider := &qrPollProviderStub{dingTalkProviderStub: dingTalkProviderStub{enabled: tc.enabled}}
			usecase := NewAuthUsecase(nil, nil, provider, nil)
			result, err := usecase.PollDingTalkQRCode(t.Context(), tc.state, tc.cookie)
			if tc.allowed {
				if err != nil || result.Status != DingTalkQRCodeWaiting || provider.calls != 1 || provider.state != tc.state {
					t.Fatalf("合法轮询失败: %+v %v", result, err)
				}
			} else if err == nil || provider.calls != 0 {
				t.Fatalf("无效状态访问了上游: %v %d", err, provider.calls)
			}
		})
	}
}
