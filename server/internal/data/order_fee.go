package data

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financebilllineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderabnormalcaseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderabnormalcase"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	"github.com/shopspring/decimal"
)

type orderFeeRepo struct {
	data *Data
}

type orderFeeApplicability struct {
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
		Where(orderfeeent.OrderIDEQ(orderID), currentOrderFeePredicate()).
		WithSettlementParty().
		WithFinanceBillLines(effectiveBillLineFilter).
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
	o, err := client.Order.Get(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if err := attachFeeCapabilities(ctx, client, o, result); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *orderFeeRepo) Get(ctx context.Context, organizationID, orderID, id uuid.UUID) (*biz.OrderFee, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	item, err := client.OrderFee.Query().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID), orderfeeent.HasOrderWith(orderent.OrganizationIDEQ(organizationID))).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderFeeNotFound, nil)
	}
	fee, err := orderFeeToBiz(item)
	if err != nil {
		return nil, err
	}
	o, err := client.Order.Get(ctx, orderID)
	if err != nil {
		return nil, err
	}
	if err := attachFeeCapabilities(ctx, client, o, []*biz.OrderFee{fee}); err != nil {
		return nil, err
	}
	return fee, nil
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
		WithFinanceBillLines(effectiveBillLineFilter).
		Only(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return orderFeeToBiz(item)
}

// OrderETDDate 返回订单 ETD 原始字符串（RFC3339）；订单不存在时报不存在。
func (r *orderFeeRepo) OrderETDDate(ctx context.Context, organizationID, orderID uuid.UUID) (string, error) {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return "", err
	}
	client, err := r.data.client(ctx)
	if err != nil {
		return "", err
	}
	item, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return "", mapEntError(err, biz.ErrOrderFeeNotFound, nil)
	}
	return item.Etd, nil
}

