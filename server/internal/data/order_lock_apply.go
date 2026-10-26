package data

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
)

// orderLockCoreSpec 描述人工审核锁定及其不可变费用证据。
type orderLockCoreSpec struct {
	reviewerID            uuid.UUID
	idempotencyKey        string
	requestFingerprint    string
	feeSnapshot           json.RawMessage
	feeContentVersion     uint64
	feeContentFingerprint string
	reviewComment         string
}

// applyOrderLockCore 写入人工审核锁。调用方必须在当前事务内取得 Order 行锁，
// 完成授权、费用审核内容及幂等校验。
// 核心负责终止状态检查、SE 单证不可变版本快照、订单锁状态写入、锁定记录
// 与 HBL 快照创建；任一步失败由调用方的事务整体回滚。
func applyOrderLockCore(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, order *ent.Order, spec orderLockCoreSpec) (*ent.OrderLockRecord, *seaOrderLockSnapshot, error) {
	if spec.reviewerID == uuid.Nil || len(spec.feeSnapshot) == 0 || len(spec.feeContentFingerprint) != 64 {
		return nil, nil, biz.ErrFeeReviewInvalid
	}
	// 完结与人工审核独立，已结案订单仍可固化资料和费用审核证据。
	if order.TerminationStatus != orderent.TerminationStatusACTIVE {
		return nil, nil, biz.ErrOrderStatusConflict
	}
	if order.LockedAt != nil {
		return nil, nil, biz.ErrOrderAlreadyLocked
	}
	businessType, parseErr := orderAccessBusinessType(order.BusinessType)
	if parseErr != nil {
		return nil, nil, parseErr
	}

	var seaSnapshot *seaOrderLockSnapshot
	if businessType == access.OrderBusinessSE {
		var snapshotErr error
		seaSnapshot, snapshotErr = createSeaOrderLockSnapshot(ctx, tx, organizationID, order.ID, &spec.reviewerID)
		if snapshotErr != nil {
			return nil, nil, snapshotErr
		}
	}

	newLockGen := order.LockGeneration + 1
	now := time.Now().UTC()
	orderVersionAtLock := order.Version + 1

	orderUpdate := tx.Order.UpdateOne(order).
		SetLockedAt(now).
		SetLockGeneration(newLockGen).
		SetLockSource(orderent.LockSourceMANUAL).
		SetVersion(orderVersionAtLock)
	orderUpdate = orderUpdate.SetLockedBy(spec.reviewerID)

	if _, err := orderUpdate.Save(ctx); err != nil {
		return nil, nil, err
	}

	recordCreate := tx.OrderLockRecord.Create().
		SetOrganizationID(organizationID).
		SetOrderID(order.ID).
		SetOrderNo(order.OrderNo).
		SetBusinessType(orderlockrecordent.BusinessType(businessType)).
		SetGeneration(newLockGen).
		SetLockSource(orderlockrecordent.LockSourceMANUAL).
		SetLockedAt(now).
		SetOrderVersionAtLock(orderVersionAtLock).
		SetIdempotencyKey(spec.idempotencyKey).
		SetRequestFingerprint(spec.requestFingerprint).
		SetFeeSnapshot(spec.feeSnapshot).
		SetFeeContentVersion(spec.feeContentVersion).
		SetFeeContentFingerprint(spec.feeContentFingerprint).
		SetReviewComment(spec.reviewComment)
	recordCreate = recordCreate.SetLockedBy(spec.reviewerID)

	if seaSnapshot != nil {
		recordCreate = recordCreate.
			SetMasterBillID(seaSnapshot.MasterBillID).
			SetMasterBillVersionID(seaSnapshot.MasterBillVersionID).
			SetTransportExecutionID(seaSnapshot.TransportExecutionID).
			SetTransportExecutionVersionID(seaSnapshot.TransportExecutionVersionID)
	}
	if businessType == access.OrderBusinessAE {
		snapshot, e := airOrderLockDocumentSnapshot(ctx, tx, organizationID, order.ID)
		if e != nil {
			return nil, nil, e
		}
		recordCreate.SetAirDocumentSnapshot(snapshot)
	}
	rec, err := recordCreate.Save(ctx)
	if err != nil {
		return nil, nil, err
	}

	var houseBillSnapshots []seaOrderLockHouseBillSnapshot
	if seaSnapshot != nil {
		houseBillSnapshots = seaSnapshot.HouseBills
	}
	for _, snap := range houseBillSnapshots {
		if _, err := tx.OrderLockHouseBillSnapshot.Create().
			SetOrganizationID(organizationID).
			SetLockRecordID(rec.ID).
			SetHouseBillID(snap.HouseBillID).
			SetHouseBillVersionID(snap.HouseBillVersionID).
			SetHouseNoSnapshot(snap.HouseNoSnapshot).
			Save(ctx); err != nil {
			return nil, nil, err
		}
	}
	return rec, seaSnapshot, nil
}

