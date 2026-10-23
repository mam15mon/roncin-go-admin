package data

import (
	"context"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
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
