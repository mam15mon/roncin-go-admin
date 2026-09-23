package data

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	financecustomsettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecustomsetting"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderservicetypeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderservicetype"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	"github.com/shopspring/decimal"
)

type orderFeeRepo struct {
	data *Data
}

type orderFeeApplicability struct {
	serviceTypeIDs  map[uuid.UUID]struct{}
	abnormalCaseIDs map[uuid.UUID]struct{}
}

func NewOrderFeeRepo(data *Data) biz.OrderFeeRepo {
	return &orderFeeRepo{data: data}
}

func (r *orderFeeRepo) order(ctx context.Context, organizationID, orderID uuid.UUID) error {
	client, err := r.data.client(ctx)
	if err != nil {
		return err
	}
	exists, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrOrderFeeNotFound
	}
	return nil
}

func (r *orderFeeRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderFee, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.OrderFee.Query().
		Where(orderfeeent.OrderIDEQ(orderID)).
		WithSettlementParty().
		Order(orderfeeent.ByDirection(), orderfeeent.ByCreatedAt(), orderfeeent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]*biz.OrderFee, 0, len(items))
	for _, item := range items {
		converted, err := orderFeeToBiz(item)
		if err != nil {
			return nil, err
		}
		result = append(result, converted)
	}
	return result, nil
}

func (r *orderFeeRepo) Get(ctx context.Context, organizationID, orderID, id uuid.UUID) (*biz.OrderFee, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.OrderFee.Query().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(organizationID))).WithSettlementParty().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeeNotFound, nil)
	}
	return orderFeeToBiz(item)
}

func (r *orderFeeRepo) BilledBillContext(ctx context.Context, organizationID, orderID, id uuid.UUID) (*biz.BilledFeeBillContext, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	line, err := client.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDEQ(id), financebilllineent.ActiveEQ(true), financebilllineent.OrderIDEQ(orderID)).WithBill(func(query *ent.FinanceBillQuery) {
		query.Where(financebillent.OrganizationIDEQ(organizationID))
	}).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrBilledFeeBillLocked, nil)
	}
	bill, err := line.Edges.BillOrErr()
	if err != nil {
		return nil, err
	}
	return &biz.BilledFeeBillContext{BillID: bill.ID, Status: biz.FinanceBillStatus(bill.Status), BillDate: bill.BillDate, Currency: bill.Currency, FeeCount: bill.FeeCount}, nil
}

func (r *orderFeeRepo) GetByIdempotencyKey(ctx context.Context, organizationID, orderID uuid.UUID, idempotencyKey string) (*biz.OrderFee, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.OrderFee.Query().
		Where(
			orderfeeent.OrderIDEQ(orderID),
			orderfeeent.IdempotencyKeyEQ(idempotencyKey),
			orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(organizationID)),
		).
		WithSettlementParty().
		Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return orderFeeToBiz(item)
}

