package data

import (
	"context"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	currencyent "github.com/roncin/roncin-go-admin/server/internal/data/ent/currency"
	feesettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/feesetting"
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
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderFeePartyInvalid
		}
		return nil, err
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

func (r *orderFeeRepo) Options(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeOptions, error) {
	applicability, err := r.loadApplicability(ctx, organizationID, orderID)
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
		taxRate, parseErr := decimal.NewFromString(feeSetting.TaxRate)
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
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderFeeSettingInvalid
		}
		return nil, err
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
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderFeeBillingUnitInvalid
		}
		return nil, err
	}
	taxRate, err := decimal.NewFromString(feeSetting.TaxRate)
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
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	created, err := tx.OrderFee.Create().
		SetID(input.ID).
		SetOrderID(orderID).
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
		SetQuantity(input.Quantity.StringFixed(4)).
		SetUnitPrice(input.UnitPrice.StringFixed(4)).
		SetTotalAmount(input.TotalAmount.StringFixed(8)).
		SetCurrency(input.Currency).
		SetExchangeRate(input.ExchangeRate.StringFixed(8)).
		SetExchangeRateSource(orderfeeent.ExchangeRateSource(input.ExchangeRateSource)).
		SetExchangeRateDate(input.ExchangeRateDate).
		SetNillableExchangeRateSettingID(input.ExchangeRateSettingID).
		SetExpenseDate(input.ExpenseDate).
		SetNillableNote(input.Note).
		Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	input.SettlementPartyName = party.LegalName
	input.OrderID = orderID
	input.CreatedAt = created.CreatedAt
	input.UpdatedAt = created.UpdatedAt
	return input, nil
}

func (r *orderFeeRepo) Update(ctx context.Context, organizationID, orderID, id uuid.UUID, input *biz.OrderFee, audit *biz.AuditEvent) (*biz.OrderFee, error) {
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
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return nil, err
	}
	item, err := tx.OrderFee.Query().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID)).ForUpdate().Only(ctx)
	if err != nil {
		_ = tx.Rollback()
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderFeeNotFound
		}
		return nil, err
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
		SetCurrency(input.Currency).
		SetExchangeRate(input.ExchangeRate.StringFixed(8)).
		SetExchangeRateSource(orderfeeent.ExchangeRateSource(input.ExchangeRateSource)).
		SetExchangeRateDate(input.ExchangeRateDate).
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
	updated, err := builder.Save(ctx)
	if err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	input.ID = id
	input.OrderID = orderID
	input.SettlementPartyName = party.LegalName
	input.CreatedAt = updated.CreatedAt
	input.UpdatedAt = updated.UpdatedAt
	return input, nil
}

func (r *orderFeeRepo) Remove(ctx context.Context, organizationID, orderID, id uuid.UUID, audit *biz.AuditEvent) error {
	if err := r.order(ctx, organizationID, orderID); err != nil {
		return err
	}
	tx, err := r.data.db.Tx(ctx)
	if err != nil {
		return err
	}
	count, err := tx.OrderFee.Delete().Where(orderfeeent.IDEQ(id), orderfeeent.OrderIDEQ(orderID)).Exec(ctx)
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	if count == 0 {
		_ = tx.Rollback()
		return biz.ErrOrderFeeNotFound
	}
	if err := writeAudit(ctx, tx.AuditLog, audit); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

func orderFeeToBiz(item *ent.OrderFee) (*biz.OrderFee, error) {
	party, err := item.Edges.SettlementPartyOrErr()
	if err != nil {
		return nil, err
	}
	quantity, err := decimal.NewFromString(item.Quantity)
	if err != nil {
		return nil, err
	}
	unitPrice, err := decimal.NewFromString(item.UnitPrice)
	if err != nil {
		return nil, err
	}
	totalAmount, err := decimal.NewFromString(item.TotalAmount)
	if err != nil {
		return nil, err
	}
	exchangeRate, err := decimal.NewFromString(item.ExchangeRate)
	if err != nil {
		return nil, err
	}
	result := &biz.OrderFee{
		ID:                    item.ID,
		OrderID:               item.OrderID,
		Direction:             biz.OrderFeeDirection(item.Direction),
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
		Currency:              item.Currency,
		ExchangeRate:          exchangeRate,
		ExchangeRateSource:    string(item.ExchangeRateSource),
		ExchangeRateDate:      item.ExchangeRateDate,
		ExchangeRateSettingID: item.ExchangeRateSettingID,
		ExpenseDate:           item.ExpenseDate,
		CreatedAt:             item.CreatedAt,
		UpdatedAt:             item.UpdatedAt,
	}
	if item.TaxRate != nil {
		taxRate, err := decimal.NewFromString(*item.TaxRate)
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
