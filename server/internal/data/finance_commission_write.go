package data

import (
	"context"
	"encoding/json"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commission "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	adjustment "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	fee "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
)

// Create 在事务内锁定来源并写入提成草稿与 CNY 快照，只负责写入并返回错误；
// 完整业务响应由用例在共享事务提交后通过普通上下文重读。方案与员工分配在
// 事务内按来源归属日期重新解析，不信任预览结果。
func (r *commissionRepo) Create(ctx context.Context, org uuid.UUID, c *biz.FinanceCommission, snapshot *biz.CommissionCNYSnapshot, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		calculation, err := calculateCommission(ctx, commissionStoreFromTx(tx), org, c.VerificationID, c.NettingID, c.EmployeeID, c.PersonnelRole, true)
		if err != nil {
			return err
		}
		// 原始 CNY 提成金额依赖锁内计算出的提成金额，按 biz 纯函数固化到快照。
		snapshot.ApplyCommissionAmount(calculation.CommissionAmount)
		// 活跃去重按来源路由：同来源同员工同角色仅允许一条非终态提成。
		duplicatePredicates := []predicate.FinanceCommission{
			commission.OrganizationIDEQ(org),
			commission.EmployeeIDEQ(c.EmployeeID),
			commission.PersonnelRoleEQ(string(calculation.PersonnelRole)),
			commission.StatusNEQ(commission.StatusCANCELLED),
		}
		if c.VerificationID != uuid.Nil {
			duplicatePredicates = append(duplicatePredicates, commission.VerificationIDEQ(c.VerificationID))
		} else {
			duplicatePredicates = append(duplicatePredicates, commission.NettingIDEQ(c.NettingID))
		}
		hasActive, err := tx.FinanceCommission.Query().Where(duplicatePredicates...).Exist(ctx)
		if err != nil {
			return err
		}
		if hasActive {
			return biz.ErrCommissionDuplicate
		}
		c.VerificationNo, c.NettingNo, c.EmployeeName, c.RuleName = calculation.VerificationNo, calculation.NettingNo, calculation.EmployeeName, calculation.RuleName
		c.PersonnelRole, c.CalculationBasis = calculation.PersonnelRole, calculation.CalculationBasis
		c.RuleID, c.RuleVersion, c.CalculationVersion, c.SourceFingerprint = calculation.RuleID, calculation.RuleVersion, calculation.CalculationVersion, calculation.SourceFingerprint
		c.BaseCurrency, c.RatePercent = calculation.BaseCurrency, calculation.RatePercent
		c.CustomerCount, c.OrderCount, c.FeeCount = calculation.CustomerCount, calculation.OrderCount, calculation.FeeCount
		c.RealizedRevenue, c.AllocatedCost, c.RealizedProfit = calculation.RealizedRevenue, calculation.AllocatedCost, calculation.RealizedProfit
		c.CommissionBaseAmount, c.CommissionAmount = calculation.CommissionBaseAmount, calculation.CommissionAmount
		create := tx.FinanceCommission.Create().SetID(c.ID).SetOrganizationID(org).SetCommissionNo(c.CommissionNo).SetIdempotencyKey(c.IdempotencyKey).SetEmployeeID(c.EmployeeID).SetEmployeeName(c.EmployeeName).SetCustomerCount(c.CustomerCount).SetOrderCount(c.OrderCount).SetFeeCount(c.FeeCount).SetRuleID(c.RuleID).SetRuleName(c.RuleName).SetPersonnelRole(string(c.PersonnelRole)).SetCalculationBasis(string(c.CalculationBasis)).SetRuleVersion(c.RuleVersion).SetCalculationVersion(c.CalculationVersion).SetSourceFingerprint(c.SourceFingerprint).SetStatus(commission.StatusDRAFT).SetBaseCurrency(c.BaseCurrency).SetRealizedRevenue(c.RealizedRevenue.StringFixed(8)).SetAllocatedCost(c.AllocatedCost.StringFixed(8)).SetRealizedProfit(c.RealizedProfit.StringFixed(8)).SetCommissionBaseAmount(c.CommissionBaseAmount.StringFixed(8)).SetRatePercent(c.RatePercent.StringFixed(4)).SetCommissionAmount(c.CommissionAmount.StringFixed(8)).SetCommissionDate(snapshot.CommissionDate).SetCnyExchangeRate(snapshot.ExchangeRate.StringFixed(8)).SetCnyExchangeRateSource(commission.CnyExchangeRateSource(snapshot.ExchangeRateSource)).SetCnyExchangeRateDate(snapshot.ExchangeRateDate).SetNillableCnyExchangeRateSettingID(snapshot.ExchangeRateSettingID).SetCnyCommissionAmount(snapshot.CommissionAmount.StringFixed(8)).SetNillableNote(c.Note).SetVersion(1)
		// 来源二选一落库：空来源显式置 NULL，保证部分唯一索引语义正确。
		if c.VerificationID != uuid.Nil {
			create = create.SetVerificationID(c.VerificationID).SetVerificationNo(c.VerificationNo)
		}
		if c.NettingID != uuid.Nil {
			create = create.SetNettingID(c.NettingID).SetNettingNo(c.NettingNo)
		}
		if _, err = create.Save(ctx); err != nil {
			return mapEntError(err, nil, biz.ErrCommissionDuplicate)
		}
		lineBuilders, builderErr := financeCommissionLineBuildersFromCalculation(tx.FinanceCommissionLine, org, c.ID, calculation)
		if builderErr != nil {
			return builderErr
		}
		if _, err = tx.FinanceCommissionLine.CreateBulk(lineBuilders...).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// financeCommissionLineBuildersFromCalculation 按计提引擎结果构造提成行写入器：
// 提成创建与申请重提快照刷新共用，保证逐订单行快照形状一致。行 ID 就地回填，
// 组织与提成父单由调用方指定。
func financeCommissionLineBuildersFromCalculation(lineClient *ent.FinanceCommissionLineClient, org, commissionID uuid.UUID, calculation *biz.CommissionCalculation) ([]*ent.FinanceCommissionLineCreate, error) {
	lineBuilders := make([]*ent.FinanceCommissionLineCreate, 0, len(calculation.Lines))
	for _, line := range calculation.Lines {
		line.ID = uuid.Must(uuid.NewV7())
		line.OrganizationID = org
		line.CommissionID = commissionID
		feeSnapshot, marshalErr := json.Marshal(line.Fees)
		if marshalErr != nil {
			return nil, marshalErr
		}
		lineBuilders = append(lineBuilders, lineClient.Create().SetID(line.ID).SetOrganizationID(line.OrganizationID).SetCommissionID(line.CommissionID).SetOrderID(line.OrderID).SetOrderNo(line.OrderNo).SetOrderDate(line.OrderDate).SetCustomerID(line.CustomerID).SetCustomerCode(line.CustomerCode).SetCustomerName(line.CustomerName).SetPersonnelAssignmentID(line.CustomerAssignmentID).SetPersonnelOrganizationID(line.CustomerAssignmentOrganizationID).SetPersonnelAssignedAt(line.CustomerAssignedAt).SetFeeCount(line.FeeCount).SetFeeSnapshot(string(feeSnapshot)).SetEmployeeID(line.EmployeeID).SetEmployeeName(line.EmployeeName).SetPersonnelRole(string(line.PersonnelRole)).SetCalculationBasis(string(line.CalculationBasis)).SetBaseCurrency(line.BaseCurrency).SetRealizedRevenue(line.RealizedRevenue.StringFixed(8)).SetAllocatedCost(line.AllocatedCost.StringFixed(8)).SetRealizedProfit(line.RealizedProfit.StringFixed(8)).SetCommissionBaseAmount(line.CommissionBaseAmount.StringFixed(8)).SetRatePercent(line.RatePercent.StringFixed(4)).SetCommissionAmount(line.CommissionAmount.StringFixed(8)).SetTotalReceivableSnapshot(line.TotalReceivableSnapshot.StringFixed(8)).SetTotalPayableSnapshot(line.TotalPayableSnapshot.StringFixed(8)).SetSnapshotStatus(commissionline.SnapshotStatusREADY).SetSnapshotSource(commissionline.SnapshotSourceNATIVE))
	}
	return lineBuilders, nil
}

func (r *commissionRepo) Transition(ctx context.Context, org, id, actor uuid.UUID, version uint64, target biz.CommissionStatus, reason string, audit *biz.AuditEvent) (*biz.FinanceCommission, error) {
	if err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if target == biz.CommissionConfirmed {
			// CONFIRMED 分支先按既有「来源 → 账单 → 订单 → 费用」锁序完成指纹
			// 复算，避免引入与提成创建路径相反的订单/账单加锁顺序。方案与员工
			// 分配按快照的员工、身份与归属日期重新解析：已生效方案的历史快照
			// 不被名单或方案后续变化重算，但解析失败或指纹漂移会拒绝确认。
			snapshot, lookupErr := tx.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).Only(ctx)
			if lookupErr != nil {
				return mapEntError(lookupErr, biz.ErrCommissionNotFound, nil)
			}
			current, calculateErr := calculateCommission(ctx, commissionStoreFromTx(tx), org, valueOrNilUUID(snapshot.VerificationID), valueOrNilUUID(snapshot.NettingID), snapshot.EmployeeID, derefCommissionPersonnelRole(snapshot.PersonnelRole), true)
			if calculateErr != nil {
				return biz.ErrCommissionSourceChanged
			}
			if snapshot.SourceFingerprint == "" || snapshot.SourceFingerprint != current.SourceFingerprint {
				return biz.ErrCommissionSourceChanged
			}
		}
		// 提成状态迁移会改变订单财务锁证据集合；在修改提成行之前统一按 UUID
		// 升序取得受影响 Order 行锁，保持 Order → 提成父单 → 调整的固定锁序，
		// 避免补录审批复核证据后被并发改写。
		preLines, preLineErr := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(id)).All(ctx)
		if preLineErr != nil {
			return preLineErr
		}
		if lockErr := lockOrdersSortedForFinance(ctx, tx, orderUUIDsFromLines(preLines)); lockErr != nil {
			return lockErr
		}
		if target == biz.CommissionConfirmed {
			orderIDs, orderIDErr := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDEQ(id)).All(ctx)
			if orderIDErr != nil {
				return orderIDErr
			}
			lineOrderIDs := orderUUIDsFromLines(orderIDs)
			hasDraftFees, draftErr := tx.OrderFee.Query().Where(fee.OrderIDIn(lineOrderIDs...), fee.StatusEQ(fee.StatusDRAFT)).Exist(ctx)
			if draftErr != nil {
				return draftErr
			}
			if hasDraftFees {
				return biz.ErrCommissionUnconfirmedFees
			}
		}
		x, err := tx.FinanceCommission.Query().Where(commission.IDEQ(id), commission.OrganizationIDEQ(org)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionNotFound, nil)
		}
		if x.Version != version {
			return biz.ErrCommissionTransition
		}
		now := time.Now()
		update := tx.FinanceCommission.UpdateOneID(id).SetVersion(version + 1)
		switch target {
		case biz.CommissionConfirmed:
			if x.Status != commission.StatusDRAFT {
				return biz.ErrCommissionTransition
			}
			update.SetStatus(commission.StatusCONFIRMED).SetConfirmedAt(now).SetConfirmedBy(actor)
		case biz.CommissionPaid:
			if x.Status != commission.StatusCONFIRMED {
				return biz.ErrCommissionTransition
			}
			update.SetStatus(commission.StatusPAID).SetPaidAt(now).SetPaidBy(actor)
		case biz.CommissionCancelled:
			if x.Status != commission.StatusDRAFT && x.Status != commission.StatusCONFIRMED {
				return biz.ErrCommissionTransition
			}
			hasAdjustments, adjustmentErr := tx.FinanceCommissionAdjustment.Query().Where(adjustment.CommissionIDEQ(id), adjustment.StatusNEQ(adjustment.StatusCANCELLED)).Exist(ctx)
			if adjustmentErr != nil {
				return adjustmentErr
			}
			if hasAdjustments {
				return biz.ErrCommissionTransition
			}
			update.SetStatus(commission.StatusCANCELLED).SetCancelledAt(now).SetCancelledBy(actor).SetCancellationReason(reason)
		default:
			return biz.ErrCommissionInvalid
		}
		if _, err = update.Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	}); err != nil {
		return nil, err
	}
	return r.Get(ctx, org, id)
}