func (r *orderFeeRepo) Options(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeOptions, error) {
	applicability, err := r.loadApplicability(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	businessOrder, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).WithCustomer().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeeNotFound, nil)
	}
	customer, err := businessOrder.Edges.CustomerOrErr()
	if err != nil {
		return nil, err
	}
	parties, err := client.Partner.Query().
		Where(partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).
		Order(partnerent.ByLegalName(), partnerent.ByCode()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	currencies, err := client.Currency.Query().
		Where(currencyent.EnabledEQ(true)).
		Order(currencyent.ByCode()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	billingUnits, err := client.BillingUnit.Query().
		Where(billingunitent.EnabledEQ(true)).
		Order(billingunitent.BySortOrder(), billingunitent.ByCode(), billingunitent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	feeSettings, err := client.FeeSetting.Query().
		Where(feesettingent.OrganizationIDEQ(organizationID), feesettingent.EnabledEQ(true)).
		WithBillingUnit().WithTaxableService().
		Order(feesettingent.BySortOrder(), feesettingent.ByFeeCode(), feesettingent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := &biz.OrderFeeOptions{
		SettlementParties: make([]biz.OrderFeeSettlementPartyOption, 0, len(parties)),
		Currencies:        make([]biz.OrderFeeCurrencyOption, 0, len(currencies)),
		FeeSettings:       make([]biz.OrderFeeSettingOption, 0, len(feeSettings)),
		BillingUnits:      make([]biz.OrderFeeBillingUnitOption, 0, len(billingUnits)),
		CustomerID:        customer.ID,
		CustomerName:      customer.LegalName,
	}
	lockNos, err := r.financeLockCommissionNos(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	if len(lockNos) > 0 {
		result.FinanceLocked = true
		result.FinanceLockCommissionNos = lockNos
		result.FinanceLockReason = "关联提成已确认或已发放，原费用事实已锁定"
	}
	for _, party := range parties {
		result.SettlementParties = append(result.SettlementParties, biz.OrderFeeSettlementPartyOption{ID: party.ID, Code: partnerCodeValue(party.Code), Name: party.LegalName})
	}
	enabledCurrencies := make(map[string]struct{}, len(currencies))
	for _, currency := range currencies {
		enabledCurrencies[currency.Code] = struct{}{}
		result.Currencies = append(result.Currencies, biz.OrderFeeCurrencyOption{Code: currency.Code, Name: currency.Name, MinorUnit: currency.MinorUnit})
	}
	for _, billingUnit := range billingUnits {
		result.BillingUnits = append(result.BillingUnits, biz.OrderFeeBillingUnitOption{ID: billingUnit.ID, Code: billingUnit.Code, Name: billingUnit.Name, QuantityMustBeInteger: billingUnit.QuantityMustBeInteger})
	}
	for _, feeSetting := range feeSettings {
		billingUnit, billingErr := feeSetting.Edges.BillingUnitOrErr()
		taxableService, taxableErr := feeSetting.Edges.TaxableServiceOrErr()
		_, currencyEnabled := enabledCurrencies[feeSetting.DefaultCurrency]
		if billingErr != nil || taxableErr != nil || !billingUnit.Enabled || !taxableService.Enabled || !currencyEnabled || !feeSettingApplies(feeSetting, applicability) {
			continue
		}
		taxRate, parseErr := decimalOf(feeSetting.TaxRate)
		if parseErr != nil {
			return nil, parseErr
		}
		result.FeeSettings = append(result.FeeSettings, biz.OrderFeeSettingOption{
			ID: feeSetting.ID, FeeCode: feeSetting.FeeCode, NameZH: feeSetting.NameZh, NameEN: feeSetting.NameEn, AliasName: feeSetting.AliasName,
			DefaultCurrency: feeSetting.DefaultCurrency, DefaultBillingUnitID: billingUnit.ID, DefaultBillingUnitName: billingUnit.Name,
			TaxRate: taxRate, TaxableServiceName: taxableService.Name,
		})
	}
	return result, nil
}

// financeLockCommissionNos 返回仍使订单处于财务锁定的提成单号：与台账净额口径
// 一致，有效提成净额 ≤ 0（已被全额冲减）时不视为锁定，费用编辑锁释放。
func (r *orderFeeRepo) financeLockCommissionNos(ctx context.Context, organizationID, orderID uuid.UUID) ([]string, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	items, err := client.FinanceCommissionLine.Query().Where(
		commissionlineent.OrganizationIDEQ(organizationID),
		commissionlineent.OrderIDEQ(orderID),
		commissionlineent.HasCommissionWith(commissionent.StatusIn(commissionent.StatusCONFIRMED, commissionent.StatusPAID)),
	).WithCommission().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, nil
	}
	adjustments, err := client.FinanceCommissionAdjustment.Query().Where(
		commissionadjustmentent.OrganizationIDEQ(organizationID),
		commissionadjustmentent.OrderIDEQ(orderID),
		commissionadjustmentent.StatusIn(commissionadjustmentent.StatusCONFIRMED, commissionadjustmentent.StatusPAID),
	).All(ctx)
	if err != nil {
		return nil, err
	}
	netAmount, err := financeCommissionLockNetAmount(items, adjustments)
	if err != nil {
		return nil, err
	}
	if !netAmount.IsPositive() {
		return nil, nil
	}
	result := make([]string, 0, len(items))
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		parent, edgeErr := item.Edges.CommissionOrErr()
		if edgeErr != nil {
			return nil, edgeErr
		}
		if _, ok := seen[parent.CommissionNo]; ok {
			continue
		}
		seen[parent.CommissionNo] = struct{}{}
		result = append(result, parent.CommissionNo)
	}
	return result, nil
}

func lockOrderForFeeMutation(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID) error {
	order, err := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if err != nil {
		return mapEntError(err, biz.ErrOrderFeeNotFound, nil)
	}
	if err := ensureOrderBusinessEditable(ctx, tx, order); err != nil {
		return err
	}
	// 财务锁复用台账的统一净额谓词（同一 SQL 语义）：有效提成净额 ≤ 0 视为已被
	// 全额冲减，释放费用编辑锁；净额 > 0 才拒绝写入，避免“列表显示已解锁、写入仍被拒”。
	locked, err := tx.Order.Query().Where(orderent.IDEQ(orderID), financeLockedOrderPredicate()).Exist(ctx)
	if err != nil {
		return err
	}
	if locked {
		return biz.ErrOrderFeeFinanceLocked
	}
	return nil
}

func (r *orderFeeRepo) ResolveCatalog(ctx context.Context, organizationID, orderID, feeSettingID, billingUnitID uuid.UUID, allowDisabledUnit bool) (*biz.OrderFeeCatalogSnapshot, error) {
	applicability, err := r.loadApplicability(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	feeSetting, err := client.FeeSetting.Query().
		Where(feesettingent.IDEQ(feeSettingID), feesettingent.OrganizationIDEQ(organizationID), feesettingent.EnabledEQ(true)).
		WithBillingUnit().WithTaxableService().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeeSettingInvalid, nil)
	}
	defaultBillingUnit, billingErr := feeSetting.Edges.BillingUnitOrErr()
	taxableService, taxableErr := feeSetting.Edges.TaxableServiceOrErr()
	defaultCurrencyEnabled, err := client.Currency.Query().Where(currencyent.CodeEQ(feeSetting.DefaultCurrency), currencyent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if billingErr != nil || taxableErr != nil || (!defaultBillingUnit.Enabled && !(allowDisabledUnit && defaultBillingUnit.ID == billingUnitID)) || !taxableService.Enabled || !defaultCurrencyEnabled || !feeSettingApplies(feeSetting, applicability) {
		return nil, biz.ErrOrderFeeSettingInvalid
	}
	billingUnit, err := client.BillingUnit.Query().
		Where(billingunitent.IDEQ(billingUnitID)).ForShare().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeeBillingUnitInvalid, nil)
	}
	if !billingUnit.Enabled && !allowDisabledUnit {
		return nil, biz.ErrOrderFeeBillingUnitInvalid
	}
	taxRate, err := decimalOf(feeSetting.TaxRate)
	if err != nil {
		return nil, err
	}
	return &biz.OrderFeeCatalogSnapshot{
		FeeCode: feeSetting.FeeCode, FeeName: feeSetting.NameZh, FeeNameEN: feeSetting.NameEn,
		BillingUnit: billingUnit.Name, QuantityMustBeInteger: billingUnit.QuantityMustBeInteger, TaxRate: taxRate, TaxableServiceName: taxableService.Name,
	}, nil
}

