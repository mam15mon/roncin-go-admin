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
// CreateDingTalkInvitation 预建扫码邀请：
// - TARGETED（定向）：手机号必填，扫码匹配后秒级激活；
// - GENERIC（通用）：手机号为空，支持多人多次扫码，扫码生成待审批注册。
// 响应返回脱敏数据与邀请链接（/login?invite=<token>）。
func (s *AdminService) CreateDingTalkInvitation(ctx context.Context, request *v1.CreateDingTalkInvitationRequest) (*v1.CreateDingTalkInvitationResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	organizationID, err := uuid.Parse(strings.TrimSpace(request.GetOrganizationId()))
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	kind := biz.DingTalkInvitationKindTargeted
	if request.GetKind() == v1.DingTalkInvitationKind_DING_TALK_INVITATION_KIND_GENERIC {
		kind = biz.DingTalkInvitationKindGeneric
	}
	var roleID *uuid.UUID
	if request.RoleId != nil && strings.TrimSpace(request.GetRoleId()) != "" {
		parsedRoleID, parseErr := uuid.Parse(strings.TrimSpace(request.GetRoleId()))
		if parseErr != nil {
			return nil, biz.ErrAdminInvalidArgument
		}
		roleID = &parsedRoleID
	}
	displayName := ""
	if request.DisplayName != nil {
		displayName = request.GetDisplayName()
	}
	mobile := ""
	if request.Mobile != nil {
		mobile = request.GetMobile()
	}
	created, err := s.dingTalkRegistrations.CreateInvitation(ctx, principal, kind, mobile, displayName, organizationID, roleID, int(request.GetExpiresInHours()))
	if err != nil {
		return nil, err
	}
	invitationURL := "/login?invite=" + created.Token
	return ok(ctx, &v1.CreateDingTalkInvitationResponse{
		Data:          dingTalkInvitationToAPI(created, true),
		InvitationUrl: invitationURL,
	}), nil
}

func (s *AdminService) GetDingTalkInvitation(ctx context.Context, request *v1.GetDingTalkInvitationRequest) (*v1.GetDingTalkInvitationResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	invitationID, err := uuid.Parse(strings.TrimSpace(request.GetId()))
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	invitation, err := s.dingTalkRegistrations.GetInvitation(ctx, principal, invitationID)
	if err != nil {
		return nil, err
	}
	invitationURL := "/login?invite=" + invitation.Token
	return ok(ctx, &v1.GetDingTalkInvitationResponse{
		Data:          dingTalkInvitationToAPI(invitation, true),
		InvitationUrl: invitationURL,
	}), nil
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
		data = append(data, dingTalkInvitationToAPI(item, false))
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

func (s *AdminService) TransferDingTalkRegistration(ctx context.Context, request *v1.TransferDingTalkRegistrationRequest) (*v1.TransferDingTalkRegistrationResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	userID, err := uuid.Parse(strings.TrimSpace(request.GetUserId()))
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	targetOrgID, err := uuid.Parse(strings.TrimSpace(request.GetTargetOrganizationId()))
	if err != nil {
		return nil, biz.ErrAdminInvalidArgument
	}
	reason := strings.TrimSpace(request.GetReason())
	if reason == "" {
		return nil, biz.ErrDingTalkRegistrationReasonMissing
	}
	if err := s.dingTalkRegistrations.TransferRegistration(ctx, principal, userID, targetOrgID, reason); err != nil {
		return nil, err
	}
	return ok(ctx, &v1.TransferDingTalkRegistrationResponse{}), nil
}

func (s *AdminService) ListTransferOrganizations(ctx context.Context, _ *v1.ListTransferOrganizationsRequest) (*v1.ListTransferOrganizationsResponse, error) {
	principal, err := requirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	choices, err := s.dingTalkRegistrations.ListTransferOrganizations(ctx, principal)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.AdminOrganization, 0, len(choices))
	for _, choice := range choices {
		data = append(data, &v1.AdminOrganization{
			Id:   choice.OrganizationID.String(),
			Name: choice.OrganizationName,
			Code: choice.OrganizationCode,
		})
	}
	return ok(ctx, &v1.ListTransferOrganizationsResponse{Data: data}), nil
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
// 列表接口传 includeToken=false，避免明文批量泄漏 Token；创建与单条查询按需返回。
func dingTalkInvitationToAPI(value *biz.DingTalkInvitation, includeToken bool) *v1.DingTalkInvitation {
	if value == nil {
		return nil
	}
	kind := v1.DingTalkInvitationKind_DING_TALK_INVITATION_KIND_TARGETED
	if value.Kind == biz.DingTalkInvitationKindGeneric {
		kind = v1.DingTalkInvitationKind_DING_TALK_INVITATION_KIND_GENERIC
	}
	token := ""
	if includeToken {
		token = value.Token
	}
	result := &v1.DingTalkInvitation{
		Id:               value.ID.String(),
		Token:            token,
		Kind:             kind,
		OrganizationId:   value.OrganizationID.String(),
		OrganizationName: value.OrganizationName,
		Status:           dingTalkInvitationStatusToAPI(value.Status),
		CreatedAt:        value.CreatedAt.Format(time.RFC3339),
		ExpiresAt:        value.ExpiresAt.Format(time.RFC3339),
		InviterName:      value.InviterName,
	}
	if value.RoleID != nil {
		roleIDStr := value.RoleID.String()
		result.RoleId = &roleIDStr
		result.RoleName = &value.RoleName
	}
	if value.Mobile != nil {
		masked := biz.MaskDingTalkMobile(*value.Mobile)
		result.MobileMasked = &masked
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
