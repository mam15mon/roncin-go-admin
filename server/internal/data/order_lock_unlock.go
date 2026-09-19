package data

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	dingtalkapprovaldispatchent "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkapprovaldispatch"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	orderunlockrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockrequest"
)

func (r *orderLockRepo) RequestOrderUnlock(ctx context.Context, caller *biz.Principal, orderID uuid.UUID, expectedOrderVersion uint64, idempotencyKey string, reason *string, audit *biz.AuditEvent) (*biz.OrderUnlockResult, error) {
	if caller == nil {
		return nil, biz.ErrOrderInvalidArgument
	}
	organizationID := caller.Organization.ID
	fingerprint := computeRequestFingerprint(organizationID, orderID, expectedOrderVersion, caller.UserID, reason)

	var resultRequest *ent.OrderUnlockRequest
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 1. 锁序 1: Order FOR UPDATE
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		businessType, parseErr := orderAccessBusinessType(order.BusinessType)
		if parseErr != nil {
			return parseErr
		}
		if order.LockedAt == nil && order.LockGeneration == 0 {
			return biz.ErrOrderNotLocked
		}

		// 2. 锁定当代 OrderLockRecord。
		lockRec, err := tx.OrderLockRecord.Query().
			Where(
				orderlockrecordent.OrderIDEQ(order.ID),
				orderlockrecordent.GenerationEQ(order.LockGeneration),
			).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return err
		}
		if access.OrderBusinessType(lockRec.BusinessType) != businessType {
			return biz.ErrOrderStatusConflict
		}

		// 3. 查询当前当代已有活动请求。
		activeReq, err := tx.OrderUnlockRequest.Query().
			Where(
				orderunlockrequestent.OrderIDEQ(order.ID),
				orderunlockrequestent.LockGenerationEQ(order.LockGeneration),
				orderunlockrequestent.StatusIn(
					biz.UnlockStatusPendingDispatch,
					biz.UnlockStatusPendingApproval,
					biz.UnlockStatusApprovedPendingApply,
					biz.UnlockStatusDispatchUnknown,
				),
			).
			ForUpdate().
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}
		if activeReq != nil && access.OrderBusinessType(activeReq.BusinessType) != businessType {
			return biz.ErrOrderStatusConflict
		}

		// 幂等检查位于锁状态和预期版本终态校验之前；取得固定事实锁后重查可覆盖并发直解。
		existingReq, requestErr := findOrderUnlockRequestByIdempotencyKey(ctx, tx.Client(), organizationID, idempotencyKey)
		if requestErr != nil {
			return requestErr
		}
		if existingReq != nil {
			if unlockRequestMatchesRequest(existingReq, orderID, caller.UserID, fingerprint) && access.OrderBusinessType(existingReq.BusinessType) == businessType {
				resultRequest = existingReq
				return nil
			}
			return biz.ErrOrderStatusConflict
		}
		if order.LockedAt == nil || lockRec.UnlockedAt != nil {
			return biz.ErrOrderNotLocked
		}
		if order.Version != expectedOrderVersion {
			return biz.ErrOrderStatusConflict
		}

		now := time.Now().UTC()

		// 5. 分流分支
		// 分支 A: Bootstrap Admin 紧急直接解锁
		if caller.IsBootstrapAdmin {
			var newReqID = uuid.Must(uuid.NewV7())
			newOrderVersion := order.Version + 1
			if _, err := tx.Order.UpdateOne(order).
				ClearLockedAt().
				ClearLockedBy().
				ClearLockSource().
				ClearAutoLockTriggerType().
				ClearAutoLockTriggerResourceID().
				ClearAutoLockTriggeredBy().
				SetVersion(newOrderVersion).
				Save(ctx); err != nil {
				return err
			}

			// 创建 APPROVED 的请求记录
			reqCreate := tx.OrderUnlockRequest.Create().
				SetID(newReqID).
				SetOrganizationID(organizationID).
				SetOrderID(order.ID).
				SetOrderNo(order.OrderNo).
				SetBusinessType(orderunlockrequestent.BusinessType(businessType)).
				SetLockRecordID(lockRec.ID).
				SetLockGeneration(order.LockGeneration).
				SetRequestedBy(caller.UserID).
				SetRequestedAt(now).
				SetNillableReason(reason).
				SetExpectedOrderVersion(expectedOrderVersion).
				SetIdempotencyKey(idempotencyKey).
				SetRequestFingerprint(fingerprint).
				SetRoute(orderunlockrequestent.RouteADMIN_EMERGENCY).
				SetStatus(orderunlockrequestent.StatusAPPROVED).
				SetDecidedBy(caller.UserID).
				SetDecidedAt(now).
				SetDecisionSource("BOOTSTRAP_ADMIN").
				SetUnlockedAt(now).
				SetResultOrderVersion(newOrderVersion)

			savedReq, err := reqCreate.Save(ctx)
			if err != nil {
				return err
			}
			resultRequest = savedReq
			// superseded_by_request_id 是即时外键，必须先保存取代它的新请求。
			if activeReq != nil {
				if _, err := tx.OrderUnlockRequest.UpdateOne(activeReq).
					SetStatus(orderunlockrequestent.StatusSTALE).
					SetSupersededByRequestID(savedReq.ID).
					Save(ctx); err != nil {
					return err
				}
			}

			// 关闭 LockRecord
			recordUpdate := tx.OrderLockRecord.UpdateOne(lockRec).
				SetUnlockedBy(caller.UserID).
				SetUnlockedAt(now).
				SetOrderVersionAtUnlock(newOrderVersion).
				SetUnlockRequestID(savedReq.ID).
				SetUnlockMode(orderlockrecordent.UnlockModeADMIN_EMERGENCY).
				SetNillableUnlockReason(reason)
			if _, err := recordUpdate.Save(ctx); err != nil {
				return err
			}

			// 审计
			if audit != nil {
				if audit.Action == "" {
					audit.Action = "order.unlock.request"
				}
				auditDetails := map[string]string{
					"route":                biz.UnlockRouteAdminEmergency,
					"business_type":        string(businessType),
					"admin_emergency":      "true",
					"order_id":             order.ID.String(),
					"order_no":             order.OrderNo,
					"lock_generation":      fmt.Sprintf("%d", order.LockGeneration),
					"result_order_version": fmt.Sprintf("%d", newOrderVersion),
					"requested_by":         caller.UserID.String(),
					"decided_by":           caller.UserID.String(),
				}
				if reason != nil {
					auditDetails["reason"] = *reason
				}
				audit.Details = auditDetails
				if err := safeWriteAudit(ctx, tx.AuditLog, audit, organizationID, caller.UserID); err != nil {
					return err
				}
			}
			return nil
		}

		// 分支 B: 业务锁定/解锁角色成员直接解锁
		qualifiedRoleMember, err := isUserQualifiedBusinessLockRole(ctx, tx.Client(), organizationID, caller.UserID, businessType)
		if err != nil {
			return err
		}
		if qualifiedRoleMember {
			var newReqID = uuid.Must(uuid.NewV7())
			newOrderVersion := order.Version + 1
			if _, err := tx.Order.UpdateOne(order).
				ClearLockedAt().
				ClearLockedBy().
				ClearLockSource().
				ClearAutoLockTriggerType().
				ClearAutoLockTriggerResourceID().
				ClearAutoLockTriggeredBy().
				SetVersion(newOrderVersion).
				Save(ctx); err != nil {
				return err
			}

			reqCreate := tx.OrderUnlockRequest.Create().
				SetID(newReqID).
				SetOrganizationID(organizationID).
				SetOrderID(order.ID).
				SetOrderNo(order.OrderNo).
				SetBusinessType(orderunlockrequestent.BusinessType(businessType)).
				SetLockRecordID(lockRec.ID).
				SetLockGeneration(order.LockGeneration).
				SetRequestedBy(caller.UserID).
				SetRequestedAt(now).
				SetNillableReason(reason).
				SetExpectedOrderVersion(expectedOrderVersion).
				SetIdempotencyKey(idempotencyKey).
				SetRequestFingerprint(fingerprint).
				SetRoute(orderunlockrequestent.RouteROLE_DIRECT).
				SetStatus(orderunlockrequestent.StatusAPPROVED).
				SetDecidedBy(caller.UserID).
				SetDecidedAt(now).
				SetDecisionSource("ROLE_DIRECT").
				SetUnlockedAt(now).
				SetResultOrderVersion(newOrderVersion)

			savedReq, err := reqCreate.Save(ctx)
			if err != nil {
				return err
			}
			resultRequest = savedReq
			// superseded_by_request_id 是即时外键，必须先保存取代它的新请求，再更新旧申请；
			// 否则角色直解与审批本地生效竞争时会稳定触发外键约束并回滚直解。
			if activeReq != nil {
				if _, err := tx.OrderUnlockRequest.UpdateOne(activeReq).
					SetStatus(orderunlockrequestent.StatusSTALE).
					SetSupersededByRequestID(savedReq.ID).
					Save(ctx); err != nil {
					return err
				}
			}

			recordUpdate := tx.OrderLockRecord.UpdateOne(lockRec).
				SetUnlockedBy(caller.UserID).
				SetUnlockedAt(now).
				SetOrderVersionAtUnlock(newOrderVersion).
				SetUnlockRequestID(savedReq.ID).
				SetUnlockMode(orderlockrecordent.UnlockModeROLE_DIRECT).
				SetNillableUnlockReason(reason)
			if _, err := recordUpdate.Save(ctx); err != nil {
				return err
			}

			if audit != nil {
				if audit.Action == "" {
					audit.Action = "order.unlock.request"
				}
				auditDetails := map[string]string{
					"route":                biz.UnlockRouteRoleDirect,
					"business_type":        string(businessType),
					"order_id":             order.ID.String(),
					"order_no":             order.OrderNo,
					"lock_generation":      fmt.Sprintf("%d", order.LockGeneration),
					"result_order_version": fmt.Sprintf("%d", newOrderVersion),
					"requested_by":         caller.UserID.String(),
					"decided_by":           caller.UserID.String(),
				}
				if reason != nil {
					auditDetails["reason"] = *reason
				}
				audit.Details = auditDetails
				if err := safeWriteAudit(ctx, tx.AuditLog, audit, organizationID, caller.UserID); err != nil {
					return err
				}
			}
			return nil
		}

		// 分支 C: 普通订单编辑人发起钉钉审批
		updatePermission := access.OrderPermission(businessType, access.OrderUpdate)
		if !caller.HasPermissionInScope(updatePermission, biz.DataScopeOrganization) || !caller.CanAccessOrganizationForPermission(updatePermission, order.OrganizationID, true) {
			return biz.ErrOrderLockRoleRequired
		}

		// 同一锁定代次若已有活动审批，普通编辑人重提返回既有活动请求
		if activeReq != nil {
			resultRequest = activeReq
			return nil
		}

		// 解析全部有效业务角色成员和钉钉绑定
		candidates, err := queryQualifiedBusinessLockCandidates(ctx, tx.Client(), organizationID, businessType)
		if err != nil {
			return err
		}

		// 检查申请人钉钉 UserID
		callerUser, err := tx.User.Get(ctx, caller.UserID)
		if err != nil {
			return err
		}
		callerDtID := ""
		if callerUser.DingtalkUserid != nil {
			callerDtID = strings.TrimSpace(*callerUser.DingtalkUserid)
		}
		approvalProcessCode := ""
		approvalCorpID := ""
		approvalEventToken := ""
		approvalEventAESKey := ""
		approvalEnabled := false
		if r.security != nil && r.security.Dingtalk != nil {
			approvalEnabled = r.security.Dingtalk.Enabled
			approvalProcessCode = strings.TrimSpace(r.security.Dingtalk.ApprovalProcessCode)
			approvalCorpID = strings.TrimSpace(r.security.Dingtalk.CorpId)
			approvalEventToken = strings.TrimSpace(r.security.Dingtalk.EventToken)
			approvalEventAESKey = strings.TrimSpace(r.security.Dingtalk.EventAesKey)
		}

		reqStatus := orderunlockrequestent.StatusPENDING_DISPATCH
		var failureCode *string
		var failureMessage *string

		if len(candidates) == 0 {
			reqStatus = orderunlockrequestent.StatusCONFIGURATION_FAILED
			fCode := "ORDER_UNLOCK_APPROVER_NOT_CONFIGURED"
			fMsg := "未配置具备对应业务类型订单锁定权限的业务角色成员"
			failureCode = &fCode
			failureMessage = &fMsg
		} else if callerDtID == "" {
			reqStatus = orderunlockrequestent.StatusCONFIGURATION_FAILED
			fCode := "ORDER_UNLOCK_DINGTALK_NOT_CONFIGURED"
			fMsg := "申请人未绑定钉钉账号"
			failureCode = &fCode
			failureMessage = &fMsg
		} else if !approvalEnabled || approvalProcessCode == "" || approvalCorpID == "" || approvalEventToken == "" || approvalEventAESKey == "" {
			reqStatus = orderunlockrequestent.StatusCONFIGURATION_FAILED
			fCode := "ORDER_UNLOCK_DINGTALK_NOT_CONFIGURED"
			fMsg := "未启用钉钉审批，或审批模板与事件回调配置不完整"
			failureCode = &fCode
			failureMessage = &fMsg
		} else {
			for _, c := range candidates {
				if c.DingTalkUserIDSnapshot == "" {
					reqStatus = orderunlockrequestent.StatusCONFIGURATION_FAILED
					fCode := "ORDER_UNLOCK_DINGTALK_NOT_CONFIGURED"
					fMsg := fmt.Sprintf("审批候选人 %s 未绑定钉钉账号", c.DisplayName)
					failureCode = &fCode
					failureMessage = &fMsg
					break
				}
			}
		}

		reqCreate := tx.OrderUnlockRequest.Create().
			SetOrganizationID(organizationID).
			SetOrderID(order.ID).
			SetOrderNo(order.OrderNo).
			SetBusinessType(orderunlockrequestent.BusinessType(businessType)).
			SetLockRecordID(lockRec.ID).
			SetLockGeneration(order.LockGeneration).
			SetRequestedBy(caller.UserID).
			SetRequestedAt(now).
			SetNillableReason(reason).
			SetExpectedOrderVersion(expectedOrderVersion).
			SetIdempotencyKey(idempotencyKey).
			SetRequestFingerprint(fingerprint).
			SetRoute(orderunlockrequestent.RouteDINGTALK_APPROVAL).
			SetStatus(reqStatus).
			SetNillableFailureCode(failureCode).
			SetNillableFailureMessage(failureMessage)

		savedReq, err := reqCreate.Save(ctx)
		if err != nil {
			return err
		}
		resultRequest = savedReq

		// 保存候选人快照
		for _, c := range candidates {
			if _, err := tx.OrderUnlockApproverCandidate.Create().
				SetRequestID(savedReq.ID).
				SetUserID(c.UserID).
				SetMembershipID(c.MembershipID).
				SetRoleID(c.RoleID).
				SetDisplayNameSnapshot(c.DisplayName).
				SetDingtalkUseridSnapshot(c.DingTalkUserIDSnapshot).
				Save(ctx); err != nil {
				return err
			}
		}

		// 若配置正常，创建后台派发任务与 Outbox
		if reqStatus == orderunlockrequestent.StatusPENDING_DISPATCH {
			candidateDtIDs := make([]string, 0, len(candidates))
			for _, c := range candidates {
				candidateDtIDs = append(candidateDtIDs, c.DingTalkUserIDSnapshot)
			}

			bgTask, bgErr := tx.BackgroundTask.Create().
				SetOrganizationID(organizationID).
				SetKind(backgroundtaskent.KindDINGTALK_APPROVAL_CREATE).
				SetIdempotencyKey(fmt.Sprintf("dt-approval-%s", savedReq.ID.String())).
				SetStatus(backgroundtaskent.StatusPENDING).
				Save(ctx)
			if bgErr != nil {
				return bgErr
			}

			if _, err := tx.DingTalkApprovalDispatch.Create().
				SetOrganizationID(organizationID).
				SetBackgroundTaskID(bgTask.ID).
				SetUnlockRequestID(savedReq.ID).
				SetProcessCodeSnapshot(approvalProcessCode).
				SetApplicantDingtalkUserid(callerDtID).
				SetCandidateDingtalkUserids(candidateDtIDs).
				SetRequestPayloadHash(fingerprint).
				SetDispatchStatus(dingtalkapprovaldispatchent.DispatchStatusPENDING).
				Save(ctx); err != nil {
				return err
			}
		}

		if audit != nil {
			if audit.Action == "" {
				audit.Action = "order.unlock.request"
			}
			auditDetails := map[string]string{
				"route":            biz.UnlockRouteDingTalkApproval,
				"business_type":    string(businessType),
				"status":           string(reqStatus),
				"order_id":         order.ID.String(),
				"order_no":         order.OrderNo,
				"lock_generation":  fmt.Sprintf("%d", order.LockGeneration),
				"requested_by":     caller.UserID.String(),
				"candidates_count": fmt.Sprintf("%d", len(candidates)),
			}
			if reason != nil {
				auditDetails["reason"] = *reason
			}
			audit.Details = auditDetails
			if err := safeWriteAudit(ctx, tx.AuditLog, audit, organizationID, caller.UserID); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil && ent.IsConstraintError(err) {
		client, clientErr := r.data.client(ctx)
		if clientErr != nil {
			return nil, clientErr
		}
		existingReq, lookupErr := findOrderUnlockRequestByIdempotencyKey(ctx, client, organizationID, idempotencyKey)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if existingReq != nil {
			if !unlockRequestMatchesRequest(existingReq, orderID, caller.UserID, fingerprint) {
				return nil, biz.ErrOrderStatusConflict
			}
			resultRequest = existingReq
			err = nil
		}
	}
	if err != nil {
		return nil, err
	}

	state, err := r.GetOrderLockState(ctx, organizationID, orderID, caller)
	if err != nil {
		return nil, err
	}

	request, err := r.mapUnlockRequestByID(ctx, resultRequest.ID)
	if err != nil {
		return nil, err
	}
	return &biz.OrderUnlockResult{
		State:   state,
		Request: request,
	}, nil
}

func (r *orderLockRepo) ListOrderUnlockRequests(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int) ([]*biz.OrderUnlockRequest, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}

	query := client.OrderUnlockRequest.Query().
		Where(
			orderunlockrequestent.OrganizationIDEQ(organizationID),
			orderunlockrequestent.OrderIDEQ(orderID),
		)

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := query.
		Order(ent.Desc(orderunlockrequestent.FieldCreatedAt), ent.Desc(orderunlockrequestent.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		WithApproverCandidates().
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*biz.OrderUnlockRequest, 0, len(items))
	for _, item := range items {
		result = append(result, r.mapUnlockRequest(ctx, client, item))
	}
	return result, total, nil
}

func (r *orderLockRepo) GetOrderUnlockRequest(ctx context.Context, organizationID, orderID, requestID uuid.UUID) (*biz.OrderUnlockRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	req, err := client.OrderUnlockRequest.Query().
		Where(
			orderunlockrequestent.IDEQ(requestID),
			orderunlockrequestent.OrganizationIDEQ(organizationID),
			orderunlockrequestent.OrderIDEQ(orderID),
		).
		WithApproverCandidates().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderUnlockRequestNotFound
		}
		return nil, err
	}
	return r.mapUnlockRequest(ctx, client, req), nil
}
