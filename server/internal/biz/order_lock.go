package biz

import (
	"context"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
)

const (
	UnlockRouteRoleDirect       = "ROLE_DIRECT"
	UnlockRouteAdminEmergency   = "ADMIN_EMERGENCY"
	UnlockRouteDingTalkApproval = "DINGTALK_APPROVAL"

	UnlockStatusPendingDispatch      = "PENDING_DISPATCH"
	UnlockStatusPendingApproval      = "PENDING_APPROVAL"
	UnlockStatusApprovedPendingApply = "APPROVED_PENDING_APPLY"
	UnlockStatusApproved             = "APPROVED"
	UnlockStatusRejected             = "REJECTED"
	UnlockStatusConfigurationFailed  = "CONFIGURATION_FAILED"
	UnlockStatusDispatchFailed       = "DISPATCH_FAILED"
	UnlockStatusDispatchUnknown      = "DISPATCH_UNKNOWN"
	UnlockStatusStale                = "STALE"

	VersionSourceOrderLock = "ORDER_LOCK"
	VersionSourceAmendment = "AMENDMENT"
	VersionSourceSwitch    = "SWITCH"
	VersionSourceVoid      = "VOID"
)

var (
	ErrOrderAlreadyLocked                 = errors.Conflict("ORDER_ALREADY_LOCKED", "订单已被锁定")
	ErrOrderNotLocked                     = errors.Conflict("ORDER_NOT_LOCKED", "订单当前未锁定")
	ErrOrderLockRoleRequired              = errors.Forbidden("ORDER_LOCK_ROLE_REQUIRED", "当前用户未分配对应业务类型的订单锁定角色或数据范围不满足要求")
	ErrOrderUnlockRequestActive           = errors.Conflict("ORDER_UNLOCK_REQUEST_ACTIVE", "当前订单已有生效中或审批中的解锁请求")
	ErrOrderUnlockApproverNotConfigured   = errors.BadRequest("ORDER_UNLOCK_APPROVER_NOT_CONFIGURED", "未配置具备对应业务类型订单锁定权限的业务角色成员")
	ErrOrderUnlockDingTalkNotConfigured   = errors.BadRequest("ORDER_UNLOCK_DINGTALK_NOT_CONFIGURED", "申请人或审批候选人未绑定钉钉账号")
	ErrOrderUnlockDingTalkDispatchFailed  = errors.InternalServer("ORDER_UNLOCK_DINGTALK_DISPATCH_FAILED", "钉钉审批发起明确失败")
	ErrOrderUnlockDingTalkDispatchUnknown = errors.InternalServer("ORDER_UNLOCK_DINGTALK_DISPATCH_UNKNOWN", "钉钉审批发起结果未知")
	ErrOrderUnlockApprovalStale           = errors.Conflict("ORDER_UNLOCK_APPROVAL_STALE", "审批请求已过期或已被直接解锁取代")
	ErrOrderUnlockApproverInvalid         = errors.Forbidden("ORDER_UNLOCK_APPROVER_INVALID", "非有效审批人")
	ErrSeaDocumentVersionConflict         = errors.Conflict("SEA_DOCUMENT_VERSION_CONFLICT", "单证版本冲突，请刷新后重试")
	ErrOrderUnlockRequestNotFound         = errors.NotFound("ORDER_NOT_FOUND", "解锁请求记录不存在")
)

// NewErrOrderBusinessLocked 构建订单业务锁定错误，携带结构化元数据。
func NewErrOrderBusinessLocked(orderID uuid.UUID, orderNo string, lockGen uint64, lockedAt time.Time, lockedByName string) error {
	return errors.Conflict("ORDER_BUSINESS_LOCKED", "订单已锁定，如需修改请先申请解锁").WithMetadata(map[string]string{
		"order_id":        orderID.String(),
		"order_no":        orderNo,
		"lock_generation": fmt.Sprintf("%d", lockGen),
		"locked_at":       lockedAt.Format(time.RFC3339),
		"locked_by_name":  lockedByName,
	})
}