func valueOrNilUUID(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}

// confirmCommissionsForApplicationApproval 是月度申请整单批准在共享事务内的
// 整批确认辅助（design §5.2）：对申请明细对应的 DRAFT 提成保持与 Transition
// CONFIRMED 分支完全一致的复核与写入语义——逐笔重算来源指纹（方案员工分配
// 唯一命中、同员工/身份/方案快照一致）、草稿费用阻断、状态检查，确认时版本
// 递增并写 CONFIRMED + confirmed_at/by + 逐笔确认审计，保持订单财务锁净额
// 联动口径不变。本函数不开启独立事务、不做部分批准：任一明细复核失败立即
// 返回领域错误，由调用方整体回滚。调用方必须已锁定申请头（expected_version
// + PENDING_REVIEW）与申请明细；申请明细来源订单与提成父单在此按 design
// §5.2 步骤 2 的固定锁序取得（订单升序 → 提成父单按 commission_id 升序）。
func confirmCommissionsForApplicationApproval(ctx context.Context, tx *ent.Tx, org, actor uuid.UUID, commissionIDs []uuid.UUID) error {
	if len(commissionIDs) == 0 {
		return biz.ErrCommissionApplicationSourceConflict
	}
	sorted := uniqueSortedUUIDs(commissionIDs)
	lineRows, err := tx.FinanceCommissionLine.Query().Where(commissionline.CommissionIDIn(sorted...)).All(ctx)
	if err != nil {
		return err
	}
	if err := lockOrdersSortedForFinance(ctx, tx, orderUUIDsFromLines(lineRows)); err != nil {
		return err
	}
	parents, err := tx.FinanceCommission.Query().
		Where(commission.IDIn(sorted...), commission.OrganizationIDEQ(org)).
		Order(ent.Asc(commission.FieldID)).
		ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	if len(parents) != len(sorted) {
		return biz.ErrCommissionApplicationSourceConflict
	}
	for _, parent := range parents {
		if parent.Status != commission.StatusDRAFT {
			return biz.ErrCommissionApplicationSourceConflict
		}
		// 与 Transition CONFIRMED 分支同口径：按快照的员工、身份与来源归属日期
		// 完整重算，解析失败或指纹漂移一律视为来源已变化。
		current, calculateErr := calculateCommission(ctx, commissionStoreFromTx(tx), org,
			valueOrNilUUID(parent.VerificationID), valueOrNilUUID(parent.NettingID),
			parent.EmployeeID, derefCommissionPersonnelRole(parent.PersonnelRole), true)
		if calculateErr != nil {
			return biz.ErrCommissionSourceChanged
		}
		if parent.SourceFingerprint == "" || parent.SourceFingerprint != current.SourceFingerprint {
			return biz.ErrCommissionSourceChanged
		}
	}
	orderIDs := orderUUIDsFromLines(lineRows)
	hasDraftFees, err := tx.OrderFee.Query().Where(fee.OrderIDIn(orderIDs...), fee.StatusEQ(fee.StatusDRAFT)).Exist(ctx)
	if err != nil {
		return err
	}
	if hasDraftFees {
		return biz.ErrCommissionUnconfirmedFees
	}
	now := time.Now()
	for _, parent := range parents {
		if _, updateErr := tx.FinanceCommission.UpdateOneID(parent.ID).
			SetVersion(parent.Version + 1).
			SetStatus(commission.StatusCONFIRMED).
			SetConfirmedAt(now).
			SetConfirmedBy(actor).
			Save(ctx); updateErr != nil {
			return updateErr
		}
		if auditErr := writeAudit(ctx, tx.AuditLog, &biz.AuditEvent{
			OrganizationID: &org,
			UserID:         &actor,
			Action:         "finance.commission.confirm",
			Result:         "success",
			ResourceType:   "finance_commission",
			ResourceID:     parent.ID.String(),
		}); auditErr != nil {
			return auditErr
		}
	}
	return nil
}

