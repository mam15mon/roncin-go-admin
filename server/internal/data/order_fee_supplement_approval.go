package data

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

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
// 计算入口完成；提交时固化汇率，审批不重读主数据汇率。
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

// LockCommissionImpactContext 复用当前经济配额和原快照规则，费用系统与人工净额分开。
func (r *orderFeeSupplementRepo) LockCommissionImpactContext(ctx context.Context, org, orderID uuid.UUID) (*biz.OrderFeeSupplementImpactContext, error) {
	var result *biz.OrderFeeSupplementImpactContext
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		p, err := loadCommissionEconomicProjection(ctx, tx.Client(), orderID, nil)
		if err != nil {
			return err
		}
		plans, err := planCommissionEconomicRows(ctx, tx.Client(), org, orderID, p, true)
		if err != nil {
			return err
		}
		result = &biz.OrderFeeSupplementImpactContext{Lines: make([]*biz.OrderFeeSupplementCommissionLine, 0, len(plans))}
		for _, plan := range plans {
			l, err := commissionImpactLineToBiz(plan.Parent, plan.Line, plan.Rows)
			if err != nil {
				return err
			}
			revenue, cost := p.Receivable, p.Payable
			l.TotalReceivableSnapshot = &revenue
			l.TotalPayableSnapshot = &cost
			kind, id := "VERIFICATION", uuid.Nil
			if plan.Parent.VerificationID != nil {
				id = *plan.Parent.VerificationID
			} else if plan.Parent.NettingID != nil {
				kind, id = "NETTING", *plan.Parent.NettingID
			}
			l.RealizedRevenue = p.SourceRevenue(kind, id)
			l.EconomicFingerprint = p.Fingerprint
			l.ParentOther, err = decimalOf(plan.Parent.CommissionAmount)
			if err != nil {
				return err
			}
			for _, row := range plan.Rows {
				if row.Status != commissionadjustmentent.StatusCONFIRMED && row.Status != commissionadjustmentent.StatusPAID {
					continue
				}
				v, err := commissionSignedAdjustment(row)
				if err != nil {
					return err
				}
				if row.OrderID == orderID && commissionSystemSource(row.SourceType) {
					if row.Status == commissionadjustmentent.StatusPAID {
						l.SystemPaid = l.SystemPaid.Add(v)
					} else {
						l.SystemPending = l.SystemPending.Add(v)
						l.PendingSystemAdjustmentIDs = append(l.PendingSystemAdjustmentIDs, row.ID)
					}
				} else {
					l.ParentOther = l.ParentOther.Add(v)
					if row.OrderID == orderID {
						l.LineOther = l.LineOther.Add(v)
					}
				}
			}
			result.Lines = append(result.Lines, l)
		}
		return nil
	})
	return result, err
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
	loaded, err := client.OrderFee.Query().Where(orderfeeent.IDEQ(created.ID)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Only(ctx)
	if err != nil {
		return nil, err
	}
	return orderFeeToBiz(loaded)
}

// CreateDecreaseSuggestion 在审批事务内创建 DECREASE+CONFIRMED+LOCKED_FEE_SUPPLEMENT
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
		if len(suggestion.CancelAdjustmentIDs) > 0 {
			if _, err := tx.FinanceCommissionAdjustment.Update().Where(commissionadjustmentent.IDIn(suggestion.CancelAdjustmentIDs...), commissionadjustmentent.CommissionIDEQ(parent.ID), commissionadjustmentent.StatusEQ(commissionadjustmentent.StatusCONFIRMED)).SetStatus(commissionadjustmentent.StatusCANCELLED).SetCancelledAt(time.Now().UTC()).SetNillableCancelledBy(audit.UserID).SetCancellationReason("补录后按系统累计净额重建").AddVersion(1).Save(ctx); err != nil {
				return err
			}
		}
		if !suggestion.Amount.IsPositive() {
			return writeAudit(ctx, tx.AuditLog, audit)
		}
		direction := commissionadjustmentent.Direction(suggestion.Direction)
		if direction == "" {
			return biz.ErrCommissionAdjustmentInvalid
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
			SetDirection(direction).
			SetStatus(commissionadjustmentent.StatusCONFIRMED).
			SetConfirmedAt(time.Now().UTC()).
			SetNillableConfirmedBy(audit.UserID).
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
		if _, err = tx.FinanceCommission.UpdateOne(parent).SetAdjustmentSequence(sequence).SetVersion(parent.Version + 1).Save(ctx); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// ---------------------------------------------------------------------------
// 通知入队
// ---------------------------------------------------------------------------