func (r *orderFeeRepo) loadApplicability(ctx context.Context, organizationID, orderID uuid.UUID) (*orderFeeApplicability, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
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

func feeSettingApplies(feeSetting *ent.FeeSetting, applicability *orderFeeApplicability) bool {
	if _, ok := applicability.serviceTypeIDs[feeSetting.ChargeCategoryID]; !ok {
		return false
	}
	if feeSetting.AbnormalCaseID != nil {
		if _, ok := applicability.abnormalCaseIDs[*feeSetting.AbnormalCaseID]; !ok {
			return false
		}
	}
	return true
}

func (r *orderFeeRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderFee, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	var created *ent.OrderFee
	var partyLegalName string
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if lockErr := lockOrderForFeeMutation(ctx, tx, organizationID, orderID); lockErr != nil {
			return lockErr
		}
		if input.BillingUnitID == nil {
			return biz.ErrOrderFeeBillingUnitInvalid
		}
		unit, unitErr := tx.BillingUnit.Query().Where(billingunitent.IDEQ(*input.BillingUnitID)).ForShare().Only(ctx)
		if unitErr != nil {
			return mapEntError(unitErr, biz.ErrOrderFeeBillingUnitInvalid, nil)
		}
		if !unit.Enabled {
			return biz.ErrOrderFeeBillingUnitInvalid
		}
		if err := biz.ValidateFeeQuantityForUnit(input.Quantity, unit.QuantityMustBeInteger); err != nil {
			return err
		}
		party, queryErr := tx.Partner.Query().Where(partnerent.IDEQ(input.SettlementPartyID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderFeePartyInvalid, nil)
		}
		partyLegalName = party.LegalName
		validCurrency, currencyErr := tx.Currency.Query().Where(currencyent.CodeEQ(input.Currency), currencyent.EnabledEQ(true)).Exist(ctx)
		if currencyErr != nil {
			return currencyErr
		}
		if !validCurrency {
			return biz.ErrOrderFeeCurrencyInvalid
		}
		var createErr error
		created, createErr = tx.OrderFee.Create().
			SetID(input.ID).
			SetOrderID(orderID).
			SetIdempotencyKey(input.IdempotencyKey).
			SetDirection(orderfeeent.Direction(input.Direction)).
			SetStatus(orderfeeent.Status(input.Status)).
			SetNillableFeeSettingID(input.FeeSettingID).
			SetFeeCode(input.FeeCode).
			SetFeeName(input.FeeName).
			SetNillableFeeNameEn(input.FeeNameEN).
			SetSettlementPartyID(input.SettlementPartyID).
			SetNillableBillingUnitID(input.BillingUnitID).
			SetBillingUnit(input.BillingUnit).
			SetNillableTaxRate(decimalPointerToString(input.TaxRate, 2)).
			SetNillableTaxableServiceName(input.TaxableServiceName).
			SetQuantity(input.Quantity.StringFixed(4)).
			SetUnitPrice(input.UnitPrice.StringFixed(4)).
			SetTotalAmount(input.TotalAmount.StringFixed(8)).
			SetTaxInclusive(input.TaxInclusive).
			SetNetAmount(input.NetAmount.StringFixed(8)).
			SetTaxAmount(input.TaxAmount.StringFixed(8)).
			SetCurrency(input.Currency).
			SetExchangeRate(input.ExchangeRate.StringFixed(8)).
			SetExchangeRateSource(orderfeeent.ExchangeRateSource(input.ExchangeRateSource)).
			SetExchangeRateDate(input.ExchangeRateDate).
			SetNillableExchangeRateSettingID(input.ExchangeRateSettingID).
			SetBaseCurrency(input.BaseCurrency).
			SetBaseCurrencyAmount(input.BaseCurrencyAmount.StringFixed(8)).
			SetExpenseDate(input.ExpenseDate).
			SetNillableNote(input.Note).
			SetVersion(input.Version).
			Save(ctx)
		if createErr != nil {
			return mapEntConstraint(createErr, "orderfee_order_id_idempotency_key", biz.ErrOrderFeeIdempotencyConflict)
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	input.SettlementPartyName = partyLegalName
	input.OrderID = orderID
	input.CreatedAt = created.CreatedAt
	input.UpdatedAt = created.UpdatedAt
	return input, nil
}

func (r *orderFeeRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderFee, billExchangeRate *biz.ResolvedRate, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	var item *ent.OrderFee
	var activeLine *ent.FinanceBillLine
	var activeBill *ent.FinanceBill
	var party *ent.Partner
	var updated *ent.OrderFee
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if lockErr := lockOrderForFeeMutation(ctx, tx, organizationID, orderID); lockErr != nil {
			return lockErr
		}
		itemSnapshot, queryErr := tx.OrderFee.Query().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID)).Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderFeeNotFound, nil)
		}
		if itemSnapshot.Status == orderfeeent.StatusBILLED {
			lineSnapshot, lineErr := tx.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDEQ(id), financebilllineent.ActiveEQ(true)).Only(ctx)
			if lineErr != nil {
				return mapEntError(lineErr, biz.ErrBilledFeeBillLocked, nil)
			}
			// 与账单确认、取消保持“账单 -> 账单行 -> 费用”的锁顺序，避免并发事务互相等待。
			activeBill, queryErr = tx.FinanceBill.Query().Where(financebillent.IDEQ(lineSnapshot.BillID), financebillent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
			if queryErr != nil {
				return queryErr
			}
			activeLine, queryErr = tx.FinanceBillLine.Query().Where(financebilllineent.IDEQ(lineSnapshot.ID), financebilllineent.OrderFeeIDEQ(id), financebilllineent.ActiveEQ(true)).ForUpdate().Only(ctx)
			if queryErr != nil {
				return mapEntError(queryErr, biz.ErrBilledFeeBillLocked, nil)
			}
			item, queryErr = tx.OrderFee.Query().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID)).WithSettlementParty().ForUpdate().Only(ctx)
			if queryErr != nil {
				return queryErr
			}
		} else {
			item, queryErr = tx.OrderFee.Query().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID)).WithSettlementParty().ForUpdate().Only(ctx)
			if queryErr != nil {
				return queryErr
			}
		}
		if item.Version != input.Version {
			return biz.ErrOrderFeeVersionConflict
		}
		currentQuantity, quantityErr := decimalOf(item.Quantity)
		if quantityErr != nil {
			return quantityErr
		}
		if input.BillingUnitID == nil {
			return biz.ErrOrderFeeBillingUnitInvalid
		}
		if !currentQuantity.Equal(input.Quantity) || item.BillingUnitID == nil || *item.BillingUnitID != *input.BillingUnitID {
			unit, unitErr := tx.BillingUnit.Query().Where(billingunitent.IDEQ(*input.BillingUnitID)).ForShare().Only(ctx)
			if unitErr != nil {
				return mapEntError(unitErr, biz.ErrOrderFeeBillingUnitInvalid, nil)
			}
			if err := biz.ValidateFeeQuantityForUnit(input.Quantity, unit.QuantityMustBeInteger); err != nil {
				return err
			}
		}
		party, queryErr = tx.Partner.Query().Where(partnerent.IDEQ(input.SettlementPartyID), partnerent.OrganizationIDEQ(organizationID)).Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderFeePartyInvalid, nil)
		}
		if (item.Status != orderfeeent.StatusBILLED || item.SettlementPartyID != input.SettlementPartyID) && !party.Enabled {
			return biz.ErrOrderFeePartyInvalid
		}
		if item.Status != orderfeeent.StatusBILLED || item.Currency != input.Currency {
			validCurrency, currencyErr := tx.Currency.Query().Where(currencyent.CodeEQ(input.Currency), currencyent.EnabledEQ(true)).Exist(ctx)
			if currencyErr != nil {
				return currencyErr
			}
			if !validCurrency {
				return biz.ErrOrderFeeCurrencyInvalid
			}
		}
		if item.Status == orderfeeent.StatusBILLED {
			if activeBill == nil || activeLine == nil {
				return biz.ErrOrderFeeVersionConflict
			}
			if activeBill.Status != financebillent.StatusDRAFT {
				return biz.ErrBilledFeeBillLocked
			}
			setting, settingErr := tx.FinanceCustomSetting.Query().Where(financecustomsettingent.OrganizationIDEQ(organizationID)).ForShare().Only(ctx)
			if settingErr != nil {
				return mapEntError(settingErr, biz.ErrBilledFeeEditDisabled, nil)
			}
			if !setting.BilledFeeEditEnabled {
				return biz.ErrBilledFeeEditDisabled
			}
			current, convertErr := orderFeeToBiz(item)
			if convertErr != nil {
				return convertErr
			}
			if validateErr := biz.ValidateBilledFeeUpdate(current, input, financeCustomSettingToPolicy(setting)); validateErr != nil {
				return validateErr
			}
			if item.Currency != input.Currency && activeBill.FeeCount != 1 {
				return biz.ErrBilledFeeCurrencyConflict
			}
			if item.Currency != input.Currency && billExchangeRate == nil {
				return biz.ErrFinanceBillInvalidArgument
			}
		} else if item.Status != orderfeeent.StatusUNBILLED {
			return biz.ErrOrderFeeInvalidTransition
		}
		builder := tx.OrderFee.UpdateOne(item).
			SetDirection(orderfeeent.Direction(input.Direction)).
			SetNillableFeeSettingID(input.FeeSettingID).
			SetFeeCode(input.FeeCode).
			SetFeeName(input.FeeName).
			SetSettlementPartyID(input.SettlementPartyID).
			SetNillableBillingUnitID(input.BillingUnitID).
			SetBillingUnit(input.BillingUnit).
			SetQuantity(input.Quantity.StringFixed(4)).
			SetUnitPrice(input.UnitPrice.StringFixed(4)).
			SetTotalAmount(input.TotalAmount.StringFixed(8)).
			SetTaxInclusive(input.TaxInclusive).
			SetNetAmount(input.NetAmount.StringFixed(8)).
			SetTaxAmount(input.TaxAmount.StringFixed(8)).
			SetCurrency(input.Currency).
			SetExchangeRate(input.ExchangeRate.StringFixed(8)).
			SetExchangeRateSource(orderfeeent.ExchangeRateSource(input.ExchangeRateSource)).
			SetExchangeRateDate(input.ExchangeRateDate).
			SetBaseCurrency(input.BaseCurrency).
			SetBaseCurrencyAmount(input.BaseCurrencyAmount.StringFixed(8)).
			SetVersion(item.Version + 1).
			SetExpenseDate(input.ExpenseDate)
		if input.FeeNameEN != nil {
			builder.SetFeeNameEn(*input.FeeNameEN)
		} else {
			builder.ClearFeeNameEn()
		}
		if input.TaxRate != nil {
			builder.SetTaxRate(input.TaxRate.StringFixed(2))
		} else {
			builder.ClearTaxRate()
		}
		if input.TaxableServiceName != nil {
			builder.SetTaxableServiceName(*input.TaxableServiceName)
		} else {
			builder.ClearTaxableServiceName()
		}
		if input.ExchangeRateSettingID != nil {
			builder.SetExchangeRateSettingID(*input.ExchangeRateSettingID)
		} else {
			builder.ClearExchangeRateSettingID()
		}
		if input.Note != nil {
			builder.SetNote(*input.Note)
		} else {
			builder.ClearNote()
		}
		var updateErr error
		updated, updateErr = builder.Save(ctx)
		if updateErr != nil {
			return updateErr
		}
		if activeLine != nil {
			lineUpdate := tx.FinanceBillLine.UpdateOneID(activeLine.ID).
				SetFeeCode(input.FeeCode).SetFeeName(input.FeeName).
				SetQuantity(input.Quantity.StringFixed(4)).SetUnitPrice(input.UnitPrice.StringFixed(4)).
				SetTotalAmount(input.TotalAmount.StringFixed(8)).SetNetAmount(input.NetAmount.StringFixed(8)).SetTaxAmount(input.TaxAmount.StringFixed(8)).
				SetCurrency(input.Currency).SetExchangeRate(input.ExchangeRate.StringFixed(8)).SetBaseCurrencyAmount(input.BaseCurrencyAmount.StringFixed(8))
			if input.TaxRate == nil {
				lineUpdate.ClearTaxRate()
			} else {
				lineUpdate.SetTaxRate(input.TaxRate.StringFixed(4))
			}
			if _, updateErr := lineUpdate.Save(ctx); updateErr != nil {
				return updateErr
			}
			lines, lineErr := tx.FinanceBillLine.Query().Where(financebilllineent.BillIDEQ(activeBill.ID), financebilllineent.ActiveEQ(true)).All(ctx)
			if lineErr != nil {
				return lineErr
			}
			total, net, tax := decimal.Zero, decimal.Zero, decimal.Zero
			for _, line := range lines {
				lineTotal, parseErr := decimalOf(line.TotalAmount)
				if parseErr != nil {
					return parseErr
				}
				lineNet, parseErr := decimalOf(line.NetAmount)
				if parseErr != nil {
					return parseErr
				}
				lineTax, parseErr := decimalOf(line.TaxAmount)
				if parseErr != nil {
					return parseErr
				}
				total, net, tax = total.Add(lineTotal), net.Add(lineNet), tax.Add(lineTax)
			}
			billRate, parseErr := decimalOf(activeBill.ExchangeRate)
			if parseErr != nil {
				return parseErr
			}
			billUpdate := tx.FinanceBill.UpdateOneID(activeBill.ID).SetTotalAmount(total.StringFixed(8)).SetNetAmount(net.StringFixed(8)).SetTaxAmount(tax.StringFixed(8)).SetVersion(activeBill.Version + 1)
			if item.Currency != input.Currency {
				billRate = billExchangeRate.Rate
				// 币种变更后按账单当前业务日期固化的总部基准汇率重建账单汇率快照。
				billUpdate.SetCurrency(input.Currency).SetExchangeRate(billRate.StringFixed(8)).SetExchangeRateSource(financebillent.ExchangeRateSource(billExchangeRate.Source)).SetExchangeRateDate(activeBill.BillDate)
				billUpdate.ClearExchangeRateSettingID()
			}
			billUpdate.SetBaseCurrencyAmount(total.Mul(billRate).RoundBank(8).StringFixed(8))
			if _, updateErr := billUpdate.Save(ctx); updateErr != nil {
				return updateErr
			}
			audit.Details["bill.id"] = activeBill.ID.String()
			audit.Details["bill.previous_version"] = decimal.NewFromInt(int64(activeBill.Version)).String()
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	input.ID = id
	input.OrderID = orderID
	input.IdempotencyKey = item.IdempotencyKey
	input.Status = biz.OrderFeeStatus(item.Status)
	input.Version = updated.Version
	input.SettlementPartyName = party.LegalName
	input.CreatedAt = updated.CreatedAt
	input.UpdatedAt = updated.UpdatedAt
	return input, nil
}

