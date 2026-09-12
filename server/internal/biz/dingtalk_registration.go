package biz

import (
	"context"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/roncin/roncin-go-admin/server/internal/access"

	"github.com/google/uuid"
)

// DingTalkRegistration 是待审批的钉钉扫码注册视图：钉钉姓名/头像来自扫码返回，
// 供管理员在审批队列人工认领。RequestedOrganizationID 为空表示存量兜底
// （落总部收口，通知总部管理员）。
type DingTalkRegistration struct {
	UserID                    uuid.UUID
	DisplayName               string
	AvatarURL                 *string
	RequestedOrganizationID   *uuid.UUID
	RequestedOrganizationName string
	// IntakeOrganizationID 是注册落库时的收口成员资格组织（注册时为总部根），由仓储填充。
	IntakeOrganizationID uuid.UUID
	RegisteredAt         time.Time
}

// RoutingOrganizationID 返回该注册的路由组织（审批通知与权限校验的目标）：
// 自选了目标组织用自选值，否则用收口成员资格所在组织（注册时为总部根）。
func (r *DingTalkRegistration) RoutingOrganizationID() uuid.UUID {
	if r == nil {
		return uuid.Nil
	}
	if r.RequestedOrganizationID != nil {
		return *r.RequestedOrganizationID
	}
	return r.IntakeOrganizationID
}

type DingTalkRegistrationListOptions struct {
	Page     int
	PageSize int
}

type DingTalkRegistrationList = PagedList[*DingTalkRegistration]

// DingTalkApproverRecipient 是注册审批通知的收件人快照。
type DingTalkApproverRecipient struct {
	UserID      uuid.UUID
	DisplayName string
}

// DingTalkRegistrationDecision 是审批用例提交给仓储的一站式决策命令；
// OrganizationIDs 为审批人按权限解析的可写组织范围，仓储在事务内复核
// 注册的路由组织确实属于该范围。
type DingTalkRegistrationDecision struct {
	ActorID         uuid.UUID
	UserID          uuid.UUID
	OrganizationIDs []uuid.UUID
	RoleIDs         []uuid.UUID
	Reason          string
	Notification    *NotificationIntent
	Audit           *AuditEvent
}

// DingTalkRegistrationRepo 覆盖邀请管理与注册审批的仓储契约。
type DingTalkRegistrationRepo interface {
	GetActorRolesPrivilegeProfiles(context.Context, uuid.UUID, uuid.UUID) ([]*AdminRoleProfile, error)
	GetRolesPrivilegeProfiles(context.Context, uuid.UUID, []uuid.UUID) ([]*AdminRoleProfile, error)
	CreateInvitation(context.Context, *DingTalkInvitation, *AuditEvent) (*DingTalkInvitation, error)
	ListInvitations(context.Context, []uuid.UUID, DingTalkInvitationListOptions) (*DingTalkInvitationList, error)
	RevokeInvitation(context.Context, uuid.UUID, uuid.UUID, []uuid.UUID, *AuditEvent) (*DingTalkInvitation, error)
	FindInvitationByToken(context.Context, string) (*DingTalkInvitation, error)
	GetPendingRegistration(context.Context, uuid.UUID) (*DingTalkRegistration, error)
	ListPendingRegistrations(context.Context, []uuid.UUID, DingTalkRegistrationListOptions) (*DingTalkRegistrationList, error)
	ApproveRegistration(context.Context, *DingTalkRegistrationDecision) (*DingTalkRegistration, error)
	RejectRegistration(context.Context, *DingTalkRegistrationDecision) error
	TransferRegistration(context.Context, *DingTalkRegistrationDecision, uuid.UUID) error
	ListApproverRecipients(context.Context, uuid.UUID) ([]*DingTalkApproverRecipient, error)
	ListApproverRecipientsWithEscalation(context.Context, uuid.UUID) ([]*DingTalkApproverRecipient, uuid.UUID, bool, error)
	GetParentOrganizationID(context.Context, uuid.UUID) (*uuid.UUID, bool, error)
}

type DingTalkRegistrationUsecase struct {
	repo DingTalkRegistrationRepo
	now  func() time.Time
}

func NewDingTalkRegistrationUsecase(repo DingTalkRegistrationRepo) *DingTalkRegistrationUsecase {
	return &DingTalkRegistrationUsecase{repo: repo, now: time.Now}
}

