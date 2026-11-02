package biz

import (
	"context"
	"crypto/subtle"
	"strings"
)

// DingTalkQRCode 只包含可供浏览器绘制的扫码确认链接，上游会话留在网关内部。
type DingTalkQRCode struct {
	URL string
}

type DingTalkQRCodeStatus string

const (
	DingTalkQRCodeWaiting   DingTalkQRCodeStatus = "WAITING"
	DingTalkQRCodeScanned   DingTalkQRCodeStatus = "SCANNED"
	DingTalkQRCodeConfirmed DingTalkQRCodeStatus = "CONFIRMED"
	DingTalkQRCodeExpired   DingTalkQRCodeStatus = "EXPIRED"
)

type DingTalkQRCodePollResult struct {
	Status   DingTalkQRCodeStatus
	AuthCode string
}

// PollDingTalkQRCode 在访问上游前绑定本浏览器本次尝试，换码仍走既有身份准入。
func (uc *AuthUsecase) PollDingTalkQRCode(ctx context.Context, state, expectedState string) (*DingTalkQRCodePollResult, error) {
	if !uc.dingtalk.Enabled() {
		return nil, ErrDingTalkDisabled
	}
	state = strings.TrimSpace(state)
	expectedState = strings.TrimSpace(expectedState)
	if state == "" || expectedState == "" || len(state) > 512 || subtle.ConstantTimeCompare([]byte(state), []byte(expectedState)) != 1 {
		return nil, ErrDingTalkStateInvalid
	}
	return uc.dingtalk.PollQRCode(ctx, state)
}