// Remove 在事务内按账单占用关系物理删除费用：锁定订单与费用行并核对版本后，
// 复核存在未取消账单的活动关联行则拒绝（须先取消对应账单）；通过后删除费用
// 主记录。费用标签关联由外键级联清理；历史账单行的来源引用由外键置空，金额
// 与科目快照保留在账单行上。reason 仅写入审计明细。
func (r *orderFeeRepo) Remove(ctx context.Context, organizationID, orderID, id, actorID uuid.UUID, expectedVersion uint64, reason string, audit *biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if lockErr := lockOrderForFeeMutation(ctx, tx, organizationID, orderID); lockErr != nil {
			return lockErr
		}
		item, queryErr := tx.OrderFee.Query().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID)).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderFeeNotFound, nil)
		}
		if item.Version != expectedVersion {
			return biz.ErrOrderFeeVersionConflict
		}
		// 占用复核以事务内事实为准：存在活动账单行且所属账单未取消即占用。
		// 草稿账单同样视为已建立；已取消账单的历史行 active=false，不再阻断。
		// 与建账事务在费用行锁上串行：建账先提交则此处命中新活动行而拒绝；
		// 删除先提交则建账的费用存在性校验失败，二者互斥不产生悬挂引用。
		occupied, occupiedErr := tx.FinanceBillLine.Query().
			Where(
				financebilllineent.OrderFeeIDEQ(id),
				financebilllineent.ActiveEQ(true),
				financebilllineent.HasBillWith(financebillent.StatusNEQ(financebillent.StatusCANCELLED)),
			).
			Exist(ctx)
		if occupiedErr != nil {
			return occupiedErr
		}
		if occupied {
			return biz.ErrOrderFeeBillOccupied
		}
		if deleteErr := tx.OrderFee.DeleteOne(item).Exec(ctx); deleteErr != nil {
			return deleteErr
		}
		audit.Details["fee.code"] = item.FeeCode
		audit.Details["fee.direction"] = string(item.Direction)
		audit.Details["fee.amount"] = item.TotalAmount
		audit.Details["fee.currency"] = item.Currency
		audit.Details["fee.previous_status"] = string(item.Status)
		audit.Details["fee.previous_version"] = decimal.NewFromInt(int64(item.Version)).String()
		return writeAudit(ctx, tx.AuditLog, audit)
	})
}

