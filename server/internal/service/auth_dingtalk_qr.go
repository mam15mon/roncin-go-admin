package service

import (
	"context"

	"github.com/go-kratos/kratos/v3/transport"
	v1 "github.com/roncin/roncin-go-admin/server/api/auth/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *AuthService) PollDingTalkQRCode(ctx context.Context, request *v1.PollDingTalkQRCodeRequest) (*v1.PollDingTalkQRCodeResponse, error) {
	authNoStore(ctx)
	expectedState := ""
	if tr, ok := transport.FromServerContext(ctx); ok {
		expectedState = cookieValue(tr.RequestHeader().Get("Cookie"), s.dingTalkStateCookieName())
	}
	result, err := s.usecase.PollDingTalkQRCode(ctx, request.GetState(), expectedState)
	if err != nil {
		return nil, err
	}
	status := v1.DingTalkQRCodeStatus_DING_TALK_QR_CODE_STATUS_UNSPECIFIED
	switch result.Status {
	case biz.DingTalkQRCodeWaiting:
		status = v1.DingTalkQRCodeStatus_DING_TALK_QR_CODE_STATUS_WAITING
	case biz.DingTalkQRCodeScanned:
		status = v1.DingTalkQRCodeStatus_DING_TALK_QR_CODE_STATUS_SCANNED
	case biz.DingTalkQRCodeConfirmed:
		status = v1.DingTalkQRCodeStatus_DING_TALK_QR_CODE_STATUS_CONFIRMED
	case biz.DingTalkQRCodeExpired:
		status = v1.DingTalkQRCodeStatus_DING_TALK_QR_CODE_STATUS_EXPIRED
	}
	return ok(ctx, &v1.PollDingTalkQRCodeResponse{Data: &v1.DingTalkQRCodePollResult{Status: status, AuthCode: optionalString(result.AuthCode, result.AuthCode != "")}}), nil
}
