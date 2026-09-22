package data

import (
	"context"
	"fmt"
	"sort"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

// ResolveFeeFactsForApproval 在审批事务的锁内复核费用快照引用：结算对象、费用
// 项、计费单位与币种必须仍然有效。汇率与金额的重解析由用例层复用现有解析与
// 计算入口完成，本方法只做引用校验并返回快照原值。
func (r *orderFeeSupplementRepo) ResolveFeeFactsForApproval(ctx context.Context, organizationID, orderID uuid.UUID, snapshot *biz.OrderFeeSupplementFeeSnapshot) (*biz.OrderFee, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	applicability, err := loadFeeApplicability(ctx, client, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	if snapshot.FeeSettingID != nil {
		feeSetting, settingErr := client.FeeSetting.Query().
			Where(feesettingent.IDEQ(*snapshot.FeeSettingID), feesettingent.OrganizationIDEQ(organizationID), feesettingent.EnabledEQ(true)).
			Only(ctx)
		if settingErr != nil || !feeSettingApplies(feeSetting, applicability) {
			return nil, biz.ErrOrderFeeSettingInvalid
		}
	}
	if snapshot.BillingUnitID != nil {
		billingUnit, unitErr := client.BillingUnit.Query().Where(billingunitent.IDEQ(*snapshot.BillingUnitID), billingunitent.EnabledEQ(true)).Only(ctx)
		if unitErr != nil || !billingUnit.Enabled {
			return nil, biz.ErrOrderFeeBillingUnitInvalid
		}
	}
	party, partyErr := client.Partner.Query().Where(partnerent.IDEQ(snapshot.SettlementPartyID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).Only(ctx)
	if partyErr != nil {
		return nil, biz.ErrOrderFeePartyInvalid
	}
	validCurrency, currencyErr := client.Currency.Query().Where(currencyent.CodeEQ(snapshot.Currency), currencyent.EnabledEQ(true)).Exist(ctx)
	if currencyErr != nil {
		return nil, currencyErr
	}
	if !validCurrency {
		return nil, biz.ErrOrderFeeCurrencyInvalid
	}
	fee := &biz.OrderFee{
		OrderID:               orderID,
		Direction:             snapshot.Direction,
		FeeSettingID:          snapshot.FeeSettingID,
		FeeCode:               snapshot.FeeCode,
		FeeName:               snapshot.FeeName,
		FeeNameEN:             snapshot.FeeNameEN,
		SettlementPartyID:     snapshot.SettlementPartyID,
		SettlementPartyName:   party.LegalName,
		BillingUnitID:         snapshot.BillingUnitID,
		BillingUnit:           snapshot.BillingUnit,
		TaxRate:               snapshot.TaxRate,
		TaxableServiceName:    snapshot.TaxableServiceName,
		Quantity:              snapshot.Quantity,
		UnitPrice:             snapshot.UnitPrice,
		TotalAmount:           snapshot.TotalAmount,
		TaxInclusive:          snapshot.TaxInclusive,
		Currency:              snapshot.Currency,
		ExchangeRate:          snapshot.ExchangeRate,
		ExchangeRateSource:    snapshot.ExchangeRateSource,
		ExchangeRateDate:      snapshot.ExchangeRateDate,
		ExchangeRateSettingID: snapshot.ExchangeRateSettingID,
		BaseCurrency:          snapshot.BaseCurrency,
		BaseCurrencyAmount:    snapshot.BaseCurrencyAmount,
		ExpenseDate:           snapshot.ExpenseDate,
		Note:                  snapshot.Note,
	}
	return fee, nil
}

// LockCommissionImpactContext 按父单 UUID 升序 FOR UPDATE 锁定包含目标订单的
// CONFIRMED/PAID 提成父单、目标订单行与全部相关调整，并固化边际影响计算上下文。
func (r *orderFeeSupplementRepo) LockCommissionImpactContext(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeSupplementImpactContext, error) {
	var result *biz.OrderFeeSupplementImpactContext
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		client := tx.Client()
		// 此前已审批且尚未作废的补录应付合计：已作废不进基线。
		priorFees, err := client.OrderFee.Query().Where(
			orderfeeent.OrderIDEQ(orderID),
			orderfeeent.SupplementRequestIDNotNil(),
			orderfeeent.StatusNEQ(orderfeeent.StatusCANCELLED),
		).All(ctx)
		if err != nil {
			return err
		}
		priorBase := decimal.Zero
		for _, item := range priorFees {
			amount, parseErr := decimalOf(item.BaseCurrencyAmount)
			if parseErr != nil {
				return parseErr
			}
			priorBase = priorBase.Add(amount)
		}
		lines, err := client.FinanceCommissionLine.Query().Where(
			commissionlineent.OrganizationIDEQ(organizationID),
			commissionlineent.OrderIDEQ(orderID),
			commissionlineent.HasCommissionWith(commissionent.StatusIn(commissionent.StatusCONFIRMED, commissionent.StatusPAID)),
		).All(ctx)
		if err != nil {
			return err
		}
		commissionIDs := make([]uuid.UUID, 0, len(lines))
		for _, line := range lines {
			commissionIDs = append(commissionIDs, line.CommissionID)
		}
		// 多张父单按 UUID 升序一次性加锁，固定加锁顺序防死锁。
		sort.Slice(commissionIDs, func(i, j int) bool { return commissionIDs[i].String() < commissionIDs[j].String() })
		result = &biz.OrderFeeSupplementImpactContext{PriorSupplementBaseAmount: priorBase.Round(8), Lines: make([]*biz.OrderFeeSupplementCommissionLine, 0, len(commissionIDs))}
		lineByCommission := make(map[uuid.UUID]*ent.FinanceCommissionLine, len(lines))
		for _, line := range lines {
			lineByCommission[line.CommissionID] = line
		}
		for _, commissionID := range commissionIDs {
			parent, parentErr := tx.FinanceCommission.Query().Where(commissionent.IDEQ(commissionID), commissionent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
			if parentErr != nil {
				return parentErr
			}
			line := lineByCommission[commissionID]
			adjustments, adjustmentErr := tx.FinanceCommissionAdjustment.Query().Where(
				commissionadjustmentent.CommissionIDEQ(commissionID),
			).Order(commissionadjustmentent.ByID()).ForUpdate().All(ctx)
			if adjustmentErr != nil {
				return adjustmentErr
			}
			contextLine, convertErr := commissionImpactLineToBiz(parent, line, adjustments)
			if convertErr != nil {
				return convertErr
			}
			result.Lines = append(result.Lines, contextLine)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// CreateApprovedFee 在审批事务内创建与申请一对一关联的 CONFIRMED 补录费用：
// 幂等键由申请 ID 派生，supplement_request_id 反向关联；唯一约束兜底并发重复。
func (r *orderFeeSupplementRepo) CreateApprovedFee(ctx context.Context, organizationID, requestID uuid.UUID, fee *biz.OrderFee, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	var created *ent.OrderFee
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		builder := tx.OrderFee.Create().
			SetID(fee.ID).
			SetOrderID(fee.OrderID).
			SetIdempotencyKey(fee.IdempotencyKey).
			SetDirection(orderfeeent.Direction(fee.Direction)).
			SetStatus(orderfeeent.StatusCONFIRMED).
			SetFeeCode(fee.FeeCode).
			SetFeeName(fee.FeeName).
			SetSettlementPartyID(fee.SettlementPartyID).
			SetBillingUnit(fee.BillingUnit).
			SetQuantity(fee.Quantity.StringFixed(4)).
			SetUnitPrice(fee.UnitPrice.StringFixed(4)).
			SetTotalAmount(fee.TotalAmount.StringFixed(8)).
			SetTaxInclusive(fee.TaxInclusive).
			SetNetAmount(fee.NetAmount.StringFixed(8)).
			SetTaxAmount(fee.TaxAmount.StringFixed(8)).
			SetCurrency(fee.Currency).
			SetExchangeRate(fee.ExchangeRate.StringFixed(8)).
			SetExchangeRateSource(orderfeeent.ExchangeRateSource(fee.ExchangeRateSource)).
			SetExchangeRateDate(fee.ExchangeRateDate).
			SetBaseCurrency(fee.BaseCurrency).
			SetBaseCurrencyAmount(fee.BaseCurrencyAmount.StringFixed(8)).
			SetExpenseDate(fee.ExpenseDate).
			SetSupplementRequestID(requestID).
			SetVersion(1)
		if fee.FeeSettingID != nil {
			builder.SetFeeSettingID(*fee.FeeSettingID)
		}
		if fee.FeeNameEN != nil {
			builder.SetFeeNameEn(*fee.FeeNameEN)
		}
		if fee.BillingUnitID != nil {
			builder.SetBillingUnitID(*fee.BillingUnitID)
		}
		if fee.TaxRate != nil {
			builder.SetTaxRate(fee.TaxRate.StringFixed(2))
		}
		if fee.TaxableServiceName != nil {
			builder.SetTaxableServiceName(*fee.TaxableServiceName)
		}
		if fee.ExchangeRateSettingID != nil {
			builder.SetExchangeRateSettingID(*fee.ExchangeRateSettingID)
		}
		if fee.Note != nil {
			builder.SetNote(*fee.Note)
		}
		item, saveErr := builder.Save(ctx)
		if saveErr != nil {
			if ent.IsConstraintError(saveErr) {
				return biz.ErrOrderFeeIdempotencyConflict
			}
			return saveErr
		}
		created = item
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	loaded, err := client.OrderFee.Query().Where(orderfeeent.IDEQ(created.ID)).WithSettlementParty().Only(ctx)
	if err != nil {
		return nil, err
	}
	return orderFeeToBiz(loaded)
}

// CreateDecreaseSuggestion 在审批事务内创建 DECREASE+DRAFT+LOCKED_FEE_SUPPLEMENT
// 调整：父单行内分配调整号并递增序号；来源关联由数据库 CHECK 强制指向补录申请。
func (r *orderFeeSupplementRepo) CreateDecreaseSuggestion(ctx context.Context, organizationID, requestID uuid.UUID, suggestion *biz.OrderFeeSupplementDecreaseSuggestion, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		parent, err := tx.FinanceCommission.Query().Where(commissionent.IDEQ(suggestion.CommissionID), commissionent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionNotFound, nil)
		}
		if parent.Status != commissionent.StatusCONFIRMED && parent.Status != commissionent.StatusPAID {
			return biz.ErrCommissionAdjustmentTransition
		}
		line, err := tx.FinanceCommissionLine.Query().Where(commissionlineent.CommissionIDEQ(parent.ID), commissionlineent.OrderIDEQ(suggestion.OrderID)).Only(ctx)
		if err != nil {
			return mapEntError(err, biz.ErrCommissionAdjustmentInvalid, nil)
		}
		sequence := parent.AdjustmentSequence + 1
		idempotencyKey := fmt.Sprintf("lfs:%s:%s:%s", requestID, suggestion.CommissionID, suggestion.OrderID)
		if _, err = tx.FinanceCommissionAdjustment.Create().
			SetID(suggestion.AdjustmentID).
			SetOrganizationID(organizationID).
			SetCommissionID(parent.ID).
			SetOrderID(line.OrderID).
			SetAdjustmentNo(fmt.Sprintf("%s-ADJ%03d", parent.CommissionNo, sequence)).
			SetIdempotencyKey(idempotencyKey).
			SetCommissionNo(parent.CommissionNo).
			SetOrderNo(line.OrderNo).
			SetEmployeeID(parent.EmployeeID).
			SetEmployeeName(parent.EmployeeName).
			SetSourceType(commissionadjustmentent.SourceTypeLOCKED_FEE_SUPPLEMENT).
			SetSourceFeeSupplementRequestID(requestID).
			SetDirection(commissionadjustmentent.DirectionDECREASE).
			SetStatus(commissionadjustmentent.StatusDRAFT).
			SetBaseCurrency(parent.BaseCurrency).
			SetAmount(suggestion.Amount.StringFixed(8)).
			SetReason(suggestion.Reason).
			SetVersion(1).
			Save(ctx); err != nil {
			if ent.IsConstraintError(err) {
				return biz.ErrFeeSupplementTransition
			}
			return err
		}
		if _, err = tx.FinanceCommission.UpdateOne(parent).SetAdjustmentSequence(sequence).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// ---------------------------------------------------------------------------
// 通知入队
// ---------------------------------------------------------------------------