func (r *orderFeeRepo) Options(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeOptions, error) {
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
	for _, party := range parties {
		result.SettlementParties = append(result.SettlementParties, biz.OrderFeeSettlementPartyOption{ID: party.ID, Code: partnerCodeValue(party.Code), Name: party.LegalName})
	}
	enabledCurrencies := make(map[string]struct{}, len(currencies))
	for _, currency := range currencies {
		enabledCurrencies[currency.Code] = struct{}{}
		result.Currencies = append(result.Currencies, biz.OrderFeeCurrencyOption{Code: currency.Code, Name: currency.Name, MinorUnit: currency.MinorUnit})
	}
	for _, billingUnit := range billingUnits {
		result.BillingUnits = append(result.BillingUnits, biz.OrderFeeBillingUnitOption{ID: billingUnit.ID, Code: billingUnit.Code, Name: billingUnit.Name, QuantityMustBeInteger: billingUnit.QuantityMustBeInteger, IsContainerUnit: billingUnit.IsContainerUnit})
	}
	for _, feeSetting := range feeSettings {
		billingUnit, billingErr := feeSetting.Edges.BillingUnitOrErr()
		taxableService, taxableErr := feeSetting.Edges.TaxableServiceOrErr()
		_, currencyEnabled := enabledCurrencies[feeSetting.DefaultCurrency]
		if billingErr != nil || taxableErr != nil || !billingUnit.Enabled || !taxableService.Enabled || !currencyEnabled {
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

func lockOrderForFeeMutation(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID) error {
	order, err := lockOrdersForFeeEconomics(ctx, tx, organizationID, orderID)
	if err != nil {
		return err
	}
	if order.LockedAt != nil {
		return biz.ErrOrderAlreadyLocked
	}
	return ensureOrderBusinessEditable(ctx, tx, order)
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
	abnormalCases, err := client.OrderAbnormalCase.Query().Where(orderabnormalcaseent.OrderIDEQ(orderID), orderabnormalcaseent.StatusEQ(orderabnormalcaseent.StatusACTIVE)).All(ctx)
	if err != nil {
		return nil, err
	}
	result := &orderFeeApplicability{abnormalCaseIDs: make(map[uuid.UUID]struct{}, len(abnormalCases))}
	for _, abnormalCase := range abnormalCases {
		result.abnormalCaseIDs[abnormalCase.AbnormalCaseID] = struct{}{}
	}
	return result, nil
}

func feeSettingApplies(feeSetting *ent.FeeSetting, applicability *orderFeeApplicability) bool {
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
		if err := validateFeeQuantitySelection(ctx, tx.Client(), organizationID, orderID, input, unit); err != nil {
			return err
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
			SetNillableFeeSettingID(input.FeeSettingID).
			SetFeeCode(input.FeeCode).
			SetFeeName(input.FeeName).
			SetNillableFeeNameEn(input.FeeNameEN).
			SetSettlementPartyID(input.SettlementPartyID).
			SetNillableBillingUnitID(input.BillingUnitID).
			SetBillingUnit(input.BillingUnit).
			SetNillableTaxRate(decimalPointerToString(input.TaxRate, 2)).
			SetNillableTaxableServiceName(input.TaxableServiceName).
			SetQuantitySource(feeQuantitySelectionMap(input.QuantitySource)).
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
			SetNillableNote(input.Note).
			SetVersion(input.Version).
			Save(ctx)
		if createErr != nil {
			return mapEntConstraint(createErr, "orderfee_order_id_idempotency_key", biz.ErrOrderFeeIdempotencyConflict)
		}
		if err := tx.Order.UpdateOneID(orderID).AddFeeContentVersion(1).Exec(ctx); err != nil {
			return err
		}
		attachFeeQuantityAudit(audit, input.QuantitySource)
		if err := reconcileOrdinaryFeeEconomics(ctx, tx.Client(), organizationID, orderID, audit); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	input.HasActiveBill = false // 刚创建的费用不可能存在账单关联
	input.SettlementPartyName = partyLegalName
	input.OrderID = orderID
	input.CreatedAt = created.CreatedAt
	input.UpdatedAt = created.UpdatedAt
	return input, nil
}

func (r *orderFeeRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderFee, billExchangeRate *biz.ResolvedRate, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	var item *ent.OrderFee
	var activeLine *ent.FinanceBillLine
	var party *ent.Partner
	var updated *ent.OrderFee
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		if lockErr := lockOrderForFeeMutation(ctx, tx, organizationID, orderID); lockErr != nil {
			return lockErr
		}
		var queryErr error

		item, queryErr = tx.OrderFee.Query().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).ForUpdate().Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderFeeNotFound, nil)
		}
		if len(item.Edges.FinanceBillLines) > 0 {
			return biz.ErrOrderFeeBillOccupied
		}
		if item.SupersededByRequestID != nil || item.EconomicEffect == orderfeeent.EconomicEffectDECREASE {
			return biz.ErrFeeCorrectionProtected
		}
		if item.Version != input.Version {
			return biz.ErrOrderFeeVersionConflict
		}
		if input.BillingUnitID == nil {
			return biz.ErrOrderFeeBillingUnitInvalid
		}
		unit, unitErr := tx.BillingUnit.Query().Where(billingunitent.IDEQ(*input.BillingUnitID)).ForShare().Only(ctx)
		if unitErr != nil {
			return mapEntError(unitErr, biz.ErrOrderFeeBillingUnitInvalid, nil)
		}
		existingSelection, selectionErr := feeQuantitySelectionFromMap(item.QuantitySource)
		if selectionErr != nil {
			return selectionErr
		}
		existingQuantity, quantityParseErr := decimalOf(item.Quantity)
		if quantityParseErr != nil {
			return quantityParseErr
		}
		// 同一费用未重新采用数量时保留历史快照，备注等私有编辑不冒充重新选源。
		if existingQuantity.Equal(input.Quantity) && item.BillingUnitID != nil && *item.BillingUnitID == *input.BillingUnitID && biz.FeeQuantitySelectionsEqual(existingSelection, input.QuantitySource) {
			input.QuantitySource = existingSelection
		} else if err := validateFeeQuantitySelection(ctx, tx.Client(), organizationID, orderID, input, unit); err != nil {
			return err
		}
		if !existingQuantity.Equal(input.Quantity) || item.BillingUnitID == nil || *item.BillingUnitID != *input.BillingUnitID {
			if err := biz.ValidateFeeQuantityForUnit(input.Quantity, unit.QuantityMustBeInteger); err != nil {
				return err
			}
		}
		party, queryErr = tx.Partner.Query().Where(partnerent.IDEQ(input.SettlementPartyID), partnerent.OrganizationIDEQ(organizationID)).Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderFeePartyInvalid, nil)
		}
		if (activeLine == nil || item.SettlementPartyID != input.SettlementPartyID) && !party.Enabled {
			return biz.ErrOrderFeePartyInvalid
		}
		if activeLine == nil || item.Currency != input.Currency {
			validCurrency, currencyErr := tx.Currency.Query().Where(currencyent.CodeEQ(input.Currency), currencyent.EnabledEQ(true)).Exist(ctx)
			if currencyErr != nil {
				return currencyErr
			}
			if !validCurrency {
				return biz.ErrOrderFeeCurrencyInvalid
			}
		}

		builder := tx.OrderFee.UpdateOne(item).
			SetDirection(orderfeeent.Direction(input.Direction)).
			SetNillableFeeSettingID(input.FeeSettingID).
			SetFeeCode(input.FeeCode).
			SetFeeName(input.FeeName).
			SetSettlementPartyID(input.SettlementPartyID).
			SetNillableBillingUnitID(input.BillingUnitID).
			SetBillingUnit(input.BillingUnit).
			SetQuantitySource(feeQuantitySelectionMap(input.QuantitySource)).
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
			SetVersion(item.Version + 1)
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

		if err := tx.Order.UpdateOneID(orderID).AddFeeContentVersion(1).Exec(ctx); err != nil {
			return err
		}
		attachFeeQuantityAudit(audit, input.QuantitySource)
		if err := reconcileOrdinaryFeeEconomics(ctx, tx.Client(), organizationID, orderID, audit); err != nil {
			return err
		}
		return writeAudit(ctx, tx.AuditLog, audit)
	})
	if err != nil {
		return nil, err
	}
	input.ID = id
	input.OrderID = orderID
	input.IdempotencyKey = item.IdempotencyKey
	input.HasActiveBill = activeLine != nil
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
		// 被替代的原费与内部冲回属于历史记录，当前正式费用按统一门禁处理。
		if item.SupersededByRequestID != nil || item.EconomicEffect == orderfeeent.EconomicEffectDECREASE {
			return biz.ErrFeeCorrectionProtected
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
		// 审计快照从已锁定费用行与同事务单位事实读取：费用名称取费用保存的名称，
		// 结算单位名称为删除时读取的当前法定名称（不声称是费用创建时的历史名称）。
		party, partyErr := tx.Partner.Query().
			Where(partnerent.IDEQ(item.SettlementPartyID), partnerent.OrganizationIDEQ(organizationID)).
			Only(ctx)
		if partyErr != nil {
			return partyErr
		}
		audit.Details["fee.code"] = item.FeeCode
		audit.Details["fee.name"] = item.FeeName
		audit.Details["fee.direction"] = string(item.Direction)
		audit.Details["fee.amount"] = item.TotalAmount
		audit.Details["fee.currency"] = item.Currency
		audit.Details["fee.settlement_party_id"] = item.SettlementPartyID.String()
		audit.Details["fee.settlement_party_name"] = party.LegalName
		audit.Details["fee.previous_version"] = decimal.NewFromInt(int64(item.Version)).String()
		if err := tx.Order.UpdateOneID(orderID).AddFeeContentVersion(1).Exec(ctx); err != nil {
			return err
		}
		if err := reconcileOrdinaryFeeEconomics(ctx, tx.Client(), organizationID, orderID, audit); err != nil {
			return err
		}
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
		if item.SupersededByRequestID != nil || item.EconomicEffect == orderfeeent.EconomicEffectDECREASE {
			return nil, biz.ErrFeeCorrectionProtected
		}
		if item.Version != versions[item.ID] {
			return nil, biz.BulkOrderFeeError(biz.ErrOrderFeeVersionConflict, item.ID, "已被其他操作人修改，请刷新后重试")
		}
	}
	return items, nil
}