// writableOrganizationIDs 返回主体按「钉钉邀请与注册审批」权限可写的组织集合，
// 是邀请与审批所有组织范围判定的唯一口径（service 不得另建一套）。
func writableOrganizationIDs(principal *Principal) ([]uuid.UUID, error) {
	if principal == nil {
		return nil, ErrPermissionDenied
	}
	scope, err := principal.ResolvePermissionOrganizationScope(access.UserDingTalkInvitationManage)
	if err != nil {
		return nil, err
	}
	if len(scope.WritableOrganizationIDs) == 0 {
		return nil, ErrPermissionDenied
	}
	return scope.WritableOrganizationIDs, nil
}

// CreateInvitation 校验手机号/类型、有效期与提权边界后创建邀请：
// - TARGETED（定向）：手机号必填，扫码匹配后秒级激活；
// - GENERIC（通用）：手机号为空，支持多人多次扫码，扫码生成待审批注册。
// 目标组织必须在调用者的可写范围内，预设角色属于目标组织且不得超出调用者自身权限。
func (uc *DingTalkRegistrationUsecase) CreateInvitation(ctx context.Context, principal *Principal, kind DingTalkInvitationKind, mobile, displayName string, organizationID uuid.UUID, roleID *uuid.UUID, expiresInHours int) (*DingTalkInvitation, error) {
	if principal == nil || organizationID == uuid.Nil {
		return nil, ErrAdminInvalidArgument
	}
	if kind == "" {
		kind = DingTalkInvitationKindTargeted
	}
	if kind != DingTalkInvitationKindTargeted && kind != DingTalkInvitationKindGeneric {
		return nil, ErrAdminInvalidArgument
	}
	if !principal.CanAccessOrganizationForPermission(access.UserDingTalkInvitationManage, organizationID, true) {
		return nil, ErrPermissionDenied
	}

	var mobilePtr *string
	if kind == DingTalkInvitationKindTargeted {
		normalized := NormalizeDingTalkMobile(mobile)
		if !ValidDingTalkMobile(normalized) {
			return nil, ErrAdminInvalidArgument
		}
		mobilePtr = &normalized
	}

	trimmedName := strings.TrimSpace(displayName)
	if utf8.RuneCountInString(trimmedName) > 100 {
		return nil, ErrAdminInvalidArgument
	}
	if expiresInHours == 0 {
		expiresInHours = DingTalkInvitationDefaultTTLHours
	}
	if expiresInHours < DingTalkInvitationMinTTLHours || expiresInHours > DingTalkInvitationMaxTTLHours {
		return nil, ErrAdminInvalidArgument
	}
	if roleID != nil && *roleID != uuid.Nil {
		if err := uc.validateRolePrivilege(ctx, principal, organizationID, []uuid.UUID{*roleID}); err != nil {
			return nil, err
		}
	} else {
		roleID = nil
	}

	token, err := GenerateInvitationToken()
	if err != nil {
		return nil, err
	}

	auditDetail := ""
	if mobilePtr != nil {
		auditDetail = MaskDingTalkMobile(*mobilePtr)
	}
	audit := adminAuditEvent(ctx, principal.UserID, nil, "admin.dingtalk.invitation.create", auditDetail)
	audit.Details["target_organization.id"] = organizationID.String()
	audit.Details["kind"] = string(kind)

	input := &DingTalkInvitation{
		Token:          token,
		Kind:           kind,
		OrganizationID: organizationID,
		RoleID:         roleID,
		Mobile:         mobilePtr,
		DisplayName:    trimmedName,
		InvitedBy:      principal.UserID,
		Status:         DingTalkInvitationStatusPending,
		ExpiresAt:      uc.now().UTC().Add(time.Duration(expiresInHours) * time.Hour),
	}
	return uc.repo.CreateInvitation(ctx, input, audit)
}

func (uc *DingTalkRegistrationUsecase) ListInvitations(ctx context.Context, principal *Principal, options DingTalkInvitationListOptions) (*DingTalkInvitationList, error) {
	organizationIDs, err := writableOrganizationIDs(principal)
	if err != nil {
		return nil, err
	}
	if !ValidListPagination(options.Page, options.PageSize) {
		return nil, ErrAdminInvalidArgument
	}
	if options.OrganizationID != uuid.Nil && !uuidInSlice(organizationIDs, options.OrganizationID) {
		return nil, ErrPermissionDenied
	}
	return uc.repo.ListInvitations(ctx, organizationIDs, options)
}

