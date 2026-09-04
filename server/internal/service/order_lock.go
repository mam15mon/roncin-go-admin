package service

import (
	"context"
	"time"

	"github.com/google/uuid"

	v1 "github.com/roncin/roncin-go-admin/server/api/order/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

// OrderLockService 提供订单业务锁定与单证不可变版本服务实现。
type OrderLockService struct {
	v1.UnimplementedOrderLockServiceServer
	usecase *biz.OrderLockUsecase
}

func NewOrderLockService(usecase *biz.OrderLockUsecase) *OrderLockService {
	return &OrderLockService{usecase: usecase}
}

var _ v1.OrderLockServiceServer = (*OrderLockService)(nil)

func (s *OrderLockService) GetOrderLockState(ctx context.Context, req *v1.GetOrderLockStateRequest) (*v1.GetOrderLockStateResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}

	state, err := s.usecase.GetOrderLockState(ctx, principal.Organization.ID, orderID, principal)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.GetOrderLockStateResponse{
		Data: orderLockStateToAPI(state),
	}), nil
}

func (s *OrderLockService) LockOrder(ctx context.Context, req *v1.LockOrderRequest) (*v1.LockOrderResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Action:         "order.lock",
		Result:         "success",
	}

	res, err := s.usecase.LockOrder(ctx, principal, orderID, req.GetExpectedOrderVersion(), req.GetIdempotencyKey(), audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.LockOrderResponse{
		Data: &v1.OrderLockResultData{
			State:      orderLockStateToAPI(res.State),
			LockRecord: orderLockRecordToAPI(res.LockRecord),
		},
	}), nil
}

func (s *OrderLockService) RequestOrderUnlock(ctx context.Context, req *v1.RequestOrderUnlockRequest) (*v1.RequestOrderUnlockResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}

	audit := &biz.AuditEvent{
		OrganizationID: &principal.Organization.ID,
		UserID:         &principal.UserID,
		Action:         "order.unlock.request",
		Result:         "success",
	}

	res, err := s.usecase.RequestOrderUnlock(ctx, principal, orderID, req.GetExpectedOrderVersion(), req.GetIdempotencyKey(), req.Reason, audit)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.RequestOrderUnlockResponse{
		Data: &v1.OrderUnlockResultData{
			State:   orderLockStateToAPI(res.State),
			Request: orderUnlockRequestToAPI(res.Request),
		},
	}), nil
}

func (s *OrderLockService) ListOrderUnlockRequests(ctx context.Context, req *v1.ListOrderUnlockRequestsRequest) (*v1.ListOrderUnlockRequestsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}

	page, pageSize, err := listPageValues(req.GetPage(), req.GetPageSize(), biz.ErrOrderInvalidArgument)
	if err != nil {
		return nil, err
	}

	items, total, err := s.usecase.ListOrderUnlockRequests(ctx, principal.Organization.ID, orderID, page, pageSize)
	if err != nil {
		return nil, err
	}

	apiItems := make([]*v1.OrderUnlockRequestData, len(items))
	for i, it := range items {
		apiItems[i] = orderUnlockRequestToAPI(it)
	}

	return okList(ctx, &v1.ListOrderUnlockRequestsResponse{
		Data: &v1.ListOrderUnlockRequestsData{
			Items:    apiItems,
			Total:    int32(total),
			Page:     int32(page),
			PageSize: int32(pageSize),
		},
	}), nil
}

func (s *OrderLockService) GetOrderUnlockRequest(ctx context.Context, req *v1.GetOrderUnlockRequestRequest) (*v1.GetOrderUnlockRequestResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	orderID, err := uuid.Parse(req.GetOrderId())
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}
	requestID, err := uuid.Parse(req.GetRequestId())
	if err != nil {
		return nil, biz.ErrOrderInvalidArgument
	}

	item, err := s.usecase.GetOrderUnlockRequest(ctx, principal.Organization.ID, orderID, requestID)
	if err != nil {
		return nil, err
	}

	return ok(ctx, &v1.GetOrderUnlockRequestResponse{
		Data: orderUnlockRequestToAPI(item),
	}), nil
}

func orderLockStateToAPI(s *biz.OrderLockState) *v1.OrderLockStateData {
	if s == nil {
		return nil
	}
	data := &v1.OrderLockStateData{
		OrderId:                 s.OrderID.String(),
		OrderNo:                 s.OrderNo,
		IsLocked:                s.IsLocked,
		LockGeneration:          s.LockGeneration,
		OrderVersion:            s.OrderVersion,
		CanLock:                 s.CanLock,
		CanRoleDirectUnlock:     s.CanRoleDirectUnlock,
		CanAdminEmergencyUnlock: s.CanAdminEmergencyUnlock,
		CanRequestUnlock:        s.CanRequestUnlock,
		LockBlockedReasons:      s.LockBlockedReasons,
		UnlockBlockedReasons:    s.UnlockBlockedReasons,
	}
	if s.LockedAt != nil {
		tStr := s.LockedAt.Format(time.RFC3339)
		data.LockedAt = &tStr
	}
	if s.LockedBy != nil {
		idStr := s.LockedBy.String()
		data.LockedBy = &idStr
	}
	if s.LockedByName != nil {
		data.LockedByName = s.LockedByName
	}
	if s.ActiveUnlockRequest != nil {
		data.ActiveUnlockRequest = orderUnlockRequestToAPI(s.ActiveUnlockRequest)
	}
	if s.CurrentLockRecord != nil {
		data.CurrentLockRecord = orderLockRecordToAPI(s.CurrentLockRecord)
	}
	return data
}