// bulkOrderFeeTargetsDigest 提取批量目标的费用 ID 与版本映射。
func bulkOrderFeeTargetsDigest(targets []biz.OrderFeeBulkTarget) ([]uuid.UUID, map[uuid.UUID]uint64) {
	feeIDs := make([]uuid.UUID, 0, len(targets))
	versions := make(map[uuid.UUID]uint64, len(targets))
	for _, target := range targets {
		feeIDs = append(feeIDs, target.FeeID)
		versions[target.FeeID] = target.ExpectedVersion
	}
	return feeIDs, versions
}

// lockOrderFeesForBulk 按费用主键固定排序锁定订单内全部目标费用行（与建账的
// 费用锁同款锁序），并完成归属与版本校验；返回锁定行。任一行不满足即返回
// 携带费用定位的业务错误，调用方整批回滚。状态资格由调用方按操作语义校验。
func lockOrderFeesForBulk(ctx context.Context, tx *ent.Tx, orderID uuid.UUID, targets []biz.OrderFeeBulkTarget) ([]*ent.OrderFee, error) {
	feeIDs, versions := bulkOrderFeeTargetsDigest(targets)
	items, queryErr := tx.OrderFee.Query().
		Where(orderfeeent.OrderIDEQ(orderID), orderfeeent.IDIn(feeIDs...)).
		Order(orderfeeent.ByID()).
		ForUpdate().
		All(ctx)
	if queryErr != nil {
		return nil, queryErr
	}
	if len(items) != len(feeIDs) {
		found := make(map[uuid.UUID]struct{}, len(items))
		for _, item := range items {
			found[item.ID] = struct{}{}
		}
		for _, id := range feeIDs {
			if _, ok := found[id]; !ok {
				return nil, biz.BulkOrderFeeError(biz.ErrOrderFeeNotFound, id, "不存在或不属于该订单")
			}
		}
	}
	for _, item := range items {
		if item.Version != versions[item.ID] {
			return nil, biz.BulkOrderFeeError(biz.ErrOrderFeeVersionConflict, item.ID, "已被其他操作人修改，请刷新后重试")
		}
	}
	return items, nil
}