// derefCommissionPersonnelRole 读取提成快照中的人员身份；存量行身份为空时返回
// 非法值，由后续解析稳定拒绝。
func derefCommissionPersonnelRole(value *string) biz.CommissionPersonnelRole {
	if value == nil {
		return ""
	}
	return biz.CommissionPersonnelRole(*value)
}

// orderUUIDsFromLines 汇总提成行的订单 ID。
func orderUUIDsFromLines(lines []*ent.FinanceCommissionLine) []uuid.UUID {
	orderIDs := make([]uuid.UUID, 0, len(lines))
	for _, line := range lines {
		orderIDs = append(orderIDs, line.OrderID)
	}
	return orderIDs
}

// lockOrdersSortedForFinance 按 UUID 升序锁定受影响的订单行：所有会改变财务锁
// 净额或证据集合的提成/调整状态迁移统一保持 Order → 提成父单 → 调整的固定锁
// 序，防止补录审批复核证据后被并发改写。取得 Order 锁仅做线性化互斥。
func lockOrdersSortedForFinance(ctx context.Context, tx *ent.Tx, orderIDs []uuid.UUID) error {
	if len(orderIDs) == 0 {
		return nil
	}
	sorted := uniqueSortedUUIDs(orderIDs)
	_, err := tx.Order.Query().Where(orderent.IDIn(sorted...)).Order(ent.Asc(orderent.FieldID)).ForUpdate().All(ctx)
	return err
}