// NewErrSeaMasterBillMemberOrderLocked 构建共享 MBL 成员订单锁定阻断错误。
func NewErrSeaMasterBillMemberOrderLocked(count int, lockedOrderNos []string) error {
	return errors.Conflict("SEA_MASTER_BILL_MEMBER_ORDER_LOCKED", "共享 MBL 关联的部分订单已被锁定，需全部解锁后才能修改共享 MBL").WithMetadata(map[string]string{
		"locked_count":     fmt.Sprintf("%d", count),
		"locked_order_nos": strings.Join(lockedOrderNos, ","),
	})
}

// OrderLockState 订单锁状态与可用动作。
type OrderLockState struct {
	OrderID                 uuid.UUID
	OrderNo                 string
	BusinessType            OrderBusinessType
	IsLocked                bool
	LockGeneration          uint64
	LockedAt                *time.Time
	LockedBy                *uuid.UUID
	LockedByName            *string
	OrderVersion            uint64
	CanLock                 bool
	CanRoleDirectUnlock     bool
	CanAdminEmergencyUnlock bool
	CanRequestUnlock        bool
	LockBlockedReasons      []string
	UnlockBlockedReasons    []string
	ActiveUnlockRequest     *OrderUnlockRequest
	CurrentLockRecord       *OrderLockRecord
}

// OrderLockRecord 锁定周期事实记录。
type OrderLockRecord struct {
	ID                   uuid.UUID
	OrganizationID       uuid.UUID
	OrderID              uuid.UUID
	OrderNo              string
	BusinessType         OrderBusinessType
	Generation           uint64
	LockedBy             uuid.UUID
	LockedByName         string
	LockedAt             time.Time
	OrderVersionAtLock   uint64
	MasterBillID         *uuid.UUID
	MasterBillVersionID  *uuid.UUID
	UnlockedBy           *uuid.UUID
	UnlockedByName       *string
	UnlockedAt           *time.Time
	OrderVersionAtUnlock *uint64
	UnlockRequestID      *uuid.UUID
	UnlockReason         *string
	UnlockMode           *string
	HouseBillSnapshots   []*OrderLockHouseBillSnapshot
}

// OrderLockHouseBillSnapshot 锁定瞬时有效 HBL 快照。
type OrderLockHouseBillSnapshot struct {
	ID                 uuid.UUID
	OrganizationID     uuid.UUID
	LockRecordID       uuid.UUID
	HouseBillID        uuid.UUID
	HouseBillVersionID uuid.UUID
	HouseNoSnapshot    string
	CreatedAt          time.Time
}

// OrderUnlockRequest 解锁请求事实记录。
type OrderUnlockRequest struct {
	ID                        uuid.UUID
	OrganizationID            uuid.UUID
	OrderID                   uuid.UUID
	OrderNo                   string
	BusinessType              OrderBusinessType
	LockRecordID              uuid.UUID
	LockGeneration            uint64
	RequestedBy               uuid.UUID
	RequestedByName           string
	RequestedAt               time.Time
	Reason                    *string
	ExpectedOrderVersion      uint64
	IdempotencyKey            string
	RequestFingerprint        string
	Route                     string
	Status                    string
	DingTalkProcessInstanceID *string
	DingTalkProcessCode       *string
	DecidedBy                 *uuid.UUID
	DecidedByName             *string
	DecidedAt                 *time.Time
	DecisionSource            *string
	FailureCode               *string
	FailureMessage            *string
	SupersededByRequestID     *uuid.UUID
	UnlockedAt                *time.Time
	ResultOrderVersion        *uint64
	ApproverCandidates        []*OrderUnlockApproverCandidate
}

// OrderUnlockApproverCandidate 申请解锁时的审批候选人快照。
type OrderUnlockApproverCandidate struct {
	ID                     uuid.UUID
	RequestID              uuid.UUID
	UserID                 uuid.UUID
	MembershipID           uuid.UUID
	RoleID                 uuid.UUID
	DisplayNameSnapshot    string
	DingTalkUserIDSnapshot string
	CreatedAt              time.Time
}

// OrderLockResult 锁定操作返回结果。
type OrderLockResult struct {
	State      *OrderLockState
	LockRecord *OrderLockRecord
}

