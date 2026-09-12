package service

import (
	"context"
	"testing"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/google/uuid"
)

func TestDingTalkInvitationToAPIMasksMobile(t *testing.T) {
	invitation := &biz.DingTalkInvitation{
		ID:               uuid.New(),
		OrganizationID:   uuid.New(),
		OrganizationName: "成都公司",
		RoleID:           uuid.New(),
		RoleName:         "操作员",
		Mobile:           "13800138000",
		DisplayName:      "备注姓名",
		Status:           biz.DingTalkInvitationStatusPending,
		CreatedAt:        time.Now().UTC(),
		ExpiresAt:        time.Now().UTC().Add(time.Hour),
	}

	api := dingTalkInvitationToAPI(invitation)
	if api.MobileMasked != "138****8000" {
		t.Fatalf("手机号应脱敏输出，实际 %q", api.MobileMasked)
	}
	if api.MobileMasked == invitation.Mobile {
		t.Fatal("完整手机号不得离开服务端")
	}
	if api.DisplayName == nil || *api.DisplayName != "备注姓名" {
		t.Fatalf("备注姓名 = %#v", api.DisplayName)
	}
	if api.Status != v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_PENDING {
		t.Fatalf("状态映射 = %v", api.Status)
	}
}

func TestDingTalkInvitationStatusRoundTrip(t *testing.T) {
	pairs := []struct {
		biz   biz.DingTalkInvitationStatus
		proto v1.DingTalkInvitationStatus
	}{
		{biz.DingTalkInvitationStatusPending, v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_PENDING},
		{biz.DingTalkInvitationStatusConsumed, v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_CONSUMED},
		{biz.DingTalkInvitationStatusExpired, v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_EXPIRED},
		{biz.DingTalkInvitationStatusRevoked, v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_REVOKED},
	}
	for _, pair := range pairs {
		bizStatus, err := dingTalkInvitationStatusFromAPI(pair.proto)
		if err != nil || bizStatus != pair.biz {
			t.Fatalf("proto→biz = (%v, %v)，期望 %v", bizStatus, err, pair.biz)
		}
		if dingTalkInvitationStatusToAPI(pair.biz) != pair.proto {
			t.Fatalf("biz→proto = %v，期望 %v", dingTalkInvitationStatusToAPI(pair.biz), pair.proto)
		}
	}
	if _, err := dingTalkInvitationStatusFromAPI(v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_UNSPECIFIED); err != biz.ErrAdminInvalidArgument {
		t.Fatalf("未指定状态错误 = %v，期望 ErrAdminInvalidArgument", err)
	}
}

func TestDingTalkRegistrationToAPIIncludesRequestedOrganization(t *testing.T) {
	requested := uuid.New()
	registration := &biz.DingTalkRegistration{
		UserID:                    uuid.New(),
		DisplayName:               "认领员工",
		RequestedOrganizationID:   &requested,
		RequestedOrganizationName: "成都公司",
		RegisteredAt:              time.Now().UTC(),
	}
	api := dingTalkRegistrationToAPI(registration)
	if api.RequestedOrganizationId == nil || *api.RequestedOrganizationId != requested.String() {
		t.Fatalf("目标组织 = %#v", api.RequestedOrganizationId)
	}
	if api.RequestedOrganizationName != "成都公司" || api.DisplayName != "认领员工" {
		t.Fatalf("注册视图 = %#v", api)
	}
	fallback := dingTalkRegistrationToAPI(&biz.DingTalkRegistration{UserID: uuid.New(), DisplayName: "总部兜底"})
	if fallback.RequestedOrganizationId != nil {
		t.Fatalf("兜底注册不应携带目标组织: %#v", fallback.RequestedOrganizationId)
	}
}

func TestAdminServiceCreateDingTalkInvitationRejectsInvalidUUID(t *testing.T) {
	service := NewAdminService(nil, nil)
	ctx := biz.WithPrincipal(context.Background(), &biz.Principal{UserID: uuid.New()})
	if _, err := service.CreateDingTalkInvitation(ctx, &v1.CreateDingTalkInvitationRequest{
		Mobile:         "13800138000",
		OrganizationId: "not-a-uuid",
		RoleId:         uuid.NewString(),
	}); err != biz.ErrAdminInvalidArgument {
		t.Fatalf("非法组织 ID 错误 = %v，期望 ErrAdminInvalidArgument", err)
	}
	if _, err := service.CreateDingTalkInvitation(ctx, &v1.CreateDingTalkInvitationRequest{
		Mobile:         "13800138000",
		OrganizationId: uuid.NewString(),
		RoleId:         "not-a-uuid",
	}); err != biz.ErrAdminInvalidArgument {
		t.Fatalf("非法角色 ID 错误 = %v，期望 ErrAdminInvalidArgument", err)
	}
}
