package data

import (
	"context"
	"sort"
	"strings"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeesupplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	orderservicetypeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderservicetype"
)

// loadFeeApplicability 读取订单的服务类型与异常案例适用范围，供费用项适用性
// 复核使用；与 orderFeeRepo.loadApplicability 同一口径，仅以显式客户端参数
// 支持事务内调用。
func loadFeeApplicability(ctx context.Context, client *ent.Client, organizationID, orderID uuid.UUID) (*orderFeeApplicability, error) {
	exists, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, biz.ErrOrderFeeNotFound
	}
	serviceTypes, err := client.OrderServiceType.Query().Where(orderservicetypeent.OrderIDEQ(orderID)).All(ctx)
	if err != nil {
		return nil, err
	}
	abnormalCases, err := client.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(orderID), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &orderFeeApplicability{serviceTypeIDs: make(map[uuid.UUID]struct{}, len(serviceTypes)), abnormalCaseIDs: make(map[uuid.UUID]struct{}, len(abnormalCases))}
	for _, serviceType := range serviceTypes {
		result.serviceTypeIDs[serviceType.MasterDataItemID] = struct{}{}
	}
	for _, abnormalCase := range abnormalCases {
		result.abnormalCaseIDs[abnormalCase.AbnormalCaseID] = struct{}{}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// 专用作废：能力投影与 design 5.1 六步事务
// ---------------------------------------------------------------------------

// supplementCancelBlockReason 是专用作废阻断的稳定投影。
type supplementCancelBlockReason struct {
	code   string
	reason string
}

// evaluateCancelCapability 按专用作废前置条件只读评估能力：仅限 APPROVED 申请、
// CONFIRMED、无活动账单行、无更晚生效且未作废的补录，且关联调整 confirmed_at/
// paid_at 从未写入、当前全为 DRAFT/CANCELLED。列表投影与专用作废命令复用本实现。
func evaluateCancelCapability(ctx context.Context, client *ent.Client, organizationID, orderID, requestID uuid.UUID) (*biz.OrderFeeSupplementCancelCapability, error) {
	result := &biz.OrderFeeSupplementCancelCapability{}
	request, err := client.OrderFeeSupplementRequest.Query().
		Where(orderfeesupplementent.IDEQ(requestID), orderfeesupplementent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFeeSupplementNotFound, nil)
	}
	if request.OrderID != orderID {
		return nil, biz.ErrFeeSupplementNotFound
	}
	if request.Status != orderfeesupplementent.StatusAPPROVED {
		return result, nil
	}
	fee, err := client.OrderFee.Query().Where(orderfeeent.SupplementRequestIDEQ(requestID)).Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return result, nil
		}
		return nil, err
	}
	result.FeeID = &fee.ID
	result.FeeStatus = string(fee.Status)
	blocked := func(code, reason string) (*biz.OrderFeeSupplementCancelCapability, error) {
		result.Cancellable = false
		result.BlockReasonCode = code
		result.BlockReason = reason
		return result, nil
	}
	if fee.Status == orderfeeent.StatusCANCELLED {
		return blocked("FEE_ALREADY_CANCELLED", "补录费用已作废")
	}
	activeLine, err := client.FinanceBillLine.Query().
		Where(financebilllineent.OrderFeeIDEQ(fee.ID), financebilllineent.ActiveEQ(true)).
		Exist(ctx)
	if err != nil {
		return nil, err
	}
	if activeLine || fee.Status == orderfeeent.StatusBILLED {
		return blocked("FEE_BILLED", "补录费用已建账，需先按现有财务链路取消账单")
	}
	if fee.Status != orderfeeent.StatusUNBILLED {
		return blocked("FEE_STATUS_INVALID", "仅 UNBILLED 状态的补录费用可专用作废")
	}
	// 更晚生效且未作废的补录存在时，必须按批准时间倒序逐笔作废，避免撤销早期
	// 成本后让后续边际冲减失真。比较键为批准时间 + 申请 ID。
	decidedAt := time.Time{}
	if request.DecidedAt != nil {
		decidedAt = *request.DecidedAt
	}
	laterEffective, err := client.OrderFee.Query().Where(
		orderfeeent.OrderIDEQ(orderID),
		orderfeeent.SupplementRequestIDNotNil(),
		orderfeeent.SupplementRequestIDNEQ(requestID),
		orderfeeent.StatusNEQ(orderfeeent.StatusCANCELLED),
		orderfeeent.HasSupplementRequestWith(
			orderfeesupplementent.StatusEQ(orderfeesupplementent.StatusAPPROVED),
			orderfeesupplementent.Or(
				orderfeesupplementent.DecidedAtGT(decidedAt),
				orderfeesupplementent.And(
					orderfeesupplementent.DecidedAtEQ(decidedAt),
					orderfeesupplementent.IDGT(requestID),
				),
			),
		),
	).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if laterEffective {
		return blocked("LATER_SUPPLEMENT_EXISTS", "存在更晚生效的有效补录，请按批准时间倒序逐笔作废")
	}
	adjustments, err := client.FinanceCommissionAdjustment.Query().
		Where(commissionadjustmentent.SourceFeeSupplementRequestIDEQ(requestID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, adjustment := range adjustments {
		if adjustment.ConfirmedAt != nil || adjustment.PaidAt != nil ||
			adjustment.Status == commissionadjustmentent.StatusCONFIRMED || adjustment.Status == commissionadjustmentent.StatusPAID {
			return blocked("ADJUSTMENT_CONFIRMED", "关联冲减建议已确认或已扣回，不允许直接作废补录费用")
		}
	}
	result.Cancellable = true
	return result, nil
}

func (r *orderFeeSupplementRepo) CancelCapability(ctx context.Context, organizationID, orderID, requestID uuid.UUID) (*biz.OrderFeeSupplementCancelCapability, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	return evaluateCancelCapability(ctx, client, organizationID, orderID, requestID)
}

// CancelApprovedFee 执行专用作废事务，固定锁序为 Order → 补录申请/费用 → 按
// UUID 排序的提成父单 → 关联调整（design 5.1 六步）。任何一步失败整体回滚；
// 该事务不删除任何事实，也不把系统作废伪装成财务主动忽略建议。
func (r *orderFeeSupplementRepo) CancelApprovedFee(ctx context.Context, input *biz.OrderFeeSupplementCancelInput) (*biz.OrderFeeSupplementCancelResult, error) {
	var result *biz.OrderFeeSupplementCancelResult
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		client := tx.Client()
		// 1. Order FOR UPDATE 后实时复核直接解锁资格；订单只作互斥点，不调用
		// 会禁止锁后专用动作的普通费用内容门禁。
		order, orderErr := tx.Order.Query().Where(orderent.IDEQ(input.OrderID), orderent.OrganizationIDEQ(input.OrganizationID)).ForUpdate().Only(ctx)
		if orderErr != nil {
			return mapEntError(orderErr, biz.ErrFeeSupplementNotFound, nil)
		}
		if order.ID != input.OrderID {
			return biz.ErrFeeSupplementNotFound
		}
		businessType, typeErr := orderAccessBusinessType(order.BusinessType)
		if typeErr != nil {
			return typeErr
		}
		qualified, grantErr := hasRealtimeLockGrantWithClient(ctx, client, input.OrganizationID, input.OperatorID, businessType, input.IsBootstrapAdmin)
		if grantErr != nil {
			return grantErr
		}
		if !qualified {
			return biz.ErrPermissionDenied
		}
		// 2. 锁定 APPROVED 申请和生成费用，比较费用版本，确认状态与来源匹配。
		request, requestErr := tx.OrderFeeSupplementRequest.Query().
			Where(orderfeesupplementent.IDEQ(input.RequestID), orderfeesupplementent.OrganizationIDEQ(input.OrganizationID)).
			ForUpdate().Only(ctx)
		if requestErr != nil {
			return mapEntError(requestErr, biz.ErrFeeSupplementNotFound, nil)
		}
		if request.OrderID != input.OrderID {
			return biz.ErrFeeSupplementNotFound
		}
		if request.Status != orderfeesupplementent.StatusAPPROVED {
			return biz.ErrFeeSupplementCancelBlocked
		}
		fee, feeErr := tx.OrderFee.Query().Where(orderfeeent.SupplementRequestIDEQ(input.RequestID)).ForUpdate().Only(ctx)
		if feeErr != nil {
			return mapEntError(feeErr, biz.ErrFeeSupplementCancelBlocked, nil)
		}
		if fee.OrderID != input.OrderID {
			return biz.ErrFeeSupplementCancelBlocked
		}
		if fee.Version != input.FeeExpectedVersion {
			return biz.ErrOrderFeeVersionConflict
		}
		blocked, capabilityErr := evaluateCancelCapability(ctx, client, input.OrganizationID, input.OrderID, input.RequestID)
		if capabilityErr != nil {
			return capabilityErr
		}
		if !blocked.Cancellable {
			return newFeeSupplementCancelBlockedError(blocked.BlockReason)
		}
		// 4. 按 design §5.1 固定锁序：先只读定位关联调整并按父单 UUID 升序锁定
		// 全部提成父单，再按 ID 锁定关联调整，与审批/确认路径保持同一顺序，
		// 避免多订单父单并发下形成反向锁序死锁。任一调整曾进入 CONFIRMED 或
		// PAID 时已在能力评估中整体拒绝。
		linkedAdjustments, linkedErr := tx.FinanceCommissionAdjustment.Query().
			Where(commissionadjustmentent.SourceFeeSupplementRequestIDEQ(input.RequestID)).
			All(ctx)
		if linkedErr != nil {
			return linkedErr
		}
		commissionIDs := make([]uuid.UUID, 0, len(linkedAdjustments))
		for _, adjustment := range linkedAdjustments {
			commissionIDs = append(commissionIDs, adjustment.CommissionID)
		}
		for _, commissionID := range uniqueSortedUUIDs(commissionIDs) {
			if _, parentErr := tx.FinanceCommission.Query().Where(commissionent.IDEQ(commissionID), commissionent.OrganizationIDEQ(input.OrganizationID)).ForUpdate().Only(ctx); parentErr != nil {
				return parentErr
			}
		}
		adjustments, adjustmentErr := tx.FinanceCommissionAdjustment.Query().
			Where(commissionadjustmentent.SourceFeeSupplementRequestIDEQ(input.RequestID)).
			Order(commissionadjustmentent.ByID()).ForUpdate().All(ctx)
		if adjustmentErr != nil {
			return adjustmentErr
		}
		// 5. 将关联 DRAFT 调整转为 CANCELLED，再将费用转为 CANCELLED；APPROVED
		// 申请保持终态不变，生成费用已作废由列表投影呈现。
		cancelledIDs := make([]uuid.UUID, 0, len(adjustments))
		now := time.Now().UTC()
		for _, adjustment := range adjustments {
			if adjustment.Status != commissionadjustmentent.StatusDRAFT {
				continue
			}
			if _, updateErr := tx.FinanceCommissionAdjustment.UpdateOne(adjustment).
				SetStatus(commissionadjustmentent.StatusCANCELLED).
				SetCancelledAt(now).
				SetCancelledBy(input.OperatorID).
				SetCancellationReason(input.Reason).
				SetVersion(adjustment.Version + 1).
				Save(ctx); updateErr != nil {
				return updateErr
			}
			cancelledIDs = append(cancelledIDs, adjustment.ID)
		}
		if _, feeUpdateErr := tx.OrderFee.UpdateOne(fee).
			SetStatus(orderfeeent.StatusCANCELLED).
			SetVersion(fee.Version + 1).
			SetCancelledAt(now).
			SetCancelledBy(input.OperatorID).
			SetCancellationReason(input.Reason).
			Save(ctx); feeUpdateErr != nil {
			return feeUpdateErr
		}
		// 6. 写作废人、时间、原因、费用前一版本与被取消调整 ID 审计后提交。
		if input.Audit != nil {
			if input.Audit.Details == nil {
				input.Audit.Details = make(map[string]string)
			}
			input.Audit.Details["fee.id"] = fee.ID.String()
			input.Audit.Details["fee.previous_version"] = decimal.NewFromInt(int64(fee.Version)).String()
			input.Audit.Details["fee.previous_status"] = string(fee.Status)
			cancelledIDTexts := make([]string, 0, len(cancelledIDs))
			for _, id := range cancelledIDs {
				cancelledIDTexts = append(cancelledIDTexts, id.String())
			}
			input.Audit.Details["cancelled_adjustment_ids"] = strings.Join(cancelledIDTexts, ",")
			if writeErr := writeAudit(ctx, tx.AuditLog, input.Audit); writeErr != nil {
				return writeErr
			}
		}
		loaded, loadErr := tx.OrderFee.Query().Where(orderfeeent.IDEQ(fee.ID)).WithSettlementParty().Only(ctx)
		if loadErr != nil {
			return loadErr
		}
		converted, convertErr := orderFeeToBiz(loaded)
		if convertErr != nil {
			return convertErr
		}
		result = &biz.OrderFeeSupplementCancelResult{Fee: converted, CancelledAdjustmentIDs: cancelledIDs}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// newFeeSupplementCancelBlockedError 以具体阻断原因构造稳定错误。
func newFeeSupplementCancelBlockedError(reason string) error {
	if reason == "" {
		return biz.ErrFeeSupplementCancelBlocked
	}
	return kratoserrors.Conflict("FEE_SUPPLEMENT_CANCEL_BLOCKED", reason)
}

func uniqueSortedUUIDs(values []uuid.UUID) []uuid.UUID {
	seen := make(map[uuid.UUID]struct{}, len(values))
	result := make([]uuid.UUID, 0, len(values))
	for _, value := range values {
		if _, exists := seen[value]; exists {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].String() < result[j].String() })
	return result
}