// OrderUnlockResult 解锁操作返回结果。
type OrderUnlockResult struct {
	State   *OrderLockState
	Request *OrderUnlockRequest
}

// OrderLockRepo 订单锁数据访问接口。
type OrderLockRepo interface {
	GetOrderLockState(ctx context.Context, organizationID, orderID uuid.UUID, caller *Principal) (*OrderLockState, error)
	LockOrder(ctx context.Context, caller *Principal, orderID uuid.UUID, expectedOrderVersion uint64, idempotencyKey string, audit *AuditEvent) (*OrderLockResult, error)
	RequestOrderUnlock(ctx context.Context, caller *Principal, orderID uuid.UUID, expectedOrderVersion uint64, idempotencyKey string, reason *string, audit *AuditEvent) (*OrderUnlockResult, error)
	ListOrderUnlockRequests(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int) ([]*OrderUnlockRequest, int, error)
	GetOrderUnlockRequest(ctx context.Context, organizationID, orderID, requestID uuid.UUID) (*OrderUnlockRequest, error)
}

// OrderLockUsecase 订单业务锁与不可变单证版本用例。
type OrderLockUsecase struct {
	repo OrderLockRepo
}

func NewOrderLockUsecase(repo OrderLockRepo) *OrderLockUsecase {
	return &OrderLockUsecase{repo: repo}
}

func (uc *OrderLockUsecase) GetOrderLockState(ctx context.Context, organizationID, orderID uuid.UUID, caller *Principal) (*OrderLockState, error) {
	return uc.repo.GetOrderLockState(ctx, organizationID, orderID, caller)
}

func (uc *OrderLockUsecase) LockOrder(ctx context.Context, caller *Principal, orderID uuid.UUID, expectedOrderVersion uint64, idempotencyKey string, audit *AuditEvent) (*OrderLockResult, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if caller == nil || caller.UserID == uuid.Nil || caller.Organization.ID == uuid.Nil || orderID == uuid.Nil || expectedOrderVersion == 0 ||
		idempotencyKey == "" || utf8.RuneCountInString(idempotencyKey) > 128 || containsControl(idempotencyKey) {
		return nil, ErrOrderInvalidArgument
	}
	return uc.repo.LockOrder(ctx, caller, orderID, expectedOrderVersion, idempotencyKey, audit)
}

func (uc *OrderLockUsecase) RequestOrderUnlock(ctx context.Context, caller *Principal, orderID uuid.UUID, expectedOrderVersion uint64, idempotencyKey string, reason *string, audit *AuditEvent) (*OrderUnlockResult, error) {
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	if reason != nil {
		normalizedReason := strings.TrimSpace(*reason)
		if utf8.RuneCountInString(normalizedReason) > 500 || containsControl(normalizedReason) {
			return nil, ErrOrderInvalidArgument
		}
		if normalizedReason == "" {
			reason = nil
		} else {
			reason = &normalizedReason
		}
	}
	if caller == nil || caller.UserID == uuid.Nil || caller.Organization.ID == uuid.Nil || orderID == uuid.Nil || expectedOrderVersion == 0 ||
		idempotencyKey == "" || utf8.RuneCountInString(idempotencyKey) > 128 || containsControl(idempotencyKey) {
		return nil, ErrOrderInvalidArgument
	}
	return uc.repo.RequestOrderUnlock(ctx, caller, orderID, expectedOrderVersion, idempotencyKey, reason, audit)
}

func (uc *OrderLockUsecase) ListOrderUnlockRequests(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int) ([]*OrderUnlockRequest, int, error) {
	if organizationID == uuid.Nil || orderID == uuid.Nil || !ValidListPagination(page, pageSize) {
		return nil, 0, ErrOrderInvalidArgument
	}
	return uc.repo.ListOrderUnlockRequests(ctx, organizationID, orderID, page, pageSize)
}

func (uc *OrderLockUsecase) GetOrderUnlockRequest(ctx context.Context, organizationID, orderID, requestID uuid.UUID) (*OrderUnlockRequest, error) {
	return uc.repo.GetOrderUnlockRequest(ctx, organizationID, orderID, requestID)
}