func (r *orderLockRepo) LockOrder(ctx context.Context, caller *biz.Principal, orderID uuid.UUID, expectedOrderVersion uint64, idempotencyKey string, review biz.OrderLockReviewInput, audit *biz.AuditEvent) (*biz.OrderLockResult, error) {
	if caller == nil {
		return nil, biz.ErrOrderLockRoleRequired
	}
	organizationID := caller.Organization.ID
	fingerprint := computeLockOrderFingerprint(organizationID, orderID, expectedOrderVersion, caller.UserID, review)

	var resultRecord *ent.OrderLockRecord
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if err := lockFeeAuthorizationParents(ctx, tx.Client(), caller.UserID); err != nil {
			return err
		}
		// 1. 锁序 1: Order FOR UPDATE，并以数据库中的权威业务类型解析权限。
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
		// bootstrap admin 显式具备锁单资格；普通用户按统一 lock grant 口径判定。
		// 人工入口的授权判断保留在内部锁定核心之外。
		if !caller.IsBootstrapAdmin {
			qualified, qualificationErr := isUserQualifiedBusinessLockRole(ctx, tx.Client(), organizationID, caller.UserID, businessType)
			if qualificationErr != nil {
				return qualificationErr
			}
			if !qualified {
				return biz.ErrOrderLockRoleRequired
			}
		}

		// 取得 Order 锁后重查幂等记录，确保两个并发同键请求中等待者能看到先提交的事实，
		// 同时避免先锁幂等事实再锁 Order 形成反向锁序。
		existingRec, recErr := findOrderLockRecordByIdempotencyKey(ctx, tx.Client(), organizationID, idempotencyKey)
		if recErr != nil {
			return recErr
		}
		if existingRec != nil {
			if lockRecordMatchesRequest(existingRec, orderID, caller.UserID, fingerprint) {
				resultRecord = existingRec
				return nil
			}
			return biz.ErrOrderStatusConflict
		}
		// 先生命周期与锁状态，后预期版本；内部核心再次复核状态不变量。
		if order.TerminationStatus != orderent.TerminationStatusACTIVE {
			return biz.ErrOrderStatusConflict
		}
		if order.LockedAt != nil {
			return biz.ErrOrderAlreadyLocked
		}
		if order.Version != expectedOrderVersion {
			return biz.ErrOrderStatusConflict
		}

		// 订单行锁串行化所有费用写入；在同一事务核对审核人查看的内容。
		if order.FeeContentVersion != review.ExpectedFeeContentVersion {
			return biz.ErrFeeReviewConflict
		}
		pending, pendingErr := hasPendingFeeSupplement(ctx, tx.Client(), order.ID)
		if pendingErr != nil {
			return pendingErr
		}
		if pending {
			return biz.ErrFeeReviewPendingSupplement
		}
		snapshot, contentFingerprint, snapshotErr := loadFeeReviewSnapshot(ctx, tx.Client(), order.ID)
		if snapshotErr != nil {
			return snapshotErr
		}
		if contentFingerprint != review.FeeContentFingerprint {
			return biz.ErrFeeReviewConflict
		}
		if snapshot.Summary.Profit.IsNegative() && strings.TrimSpace(review.Comment) == "" {
			return biz.ErrFeeReviewInvalid
		}
		encodedSnapshot, encodeErr := json.Marshal(snapshot)
		if encodeErr != nil {
			return encodeErr
		}

		// 2. 原子固化人工审核费用证据、单证快照和唯一订单锁。
		rec, seaSnapshot, coreErr := applyOrderLockCore(ctx, tx, organizationID, order, orderLockCoreSpec{
			reviewerID:            caller.UserID,
			idempotencyKey:        idempotencyKey,
			requestFingerprint:    fingerprint,
			feeSnapshot:           encodedSnapshot,
			feeContentVersion:     order.FeeContentVersion,
			feeContentFingerprint: contentFingerprint,
			reviewComment:         review.Comment,
		})
		if coreErr != nil {
			return coreErr
		}
		resultRecord = rec

		// 3. 写入审计日志
		if audit != nil {
			if audit.Action == "" {
				audit.Action = "order.lock"
			}
			audit.Details = map[string]string{
				"business_type":           string(businessType),
				"order_id":                order.ID.String(),
				"order_no":                order.OrderNo,
				"lock_generation":         fmt.Sprintf("%d", rec.Generation),
				"order_version_at_lock":   fmt.Sprintf("%d", rec.OrderVersionAtLock),
				"fee_content_version":     fmt.Sprintf("%d", order.FeeContentVersion),
				"fee_content_fingerprint": contentFingerprint,
				"review_comment":          review.Comment,
			}
			if seaSnapshot != nil {
				audit.Details["master_bill_id"] = seaSnapshot.MasterBillID.String()
				audit.Details["master_bill_version_id"] = seaSnapshot.MasterBillVersionID.String()
				audit.Details["house_bills_count"] = fmt.Sprintf("%d", len(seaSnapshot.HouseBills))
			}
			if err := safeWriteAudit(ctx, tx.AuditLog, audit, organizationID, caller.UserID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil && ent.IsConstraintError(err) {
		// 不同 Order 可并发使用同一组织级幂等键；唯一索引竞争失败后在事务外读取胜者，
		// 同指纹返回原事实，异指纹映射为稳定 409，而不是泄漏驱动错误。
		client, clientErr := r.data.client(ctx)
		if clientErr != nil {
			return nil, clientErr
		}
		existingRec, lookupErr := findOrderLockRecordByIdempotencyKey(ctx, client, organizationID, idempotencyKey)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if existingRec != nil {
			if !lockRecordMatchesRequest(existingRec, orderID, caller.UserID, fingerprint) {
				return nil, biz.ErrOrderStatusConflict
			}
			resultRecord = existingRec
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

	lockRecord, err := r.mapLockRecordByID(ctx, resultRecord.ID)
	if err != nil {
		return nil, err
	}
	return &biz.OrderLockResult{
		State:      state,
		LockRecord: lockRecord,
	}, nil
}
