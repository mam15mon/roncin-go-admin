package data

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
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
	exists, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrOrderFeeNotFound
	}
	return nil
}

func (r *orderFeeRepo) settlementParty(ctx context.Context, organizationID, partyID uuid.UUID) (*ent.Partner, error) {
	item, err := r.data.db.Partner.Query().Where(partnerent.IDEQ(partyID), partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeePartyInvalid, nil)
	}
	return item, nil
}

func (r *orderFeeRepo) validateCurrency(ctx context.Context, code string) error {
	exists, err := r.data.db.Currency.Query().Where(currencyent.CodeEQ(code), currencyent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return err
	}
	if !exists {
		return biz.ErrOrderFeeCurrencyInvalid
	}
	return nil
}

func (r *orderFeeRepo) List(ctx context.Context, organizationID, orderID uuid.UUID) ([]*biz.OrderFee, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	items, err := r.data.db.OrderFee.Query().
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
	item, err := r.data.db.OrderFee.Query().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(organizationID))).WithSettlementParty().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeeNotFound, nil)
	}
	return orderFeeToBiz(item)
}

func (r *orderFeeRepo) BilledBillContext(ctx context.Context, organizationID, orderID, id uuid.UUID) (*biz.BilledFeeBillContext, error) {
	line, err := r.data.db.FinanceBillLine.Query().Where(financebilllineent.OrderFeeIDEQ(id), financebilllineent.ActiveEQ(true), financebilllineent.OrderIDEQ(orderID)).WithBill(func(query *ent.FinanceBillQuery) {
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
	item, err := r.data.db.OrderFee.Query().
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

func (r *orderFeeRepo) ExchangeRateContext(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeExchangeRateContext, error) {
	item, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeeNotFound, nil)
	}
	return &biz.OrderFeeExchangeRateContext{
		TradeDirection: biz.OrderTradeDirection(item.TradeDirection),
		ETD:            item.Etd, ETA: item.Eta, BusinessTime: item.OrderDate, OrderCreatedAt: item.CreatedAt,
	}, nil
}

func (r *orderFeeRepo) Options(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeOptions, error) {
	applicability, err := r.loadApplicability(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	businessOrder, err := r.data.db.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).WithCustomer().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeeNotFound, nil)
	}
	customer, err := businessOrder.Edges.CustomerOrErr()
	if err != nil {
		return nil, err
	}
	headquartersID, err := resolveHeadquartersOrganizationID(ctx, r.data.db.Organization, organizationID)
	if err != nil {
		return nil, err
	}
	parties, err := r.data.db.Partner.Query().
		Where(partnerent.OrganizationIDEQ(organizationID), partnerent.EnabledEQ(true)).
		Order(partnerent.ByLegalName(), partnerent.ByCode()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	currencies, err := r.data.db.Currency.Query().
		Where(currencyent.EnabledEQ(true)).
		Order(currencyent.ByCode()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	billingUnits, err := r.data.db.BillingUnit.Query().
		Where(billingunitent.OrganizationIDEQ(headquartersID), billingunitent.EnabledEQ(true)).
		Order(billingunitent.BySortOrder(), billingunitent.ByCode(), billingunitent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	feeSettings, err := r.data.db.FeeSetting.Query().
		Where(feesettingent.OrganizationIDEQ(headquartersID), feesettingent.EnabledEQ(true)).
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
		result.SettlementParties = append(result.SettlementParties, biz.OrderFeeSettlementPartyOption{ID: party.ID, Code: party.Code, Name: party.LegalName})
	}
	enabledCurrencies := make(map[string]struct{}, len(currencies))
	for _, currency := range currencies {
		enabledCurrencies[currency.Code] = struct{}{}
		result.Currencies = append(result.Currencies, biz.OrderFeeCurrencyOption{Code: currency.Code, Name: currency.Name, MinorUnit: currency.MinorUnit})
	}
	for _, billingUnit := range billingUnits {
		result.BillingUnits = append(result.BillingUnits, biz.OrderFeeBillingUnitOption{ID: billingUnit.ID, Code: billingUnit.Code, Name: billingUnit.Name})
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

func (r *orderFeeRepo) financeLockCommissionNos(ctx context.Context, organizationID, orderID uuid.UUID) ([]string, error) {
	items, err := r.data.db.FinanceCommissionLine.Query().Where(
		commissionlineent.OrganizationIDEQ(organizationID),
		commissionlineent.OrderIDEQ(orderID),
		commissionlineent.HasCommissionWith(commissionent.StatusIn(commissionent.StatusCONFIRMED, commissionent.StatusPAID)),
	).WithCommission().All(ctx)
	if err != nil {
		return nil, err
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
	locked, err := tx.FinanceCommissionLine.Query().Where(
		commissionlineent.OrganizationIDEQ(organizationID),
		commissionlineent.OrderIDEQ(orderID),
		commissionlineent.HasCommissionWith(commissionent.StatusIn(commissionent.StatusCONFIRMED, commissionent.StatusPAID)),
	).Exist(ctx)
	if err != nil {
		return err
	}
	if locked {
		return biz.ErrOrderFeeFinanceLocked
	}
	return nil
}

func (r *orderFeeRepo) ResolveCatalog(ctx context.Context, organizationID, orderID, feeSettingID, billingUnitID uuid.UUID) (*biz.OrderFeeCatalogSnapshot, error) {
	applicability, err := r.loadApplicability(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	headquartersID, err := resolveHeadquartersOrganizationID(ctx, r.data.db.Organization, organizationID)
	if err != nil {
		return nil, err
	}
	feeSetting, err := r.data.db.FeeSetting.Query().
		Where(feesettingent.IDEQ(feeSettingID), feesettingent.OrganizationIDEQ(headquartersID), feesettingent.EnabledEQ(true)).
		WithBillingUnit().WithTaxableService().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeeSettingInvalid, nil)
	}
	defaultBillingUnit, billingErr := feeSetting.Edges.BillingUnitOrErr()
	taxableService, taxableErr := feeSetting.Edges.TaxableServiceOrErr()
	defaultCurrencyEnabled, err := r.data.db.Currency.Query().Where(currencyent.CodeEQ(feeSetting.DefaultCurrency), currencyent.EnabledEQ(true)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	if billingErr != nil || taxableErr != nil || !defaultBillingUnit.Enabled || !taxableService.Enabled || !defaultCurrencyEnabled || !feeSettingApplies(feeSetting, applicability) {
		return nil, biz.ErrOrderFeeSettingInvalid
	}
	billingUnit, err := r.data.db.BillingUnit.Query().
		Where(billingunitent.IDEQ(billingUnitID), billingunitent.OrganizationIDEQ(headquartersID), billingunitent.EnabledEQ(true)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeeBillingUnitInvalid, nil)
	}
	taxRate, err := decimalOf(feeSetting.TaxRate)
	if err != nil {
		return nil, err
	}
	return &biz.OrderFeeCatalogSnapshot{
		FeeCode: feeSetting.FeeCode, FeeName: feeSetting.NameZh, FeeNameEN: feeSetting.NameEn,
		BillingUnit: billingUnit.Name, TaxRate: taxRate, TaxableServiceName: taxableService.Name,
	}, nil
}

func (r *orderFeeRepo) loadApplicability(ctx context.Context, organizationID, orderID uuid.UUID) (*orderFeeApplicability, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	serviceTypes, err := r.data.db.OrderServiceType.Query().Where(orderservicetypeent.OrderIDEQ(orderID)).All(ctx)
	if err != nil {
		return nil, err
	}
	abnormalCases, err := r.data.db.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(orderID), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).All(ctx)
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
	if feeSetting.ServiceTypeID != nil {
		if _, ok := applicability.serviceTypeIDs[*feeSetting.ServiceTypeID]; !ok {
			return false
		}
	}
	if feeSetting.AbnormalCaseID != nil {
		if _, ok := applicability.abnormalCaseIDs[*feeSetting.AbnormalCaseID]; !ok {
			return false
		}
	}
	return true
}

func (r *orderFeeRepo) Add(ctx context.Context, organizationID, orderID uuid.UUID, input *biz.OrderFee, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	party, err := r.settlementParty(ctx, organizationID, input.SettlementPartyID)
	if err != nil {
		return nil, err
	}
	if err := r.validateCurrency(ctx, input.Currency); err != nil {
		return nil, err
	}
	var created *ent.OrderFee
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if lockErr := lockOrderForFeeMutation(ctx, tx, organizationID, orderID); lockErr != nil {
			return lockErr
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
	input.SettlementPartyName = party.LegalName
	input.OrderID = orderID
	input.CreatedAt = created.CreatedAt
	input.UpdatedAt = created.UpdatedAt
	return input, nil
}

func (r *orderFeeRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderFee, billExchangeRate *biz.ResolvedExchangeRate, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
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
			ownerID, ownerErr := resolveHeadquartersOrganizationID(ctx, tx.Organization, organizationID)
			if ownerErr != nil {
				return ownerErr
			}
			setting, settingErr := tx.FinanceCustomSetting.Query().Where(financecustomsettingent.OrganizationIDEQ(ownerID)).ForShare().Only(ctx)
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
			if item.Currency != input.Currency && (billExchangeRate == nil || billExchangeRate.RateDate != activeBill.BillDate) {
				return biz.ErrFinanceBillInvalidArgument
			}
		} else if item.Status != orderfeeent.StatusDRAFT {
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
				billUpdate.SetCurrency(input.Currency).SetExchangeRate(billRate.StringFixed(8)).SetExchangeRateSource(financebillent.ExchangeRateSource(billExchangeRate.Source)).SetExchangeRateDate(billExchangeRate.RateDate)
				if billExchangeRate.SettingID == nil {
					billUpdate.ClearExchangeRateSettingID()
				} else {
					billUpdate.SetExchangeRateSettingID(*billExchangeRate.SettingID)
				}
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

func (r *orderFeeRepo) Transition(ctx context.Context, organizationID, orderID, id, actorID uuid.UUID, expectedVersion uint64, from, to biz.OrderFeeStatus, reason *string, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return nil, err
	}
	var updated *ent.OrderFee
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
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
		if item.Status != orderfeeent.Status(from) {
			return biz.ErrOrderFeeInvalidTransition
		}
		builder := tx.OrderFee.UpdateOne(item).SetStatus(orderfeeent.Status(to)).SetVersion(item.Version + 1)
		if to != biz.OrderFeeCancelled {
			builder.ClearCancelledAt().ClearCancelledBy().ClearCancellationReason()
		}
		var updateErr error
		updated, updateErr = builder.Save(ctx)
		if updateErr != nil {
			return updateErr
		}
		audit.Details["fee.from_status"] = string(from)
		audit.Details["fee.to_status"] = string(to)
		audit.Details["fee.previous_version"] = decimal.NewFromInt(int64(expectedVersion)).String()
		if reason != nil {
			audit.Details["reason"] = *reason
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	loaded, err := r.data.db.OrderFee.Query().Where(orderfeeent.IDEQ(updated.ID)).WithSettlementParty().Only(ctx)
	if err != nil {
		return nil, err
	}
	return orderFeeToBiz(loaded)
}

func (r *orderFeeRepo) Remove(ctx context.Context, organizationID, orderID, id, actorID uuid.UUID, expectedVersion uint64, reason string, audit *biz.AuditEvent) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
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
		if item.Status != orderfeeent.StatusDRAFT && item.Status != orderfeeent.StatusCONFIRMED {
			return biz.ErrOrderFeeInvalidTransition
		}
		now := time.Now().UTC()
		if _, updateErr := tx.OrderFee.UpdateOne(item).
			SetStatus(orderfeeent.StatusCANCELLED).
			SetVersion(item.Version + 1).
			SetCancelledAt(now).
			SetCancelledBy(actorID).
			SetCancellationReason(reason).
			Save(ctx); updateErr != nil {
			return updateErr
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