// RevokeInvitation 撤销未消费的活跃邀请；组织范围在仓储事务内复核。
func (uc *DingTalkRegistrationUsecase) RevokeInvitation(ctx context.Context, principal *Principal, id uuid.UUID) error {
	if principal == nil || id == uuid.Nil {
		return ErrAdminInvalidArgument
	}
	organizationIDs, err := writableOrganizationIDs(principal)
	if err != nil {
		return err
	}
	audit := adminAuditEvent(ctx, principal.UserID, &id, "admin.dingtalk.invitation.revoke", "")
	_, err = uc.repo.RevokeInvitation(ctx, principal.UserID, id, organizationIDs, audit)
	return err
}

func (uc *DingTalkRegistrationUsecase) ListPendingRegistrations(ctx context.Context, principal *Principal, options DingTalkRegistrationListOptions) (*DingTalkRegistrationList, error) {
	organizationIDs, err := writableOrganizationIDs(principal)
	if err != nil {
		return nil, err
	}
	if !ValidListPagination(options.Page, options.PageSize) {
		return nil, ErrAdminInvalidArgument
	}
	return uc.repo.ListPendingRegistrations(ctx, organizationIDs, options)
}

// ApproveRegistration 一站式同意：启用账号 + 目标组织成员资格 + 初始角色 +
// 通知本人。幂等由 credential 状态机兜底：非待审批状态返回冲突错误。
func (uc *DingTalkRegistrationUsecase) ApproveRegistration(ctx context.Context, principal *Principal, userID uuid.UUID, roleIDs []uuid.UUID) (*DingTalkRegistration, error) {
	if principal == nil || userID == uuid.Nil || len(roleIDs) == 0 {
		return nil, ErrAdminInvalidArgument
	}
	organizationIDs, err := writableOrganizationIDs(principal)
	if err != nil {
		return nil, err
	}
	registration, err := uc.repo.GetPendingRegistration(ctx, userID)
	if err != nil {
		return nil, err
	}
	routingOrganizationID := registration.RoutingOrganizationID()
	if !uuidInSlice(organizationIDs, routingOrganizationID) {
		return nil, ErrPermissionDenied
	}
	if err := uc.validateRolePrivilege(ctx, principal, routingOrganizationID, roleIDs); err != nil {
		return nil, err
	}
	audit := adminAuditEvent(ctx, principal.UserID, &userID, "admin.dingtalk.registration.approve", "")
	audit.Details["target_organization.id"] = routingOrganizationID.String()
	return uc.repo.ApproveRegistration(ctx, &DingTalkRegistrationDecision{
		ActorID:         principal.UserID,
		UserID:          userID,
		OrganizationIDs: organizationIDs,
		RoleIDs:         roleIDs,
		Notification:    NewDingTalkUserAuthorizedNotification(userID),
		Audit:           audit,
	})
}

// RejectRegistration 拒绝注册：停用收口成员资格、通知本人并留痕；幂等语义同审批。
func (uc *DingTalkRegistrationUsecase) RejectRegistration(ctx context.Context, principal *Principal, userID uuid.UUID, reason string) error {
	if principal == nil || userID == uuid.Nil {
		return ErrAdminInvalidArgument
	}
	reason = strings.TrimSpace(reason)
	if reason == "" || utf8.RuneCountInString(reason) > 200 {
		return ErrDingTalkRegistrationReasonMissing
	}
	organizationIDs, err := writableOrganizationIDs(principal)
	if err != nil {
		return err
	}
	registration, err := uc.repo.GetPendingRegistration(ctx, userID)
	if err != nil {
		return err
	}
	if !uuidInSlice(organizationIDs, registration.RoutingOrganizationID()) {
		return ErrPermissionDenied
	}
	audit := adminAuditEvent(ctx, principal.UserID, &userID, "admin.dingtalk.registration.reject", reason)
	return uc.repo.RejectRegistration(ctx, &DingTalkRegistrationDecision{
		ActorID:         principal.UserID,
		UserID:          userID,
		OrganizationIDs: organizationIDs,
		Reason:          reason,
		Notification:    NewDingTalkRegistrationRejectedNotification(userID),
		Audit:           audit,
	})
}