// feeOccupiedSet 一次批量查询返回给定费用中存在有效账单关联的集合（活动账单行
// 且所属账单未取消），供批量操作逐行准入复核，避免逐费用 N+1 查询。
func feeOccupiedSet(ctx context.Context, tx *ent.Tx, feeIDs []uuid.UUID) (map[uuid.UUID]struct{}, error) {
	lines, err := tx.FinanceBillLine.Query().
		Where(financebilllineent.OrderFeeIDIn(feeIDs...), effectiveBillLinePredicate()).
		All(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[uuid.UUID]struct{}, len(lines))
	for _, line := range lines {
		result[line.OrderFeeID] = struct{}{}
	}
	return result, nil
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
		feeIDs := make([]uuid.UUID, 0, len(items))
		for _, item := range items {
			feeIDs = append(feeIDs, item.ID)
		}
		occupied, occupiedErr := feeOccupiedSet(ctx, tx, feeIDs)
		if occupiedErr != nil {
			return occupiedErr
		}
		for _, item := range items {
			if _, isBilled := occupied[item.ID]; isBilled {
				return biz.BulkOrderFeeError(biz.ErrBilledFeeFieldForbidden, item.ID, "已建账费用不可修改结算单位")
			}
		}
		for _, item := range items {
			builder := tx.OrderFee.UpdateOne(item).SetVersion(item.Version + 1)
			builder.SetSettlementPartyID(*input.SettlementPartyID)
			if _, updateErr := builder.Save(ctx); updateErr != nil {
				return updateErr
			}
			if audit := audits[item.ID]; audit != nil {
				audit.Details["fee.previous_version"] = decimal.NewFromInt(int64(item.Version)).String()
				audit.Details["fee.version"] = decimal.NewFromInt(int64(item.Version + 1)).String()
				if writeErr := writeAudit(ctx, tx.AuditLog, audit); writeErr != nil {
					return writeErr
				}
			}
		}
		if err := tx.Order.UpdateOneID(orderID).AddFeeContentVersion(1).Exec(ctx); err != nil {
			return err
		}
		var audit *biz.AuditEvent
		for _, candidate := range audits {
			audit = candidate
			break
		}
		return reconcileOrdinaryFeeEconomics(ctx, tx.Client(), organizationID, orderID, audit)
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
			if item.SupersededByRequestID != nil || item.EconomicEffect == orderfeeent.EconomicEffectDECREASE {
				return biz.ErrFeeCorrectionProtected
			}
		}
		// 批量解析结算单位名称快照：按单位 ID 去重后在组织边界内一次查询，避免
		// 逐费用 N+1；名称为删除事务内读取的当前法定名称（不声称是历史名称）。
		partyIDs := make([]uuid.UUID, 0, len(items))
		partySeen := make(map[uuid.UUID]struct{}, len(items))
		for _, item := range items {
			if _, exists := partySeen[item.SettlementPartyID]; !exists {
				partySeen[item.SettlementPartyID] = struct{}{}
				partyIDs = append(partyIDs, item.SettlementPartyID)
			}
		}
		parties, partyErr := tx.Partner.Query().
			Where(partnerent.OrganizationIDEQ(organizationID), partnerent.IDIn(partyIDs...)).
			All(ctx)
		if partyErr != nil {
			return partyErr
		}
		partyNames := make(map[uuid.UUID]string, len(parties))
		for _, party := range parties {
			partyNames[party.ID] = party.LegalName
		}
		for _, item := range items {
			if deleteErr := tx.OrderFee.DeleteOne(item).Exec(ctx); deleteErr != nil {
				return deleteErr
			}
			if audit := audits[item.ID]; audit != nil {
				audit.Details["fee.code"] = item.FeeCode
				audit.Details["fee.name"] = item.FeeName
				audit.Details["fee.direction"] = string(item.Direction)
				audit.Details["fee.amount"] = item.TotalAmount
				audit.Details["fee.currency"] = item.Currency
				audit.Details["fee.settlement_party_id"] = item.SettlementPartyID.String()
				audit.Details["fee.settlement_party_name"] = partyNames[item.SettlementPartyID]
				audit.Details["fee.previous_version"] = decimal.NewFromInt(int64(item.Version)).String()
				if writeErr := writeAudit(ctx, tx.AuditLog, audit); writeErr != nil {
					return writeErr
				}
			}
		}
		if err := tx.Order.UpdateOneID(orderID).AddFeeContentVersion(1).Exec(ctx); err != nil {
			return err
		}
		var audit *biz.AuditEvent
		for _, candidate := range audits {
			audit = candidate
			break
		}
		return reconcileOrdinaryFeeEconomics(ctx, tx.Client(), organizationID, orderID, audit)
	})
}

