package service

import (
	"context"
	"strings"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/admin/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/google/uuid"
)

// CreateDingTalkInvitation 预建扫码邀请（通道 A）：目标组织必须在调用者按
// 「钉钉邀请与注册审批」权限解析的可写范围内；手机号归一化后入库，响应只回脱敏形式。
func (s *AdminService) CreateDingTalkInvitation(ctx context.Context, request *v1.CreateDingTalkInvitationRequest) (*v1.CreateDingTalkInvitationResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	organizationID, err := uuid.Parse(strings.TrimSpace(request.GetOrganizationId()))
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	roleID, err := uuid.Parse(strings.TrimSpace(request.GetRoleId()))
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	displayName := ""
	if request.DisplayName != nil {
		displayName = request.GetDisplayName()
	}
	created, err := s.dingTalkRegistrations.CreateInvitation(ctx, principal, request.GetMobile(), displayName, organizationID, roleID, int(request.GetExpiresInHours()))
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.CreateDingTalkInvitationResponse{Data: dingTalkInvitationToAPI(created)}), nil
}

func (s *AdminService) ListDingTalkInvitations(ctx context.Context, request *v1.ListDingTalkInvitationsRequest) (*v1.ListDingTalkInvitationsResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrAdminInvalidArgument)
	if err != nil {
		return nil, err
	}
	options := biz.DingTalkInvitationListOptions{Page: page, PageSize: pageSize}
	if request.OrganizationId != nil && strings.TrimSpace(request.GetOrganizationId()) != "" {
		organizationID, parseErr := uuid.Parse(strings.TrimSpace(request.GetOrganizationId()))
		if parseErr != nil {
			return nil, biz.ErrAdminInvalidArgument
		}
		options.OrganizationID = organizationID
	}
	if request.Status != nil {
		status, statusErr := dingTalkInvitationStatusFromAPI(request.GetStatus())
		if statusErr != nil {
			return nil, statusErr
		}
		options.Status = &status
	}
	list, err := s.dingTalkRegistrations.ListInvitations(ctx, principal, options)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.DingTalkInvitation, 0, len(list.Items))
	for _, item := range list.Items {
		data = append(data, dingTalkInvitationToAPI(item))
	}
	return okList(ctx, &v1.ListDingTalkInvitationsResponse{Data: data, Total: int32(list.Total), Page: int32(list.Page), PageSize: int32(list.PageSize)}), nil
}

func (s *AdminService) RevokeDingTalkInvitation(ctx context.Context, request *v1.RevokeDingTalkInvitationRequest) (*v1.RevokeDingTalkInvitationResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	invitationID, err := uuid.Parse(strings.TrimSpace(request.GetId()))
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	if err := s.dingTalkRegistrations.RevokeInvitation(ctx, principal, invitationID); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RevokeDingTalkInvitationResponse{}), nil
}

func (s *AdminService) ListDingTalkRegistrations(ctx context.Context, request *v1.ListDingTalkRegistrationsRequest) (*v1.ListDingTalkRegistrationsResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	page, pageSize, err := listPageValues(request.GetPage(), request.GetPageSize(), biz.ErrAdminInvalidArgument)
	if err != nil {
		return nil, err
	}
	list, err := s.dingTalkRegistrations.ListPendingRegistrations(ctx, principal, biz.DingTalkRegistrationListOptions{Page: page, PageSize: pageSize})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.DingTalkRegistration, 0, len(list.Items))
	for _, item := range list.Items {
		data = append(data, dingTalkRegistrationToAPI(item))
	}
	return okList(ctx, &v1.ListDingTalkRegistrationsResponse{Data: data, Total: int32(list.Total), Page: int32(list.Page), PageSize: int32(list.PageSize)}), nil
}