func orderLockRecordToAPI(r *biz.OrderLockRecord) *v1.OrderLockRecordData {
	if r == nil {
		return nil
	}
	data := &v1.OrderLockRecordData{
		Id:                  r.ID.String(),
		OrderId:             r.OrderID.String(),
		OrderNo:             r.OrderNo,
		Generation:          r.Generation,
		LockedBy:            r.LockedBy.String(),
		LockedByName:        r.LockedByName,
		LockedAt:            r.LockedAt.Format(time.RFC3339),
		OrderVersionAtLock:  r.OrderVersionAtLock,
		MasterBillId:        r.MasterBillID.String(),
		MasterBillVersionId: r.MasterBillVersionID.String(),
		UnlockReason:        r.UnlockReason,
		UnlockMode:          r.UnlockMode,
	}
	if r.UnlockedBy != nil {
		uStr := r.UnlockedBy.String()
		data.UnlockedBy = &uStr
	}
	if r.UnlockedByName != nil {
		data.UnlockedByName = r.UnlockedByName
	}
	if r.UnlockedAt != nil {
		tStr := r.UnlockedAt.Format(time.RFC3339)
		data.UnlockedAt = &tStr
	}
	if r.OrderVersionAtUnlock != nil {
		data.OrderVersionAtUnlock = r.OrderVersionAtUnlock
	}
	if r.UnlockRequestID != nil {
		reqIDStr := r.UnlockRequestID.String()
		data.UnlockRequestId = &reqIDStr
	}
	if len(r.HouseBillSnapshots) > 0 {
		snaps := make([]*v1.OrderLockHouseBillSnapshotData, len(r.HouseBillSnapshots))
		for i, snap := range r.HouseBillSnapshots {
			snaps[i] = &v1.OrderLockHouseBillSnapshotData{
				Id:                 snap.ID.String(),
				LockRecordId:       snap.LockRecordID.String(),
				HouseBillId:        snap.HouseBillID.String(),
				HouseBillVersionId: snap.HouseBillVersionID.String(),
				HouseNoSnapshot:    snap.HouseNoSnapshot,
				CreatedAt:          snap.CreatedAt.Format(time.RFC3339),
			}
		}
		data.HouseBillSnapshots = snaps
	}
	return data
}

func orderUnlockRequestToAPI(r *biz.OrderUnlockRequest) *v1.OrderUnlockRequestData {
	if r == nil {
		return nil
	}
	data := &v1.OrderUnlockRequestData{
		Id:                        r.ID.String(),
		OrderId:                   r.OrderID.String(),
		OrderNo:                   r.OrderNo,
		LockRecordId:              r.LockRecordID.String(),
		LockGeneration:            r.LockGeneration,
		RequestedBy:               r.RequestedBy.String(),
		RequestedByName:           r.RequestedByName,
		RequestedAt:               r.RequestedAt.Format(time.RFC3339),
		Reason:                    r.Reason,
		ExpectedOrderVersion:      r.ExpectedOrderVersion,
		IdempotencyKey:            r.IdempotencyKey,
		Route:                     r.Route,
		Status:                    r.Status,
		DingtalkProcessInstanceId: r.DingTalkProcessInstanceID,
		DingtalkProcessCode:       r.DingTalkProcessCode,
		DecidedByName:             r.DecidedByName,
		DecisionSource:            r.DecisionSource,
		FailureCode:               r.FailureCode,
		FailureMessage:            r.FailureMessage,
		ResultOrderVersion:        r.ResultOrderVersion,
	}
	if r.DecidedBy != nil {
		dStr := r.DecidedBy.String()
		data.DecidedBy = &dStr
	}
	if r.DecidedAt != nil {
		tStr := r.DecidedAt.Format(time.RFC3339)
		data.DecidedAt = &tStr
	}
	if r.SupersededByRequestID != nil {
		sStr := r.SupersededByRequestID.String()
		data.SupersededByRequestId = &sStr
	}
	if r.UnlockedAt != nil {
		tStr := r.UnlockedAt.Format(time.RFC3339)
		data.UnlockedAt = &tStr
	}
	if len(r.ApproverCandidates) > 0 {
		cands := make([]*v1.OrderUnlockApproverCandidateData, len(r.ApproverCandidates))
		for i, cand := range r.ApproverCandidates {
			cands[i] = &v1.OrderUnlockApproverCandidateData{
				Id:                     cand.ID.String(),
				RequestId:              cand.RequestID.String(),
				UserId:                 cand.UserID.String(),
				MembershipId:           cand.MembershipID.String(),
				RoleId:                 cand.RoleID.String(),
				DisplayNameSnapshot:    cand.DisplayNameSnapshot,
				DingtalkUseridSnapshot: cand.DingTalkUserIDSnapshot,
			}
		}
		data.ApproverCandidates = cands
	}
	return data
}