func orderFeeToBiz(item *ent.OrderFee) (*biz.OrderFee, error) {
	party, err := item.Edges.SettlementPartyOrErr()
	if err != nil {
		return nil, err
	}
	// 有效账单关联必须预加载后投影：未装配关联时显式失败，禁止把未加载事实
	// 静默当作未建账（调用方经 effectiveBillLineFilter 批量装配）。
	billLines, err := item.Edges.FinanceBillLinesOrErr()
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
	quantitySource, err := feeQuantitySelectionFromMap(item.QuantitySource)
	if err != nil {
		return nil, err
	}
	result := &biz.OrderFee{
		ID:                  item.ID,
		SupplementRequestID: item.SupplementRequestID,
		EconomicEffect:      biz.OrderFeeEconomicEffect(item.EconomicEffect), AccountingRequired: item.AccountingRequired, SourceFeeID: item.SourceFeeID, CorrectionRequestID: item.CorrectionRequestID, SupersededByRequestID: item.SupersededByRequestID,
		OrderID:               item.OrderID,
		IdempotencyKey:        item.IdempotencyKey,
		Direction:             biz.OrderFeeDirection(item.Direction),
		HasActiveBill:         len(billLines) > 0,
		FeeSettingID:          item.FeeSettingID,
		FeeCode:               item.FeeCode,
		FeeName:               item.FeeName,
		FeeNameEN:             item.FeeNameEn,
		SettlementPartyID:     item.SettlementPartyID,
		SettlementPartyName:   party.LegalName,
		BillingUnitID:         item.BillingUnitID,
		BillingUnit:           item.BillingUnit,
		TaxableServiceName:    item.TaxableServiceName,
		QuantitySource:        quantitySource,
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
		Version:               item.Version,
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