func (s *AdminService) ApproveDingTalkRegistration(ctx context.Context, request *v1.ApproveDingTalkRegistrationRequest) (*v1.ApproveDingTalkRegistrationResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(strings.TrimSpace(request.GetId()))
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	roleIDs, err := parseUniqueUUIDValues(request.GetRoleIds(), biz.ErrAdminInvalidArgument)
	if err != nil {
		return nil, err
	}
	approved, err := s.dingTalkRegistrations.ApproveRegistration(ctx, principal, userID, roleIDs)
	if err != nil {
		return nil, err
	}
	return ok(ctx, &v1.ApproveDingTalkRegistrationResponse{Data: dingTalkRegistrationToAPI(approved)}), nil
}

func (s *AdminService) RejectDingTalkRegistration(ctx context.Context, request *v1.RejectDingTalkRegistrationRequest) (*v1.RejectDingTalkRegistrationResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(strings.TrimSpace(request.GetId()))
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	if err := s.dingTalkRegistrations.RejectRegistration(ctx, principal, userID, request.GetReason()); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.RejectDingTalkRegistrationResponse{}), nil
}

func dingTalkInvitationStatusFromAPI(value v1.DingTalkInvitationStatus) (biz.DingTalkInvitationStatus, error) {
	switch value {
	case v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_PENDING:
		return biz.DingTalkInvitationStatusPending, nil
	case v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_CONSUMED:
		return biz.DingTalkInvitationStatusConsumed, nil
	case v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_EXPIRED:
		return biz.DingTalkInvitationStatusExpired, nil
	case v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_REVOKED:
		return biz.DingTalkInvitationStatusRevoked, nil
	default:
		return "", biz.ErrAdminInvalidArgument
	}
}

func dingTalkInvitationStatusToAPI(value biz.DingTalkInvitationStatus) v1.DingTalkInvitationStatus {
	switch value {
	case biz.DingTalkInvitationStatusPending:
		return v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_PENDING
	case biz.DingTalkInvitationStatusConsumed:
		return v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_CONSUMED
	case biz.DingTalkInvitationStatusExpired:
		return v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_EXPIRED
	case biz.DingTalkInvitationStatusRevoked:
		return v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_REVOKED
	default:
		return v1.DingTalkInvitationStatus_DING_TALK_INVITATION_STATUS_UNSPECIFIED
	}
}

// dingTalkInvitationToAPI 输出邀请视图；手机号在此统一脱敏，完整号码不出服务端。
func dingTalkInvitationToAPI(value *biz.DingTalkInvitation) *v1.DingTalkInvitation {
	result := &v1.DingTalkInvitation{
		Id:               value.ID.String(),
		OrganizationId:   value.OrganizationID.String(),
		OrganizationName: value.OrganizationName,
		RoleId:           value.RoleID.String(),
		RoleName:         value.RoleName,
		MobileMasked:     biz.MaskDingTalkMobile(value.Mobile),
		Status:           dingTalkInvitationStatusToAPI(value.Status),
		CreatedAt:        value.CreatedAt.Format(time.RFC3339),
		ExpiresAt:        value.ExpiresAt.Format(time.RFC3339),
		InviterName:      value.InviterName,
	}
	if value.DisplayName != "" {
		result.DisplayName = &value.DisplayName
	}
	if value.ConsumedByName != "" {
		result.ConsumedName = &value.ConsumedByName
	}
	return result
}

func dingTalkRegistrationToAPI(value *biz.DingTalkRegistration) *v1.DingTalkRegistration {
	result := &v1.DingTalkRegistration{
		UserId:       value.UserID.String(),
		DisplayName:  value.DisplayName,
		AvatarUrl:    value.AvatarURL,
		RegisteredAt: value.RegisteredAt.Format(time.RFC3339),
	}
	if value.RequestedOrganizationID != nil {
		requested := value.RequestedOrganizationID.String()
		result.RequestedOrganizationId = &requested
	}
	result.RequestedOrganizationName = value.RequestedOrganizationName
	return result
}