// ensureOrderFeeBulkUpdatable 校验批量定向修改的状态资格：仅未建账费用可改
// 结算单位/费用时间，与单条修改对已建账费用这两个字段的锁定口径一致。
func ensureOrderFeeBulkUpdatable(item *ent.OrderFee) error {
	switch item.Status {
	case orderfeeent.StatusUNBILLED:
		return nil
	case orderfeeent.StatusBILLED:
		return biz.BulkOrderFeeError(biz.ErrBilledFeeFieldForbidden, item.ID, "已进账单费用不可修改结算单位/费用时间")
	default:
		return biz.BulkOrderFeeError(biz.ErrOrderFeeInvalidTransition, item.ID, "当前状态不允许批量维护")
	}
}

// BulkUpdate 批量定向修改未建账费用：整批单一事务，先锁订单行，再按费用主键
// 固定排序锁全部目标行并完成全部校验，校验通过后才逐行定向写入并同事务写入
// 逐行审计；任一行失败整批回滚。仅更新目标字段：改结算单位不动金额与汇率；
// 改费用时间仅重写系统来源行的汇率快照与折本币（手工来源保持原汇率与来源），
// 禁止整 DTO 回写重算税率、名称与金额快照。
func (r *orderFeeRepo) BulkUpdate(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderFeeBulkUpdateInput, audits map[uuid.UUID]*biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if lockErr := lockOrderForFeeMutation(ctx, tx, organizationID, orderID); lockErr != nil {
			return lockErr
		}
		if input.SettlementPartyID != nil {
			// 结算单位法定名称不落费用行（经结算单位关联实时取数），此处仅复核
			// 档案存在、组织归属与启用状态，与单条修改同口径。
			if _, partyErr := tx.Partner.Query().
				Where(partnerent.IDEQ(*input.SettlementPartyID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).
				Only(ctx); partyErr != nil {
				return mapEntError(partyErr, biz.ErrOrderFeePartyInvalid, nil)
			}
		}
		items, lockErr := lockOrderFeesForBulk(ctx, tx, orderID, input.Targets)
		if lockErr != nil {
			return lockErr
		}
		for _, item := range items {
			if statusErr := ensureOrderFeeBulkUpdatable(item); statusErr != nil {
				return statusErr
			}
		}
		for _, item := range items {
			builder := tx.OrderFee.UpdateOne(item).SetVersion(item.Version + 1)
			if input.SettlementPartyID != nil {
				builder.SetSettlementPartyID(*input.SettlementPartyID)
			} else {
				newExpenseDate := *input.ExpenseDate
				builder.SetExpenseDate(newExpenseDate)
				if plan := input.RatePlans[item.ID]; plan != nil {
					builder.
						SetExchangeRate(plan.Rate.StringFixed(8)).
						SetExchangeRateSource(orderfeeent.ExchangeRateSource(plan.Source)).
						SetExchangeRateDate(plan.RateDate).
						SetBaseCurrencyAmount(plan.BaseAmount.StringFixed(8))
					if plan.SettingID != nil {
						builder.SetExchangeRateSettingID(*plan.SettingID)
					} else {
						builder.ClearExchangeRateSettingID()
					}
				} else {
					// 手工来源：汇率、来源与折本币保持原值，汇率日期同步新发生日期的日期部分。
					rateDate, _, _ := strings.Cut(newExpenseDate, " ")
					builder.SetExchangeRateDate(rateDate)
				}
			}
			if _, updateErr := builder.Save(ctx); updateErr != nil {
				return updateErr
			}
			if audit := audits[item.ID]; audit != nil {
				audit.Details["fee.previous_status"] = string(item.Status)
				audit.Details["fee.previous_version"] = decimal.NewFromInt(int64(item.Version)).String()
				audit.Details["fee.version"] = decimal.NewFromInt(int64(item.Version + 1)).String()
				audit.Details["fee.status"] = string(item.Status)
				if writeErr := writeAudit(ctx, tx.AuditLog, audit); writeErr != nil {
					return writeErr
				}
			}
		}
		return nil
	})
}