// TransferRegistration 将待审批注册一键转派至目标分公司：
// 1. 调用者对当前路由组织持邀请管理权限；
// 2. 目标分公司不能等于当前路由组织；
// 3. 行级悲观锁原子更新 dingtalk_requested_organization_id 并向新组织审批人重新入队通知。
func (uc *DingTalkRegistrationUsecase) TransferRegistration(ctx context.Context, principal *Principal, userID, targetOrgID uuid.UUID, reason string) error {
	if principal == nil || userID == uuid.Nil || targetOrgID == uuid.Nil {
		return ErrAdminInvalidArgument
	}
	organizationIDs, err := writableOrganizationIDs(principal)
	if err != nil {
		return err
	}
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return ErrDingTalkRegistrationReasonMissing
	}
	if utf8.RuneCountInString(reason) > 200 {
		return ErrAdminInvalidArgument
	}
	registration, err := uc.repo.GetPendingRegistration(ctx, userID)
	if err != nil {
		return err
	}
	if !uuidInSlice(organizationIDs, registration.RoutingOrganizationID()) {
		return ErrPermissionDenied
	}
	if registration.RoutingOrganizationID() == targetOrgID {
		return ErrDingTalkRegistrationTransferSame
	}
	audit := adminAuditEvent(ctx, principal.UserID, &userID, "admin.dingtalk.registration.transfer", reason)
	audit.Details["user_id"] = userID.String()
	audit.Details["target_organization.id"] = targetOrgID.String()
	audit.Details["reason"] = reason

	return uc.repo.TransferRegistration(ctx, &DingTalkRegistrationDecision{
		ActorID:         principal.UserID,
		UserID:          userID,
		OrganizationIDs: organizationIDs,
		Reason:          reason,
		Audit:           audit,
	}, targetOrgID)
}

func (uc *DingTalkRegistrationUsecase) ListApproverRecipientsWithEscalation(ctx context.Context, targetOrgID uuid.UUID) ([]*DingTalkApproverRecipient, uuid.UUID, bool, error) {
	if targetOrgID == uuid.Nil {
		return nil, uuid.Nil, false, ErrAdminInvalidArgument
	}
	return uc.repo.ListApproverRecipientsWithEscalation(ctx, targetOrgID)
}

// validateRolePrivilege 邀请与审批授予的初始角色都不得超出调用者自身权限，
// 校验口径与用户管理（AdminUsecase.validateRolesPrivilege）一致。
func (uc *DingTalkRegistrationUsecase) validateRolePrivilege(ctx context.Context, principal *Principal, targetOrganizationID uuid.UUID, roleIDs []uuid.UUID) error {
	if principal == nil || principal.Organization.ID == uuid.Nil || principal.UserID == uuid.Nil || targetOrganizationID == uuid.Nil {
		return ErrAdminPrivilegeEscalation
	}
	if len(roleIDs) == 0 {
		return nil
	}
	profile, err := uc.actorPrivilegeProfile(ctx, principal.Organization.ID, principal.UserID)
	if err != nil {
		return err
	}
	roleProfiles, err := uc.repo.GetRolesPrivilegeProfiles(ctx, targetOrganizationID, roleIDs)
	if err != nil {
		return err
	}
	for _, roleProfile := range roleProfiles {
		if err := checkPrivilegeEscalation(profile, roleProfile.DataScope, roleProfile.PermissionKeys, roleProfile.OrganizationAccesses, roleProfile.Code == "administrator"); err != nil {
			return err
		}
	}
	return nil
}

func (uc *DingTalkRegistrationUsecase) actorPrivilegeProfile(ctx context.Context, organizationID, actorID uuid.UUID) (*AdminPrivilegeProfile, error) {
	roles, err := uc.repo.GetActorRolesPrivilegeProfiles(ctx, organizationID, actorID)
	if err != nil {
		return nil, err
	}
	profile := &AdminPrivilegeProfile{RoleProfiles: make([]AdminRoleProfile, 0, len(roles))}
	for _, role := range roles {
		if role == nil {
			continue
		}
		if role.Code == "administrator" {
			profile.IsSuperAdmin = true
		}
		profile.RoleProfiles = append(profile.RoleProfiles, *role)
	}
	return profile, nil
}

func uuidInSlice(values []uuid.UUID, target uuid.UUID) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
