package service

import (
	"context"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

func (s *PartnerService) ListPartnerSettlementRules(ctx context.Context, request *v1.ListPartnerSettlementRulesRequest) (*v1.ListPartnerSettlementRulesResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	roleType := partnerRoleTypeFromAPI(request.GetRoleType())
	if err != nil || !roleType.Valid() {
		return nil, biz.ErrPartnerSettlementRuleInvalidArgument
	}
	items, err := s.settlementRuleUsecase.List(ctx, principal.Organization.ID, partnerID, roleType)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerSettlementRule, 0, len(items))
	for _, item := range items {
		data = append(data, partnerSettlementRuleToAPI(item))
	}
	return &v1.ListPartnerSettlementRulesResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) CreatePartnerSettlementRule(ctx context.Context, request *v1.CreatePartnerSettlementRuleRequest) (*v1.CreatePartnerSettlementRuleResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	roleType := partnerRoleTypeFromAPI(request.GetRoleType())
	if err != nil || !roleType.Valid() || request.GetRule() == nil {
		return nil, biz.ErrPartnerSettlementRuleInvalidArgument
	}
	created, err := s.settlementRuleUsecase.Create(ctx, principal.Organization.ID, principal.UserID, partnerID, roleType, partnerSettlementRuleFromAPI(request.GetRule()))
	if err != nil {
		return nil, err
	}
	return &v1.CreatePartnerSettlementRuleResponse{Success: true, Code: 0, Message: "OK", Data: partnerSettlementRuleToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) UpdatePartnerSettlementRule(ctx context.Context, request *v1.UpdatePartnerSettlementRuleRequest) (*v1.UpdatePartnerSettlementRuleResponse, error) {
	principal, ok := biz.PrincipalFromContext(ctx)
	if !ok {
		return nil, biz.ErrSessionRequired
	}
	partnerID, partnerErr := uuid.Parse(request.GetPartnerId())
	id, idErr := uuid.Parse(request.GetId())
	roleType := partnerRoleTypeFromAPI(request.GetRoleType())
	if partnerErr != nil || idErr != nil || !roleType.Valid() || request.GetRule() == nil {
		return nil, biz.ErrPartnerSettlementRuleInvalidArgument
	}
	updated, err := s.settlementRuleUsecase.Update(ctx, principal.Organization.ID, principal.UserID, partnerID, id, roleType, partnerSettlementRuleFromAPI(request.GetRule()))
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePartnerSettlementRuleResponse{Success: true, Code: 0, Message: "OK", Data: partnerSettlementRuleToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func partnerStatementModeFromAPI(value v1.PartnerStatementMode) biz.PartnerStatementMode {
	if value == v1.PartnerStatementMode_PARTNER_STATEMENT_MODE_SINGLE {
		return biz.PartnerStatementSingle
	}
	if value == v1.PartnerStatementMode_PARTNER_STATEMENT_MODE_MULTI {
		return biz.PartnerStatementMulti
	}
	return ""
}

func partnerStatementModeToAPI(value biz.PartnerStatementMode) v1.PartnerStatementMode {
	if value == biz.PartnerStatementSingle {
		return v1.PartnerStatementMode_PARTNER_STATEMENT_MODE_SINGLE
	}
	if value == biz.PartnerStatementMulti {
		return v1.PartnerStatementMode_PARTNER_STATEMENT_MODE_MULTI
	}
	return v1.PartnerStatementMode_PARTNER_STATEMENT_MODE_UNSPECIFIED
}

func partnerSettlementMethodFromAPI(value v1.PartnerSettlementMethod) biz.PartnerSettlementMethod {
	switch value {
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_BY_TICKET:
		return biz.PartnerSettlementByTicket
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_MONTHLY:
		return biz.PartnerSettlementMonthly
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_WEEKLY:
		return biz.PartnerSettlementWeekly
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_SEMI_MONTHLY:
		return biz.PartnerSettlementSemiMonthly
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_BI_MONTHLY:
		return biz.PartnerSettlementBiMonthly
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_QUARTERLY:
		return biz.PartnerSettlementQuarterly
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_DAYS_45:
		return biz.PartnerSettlementDays45
	case v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_PREPAID:
		return biz.PartnerSettlementPrepaid
	default:
		return ""
	}
}

func partnerSettlementMethodToAPI(value biz.PartnerSettlementMethod) v1.PartnerSettlementMethod {
	switch value {
	case biz.PartnerSettlementByTicket:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_BY_TICKET
	case biz.PartnerSettlementMonthly:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_MONTHLY
	case biz.PartnerSettlementWeekly:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_WEEKLY
	case biz.PartnerSettlementSemiMonthly:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_SEMI_MONTHLY
	case biz.PartnerSettlementBiMonthly:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_BI_MONTHLY
	case biz.PartnerSettlementQuarterly:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_QUARTERLY
	case biz.PartnerSettlementDays45:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_DAYS_45
	case biz.PartnerSettlementPrepaid:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_PREPAID
	default:
		return v1.PartnerSettlementMethod_PARTNER_SETTLEMENT_METHOD_UNSPECIFIED
	}
}

func partnerSettlementBaseFromAPI(value v1.PartnerSettlementBase) *biz.PartnerSettlementBase {
	var result biz.PartnerSettlementBase
	switch value {
	case v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_BILL_DATE:
		result = biz.PartnerSettlementBillDate
	case v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_SAILING_DATE:
		result = biz.PartnerSettlementSailingDate
	case v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_ARRIVAL_DATE:
		result = biz.PartnerSettlementArrivalDate
	default:
		return nil
	}
	return &result
}

func partnerSettlementBaseToAPI(value *biz.PartnerSettlementBase) *v1.PartnerSettlementBase {
	if value == nil {
		return nil
	}
	result := v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_UNSPECIFIED
	switch *value {
	case biz.PartnerSettlementBillDate:
		result = v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_BILL_DATE
	case biz.PartnerSettlementSailingDate:
		result = v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_SAILING_DATE
	case biz.PartnerSettlementArrivalDate:
		result = v1.PartnerSettlementBase_PARTNER_SETTLEMENT_BASE_ARRIVAL_DATE
	}
	return &result
}

func partnerSettlementRuleFromAPI(value *v1.PartnerSettlementRuleInput) *biz.PartnerSettlementRule {
	if value == nil {
		return nil
	}
	result := &biz.PartnerSettlementRule{
		StatementMode: partnerStatementModeFromAPI(value.GetStatementMode()), SettlementMethod: partnerSettlementMethodFromAPI(value.GetSettlementMethod()),
		SettlementCurrency: value.GetSettlementCurrency(), IsActive: value.GetIsActive(),
	}
	if value.SettlementDay != nil {
		item := int(value.GetSettlementDay())
		result.SettlementDay = &item
	}
	if value.SettlementCycleDays != nil {
		item := int(value.GetSettlementCycleDays())
		result.SettlementCycleDays = &item
	}
	if value.SettlementBase != nil {
		result.SettlementBase = partnerSettlementBaseFromAPI(value.GetSettlementBase())
	}
	if value.CreditLimitMinor != nil {
		item := value.GetCreditLimitMinor()
		result.CreditLimitMinor = &item
	}
	if value.CreditCurrency != nil {
		item := value.GetCreditCurrency()
		result.CreditCurrency = &item
	}
	return result
}

func partnerSettlementRuleToAPI(value *biz.PartnerSettlementRule) *v1.PartnerSettlementRule {
	if value == nil {
		return nil
	}
	result := &v1.PartnerSettlementRule{
		Id: value.ID.String(), PartnerRoleId: value.PartnerRoleID.String(), StatementMode: partnerStatementModeToAPI(value.StatementMode),
		SettlementMethod: partnerSettlementMethodToAPI(value.SettlementMethod), SettlementCurrency: value.SettlementCurrency, IsActive: value.IsActive,
	}
	if value.SettlementDay != nil {
		item := int32(*value.SettlementDay)
		result.SettlementDay = &item
	}
	if value.SettlementCycleDays != nil {
		item := int32(*value.SettlementCycleDays)
		result.SettlementCycleDays = &item
	}
	result.SettlementBase = partnerSettlementBaseToAPI(value.SettlementBase)
	result.CreditLimitMinor = value.CreditLimitMinor
	result.CreditCurrency = value.CreditCurrency
	return result
}