// BulkRemove 批量删除未建账费用：整批单一事务，锁订单行后按费用主键固定排序
// 锁全部目标行，逐行复核归属、版本、账单占用与状态（占用错误携带账单号便于
// 界面定位），全部通过后统一物理删除并逐行写审计；任一行失败整批回滚。费用
// 标签关联随删除级联清理，历史账单行的来源引用由外键置空，金额与科目快照
// 保留在账单行上。
func (r *orderFeeRepo) BulkRemove(ctx context.Context, organizationID, orderID uuid.UUID, targets []biz.OrderFeeBulkTarget, audits map[uuid.UUID]*biz.AuditEvent) error {
	return r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if lockErr := lockOrderForFeeMutation(ctx, tx, organizationID, orderID); lockErr != nil {
			return lockErr
		}
		items, lockErr := lockOrderFeesForBulk(ctx, tx, orderID, targets)
		if lockErr != nil {
			return lockErr
		}
		// 占用复核以事务内事实为准（与单条删除同款谓词）：存在活动账单行且所属
		// 账单未取消即占用；与建账事务在费用行锁上串行，二者互斥不产生悬挂引用。
		for _, item := range items {
			line, lineErr := tx.FinanceBillLine.Query().
				Where(
					financebilllineent.OrderFeeIDEQ(item.ID),
					financebilllineent.ActiveEQ(true),
					financebilllineent.HasBillWith(financebillent.StatusNEQ(financebillent.StatusCANCELLED)),
				).
				WithBill().
				First(ctx)
			if lineErr != nil && !ent.IsNotFound(lineErr) {
				return lineErr
			}
			if line != nil {
				bill, billErr := line.Edges.BillOrErr()
				if billErr != nil {
					return billErr
				}
				return biz.BulkOrderFeeError(biz.ErrOrderFeeBillOccupied, item.ID, fmt.Sprintf("已进入未取消的账单 %s，请先取消对应账单后再删除", bill.BillNo))
			}
			if item.Status != orderfeeent.StatusUNBILLED {
				return biz.BulkOrderFeeError(biz.ErrOrderFeeInvalidTransition, item.ID, "仅未建账费用可批量删除")
			}
		}
		for _, item := range items {
			if deleteErr := tx.OrderFee.DeleteOne(item).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
			if audit := audits[item.ID]; audit != nil {
				audit.Details["fee.code"] = item.FeeCode
				audit.Details["fee.direction"] = string(item.Direction)
				audit.Details["fee.amount"] = item.TotalAmount
				audit.Details["fee.currency"] = item.Currency
				audit.Details["fee.previous_status"] = string(item.Status)
				audit.Details["fee.previous_version"] = decimal.NewFromInt(int64(item.Version)).String()
				if writeErr := writeAudit(ctx, tx.AuditLog, audit); writeErr != nil {
					return writeErr
				}
			}
		}
		return nil
	})
}

func orderFeeToBiz(item *ent.OrderFee) (*biz.OrderFee, error) {
	party, err := item.Edges.SettlementPartyOrErr()
	if err != nil {
		return nil, err
	}
	quantity, err := decimalOf(item.Quantity)
	if err != nil {
		return nil, err
	}
	unitPrice, err := decimalOf(item.UnitPrice)
	if err != nil {
		return nil, err
	}
	totalAmount, err := decimalOf(item.TotalAmount)
	if err != nil {
		return nil, err
	}
	netAmount, err := decimalOf(item.NetAmount)
	if err != nil {
		return nil, err
	}
	taxAmount, err := decimalOf(item.TaxAmount)
	if err != nil {
		return nil, err
	}
	baseCurrencyAmount, err := decimalOf(item.BaseCurrencyAmount)
	if err != nil {
		return nil, err
	}
	exchangeRate, err := decimalOf(item.ExchangeRate)
	if err != nil {
		return nil, err
	}
	result := &biz.OrderFee{
		ID:                    item.ID,
		OrderID:               item.OrderID,
		IdempotencyKey:        item.IdempotencyKey,
		Direction:             biz.OrderFeeDirection(item.Direction),
		Status:                biz.OrderFeeStatus(item.Status),
		FeeSettingID:          item.FeeSettingID,
		FeeCode:               item.FeeCode,
		FeeName:               item.FeeName,
		FeeNameEN:             item.FeeNameEn,
		SettlementPartyID:     item.SettlementPartyID,
		SettlementPartyName:   party.LegalName,
		BillingUnitID:         item.BillingUnitID,
		BillingUnit:           item.BillingUnit,
		TaxableServiceName:    item.TaxableServiceName,
		Quantity:              quantity,
		UnitPrice:             unitPrice,
		TotalAmount:           totalAmount,
		TaxInclusive:          item.TaxInclusive,
		NetAmount:             netAmount,
		TaxAmount:             taxAmount,
		Currency:              item.Currency,
		ExchangeRate:          exchangeRate,
		ExchangeRateSource:    string(item.ExchangeRateSource),
		ExchangeRateDate:      item.ExchangeRateDate,
		ExchangeRateSettingID: item.ExchangeRateSettingID,
		BaseCurrency:          item.BaseCurrency,
		BaseCurrencyAmount:    baseCurrencyAmount,
		ExpenseDate:           item.ExpenseDate,
		Version:               item.Version,
		CancelledAt:           item.CancelledAt,
		CancelledBy:           item.CancelledBy,
		CancellationReason:    item.CancellationReason,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
	if item.TaxRate != nil {
		taxRate, err := decimalOf(*item.TaxRate)
		if err != nil {
			return nil, err
		}
		result.TaxRate = &taxRate
	}
	if item.Note != "" {
		note := item.Note
		result.Note = &note
	}
	return result, nil
}

func decimalPointerToString(value *decimal.Decimal, scale int32) *string {
	if value == nil {
		return nil
	}
	text := value.StringFixed(scale)
	return &text
}

var _ biz.OrderFeeRepo = (*orderFeeRepo)(nil)
