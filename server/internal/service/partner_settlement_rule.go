package service

import (
	"context"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"

	"github.com/google/uuid"
)

func (s *PartnerService) ListPartnerSettlementRules(ctx context.Context, request *v1.ListPartnerSettlementRulesRequest) (*v1.ListPartnerSettlementRulesResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	return okList(ctx, &v1.ListPartnerSettlementRulesResponse{Data: data}), nil
}

func (s *PartnerService) CreatePartnerSettlementRule(ctx context.Context, request *v1.CreatePartnerSettlementRuleRequest) (*v1.CreatePartnerSettlementRuleResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	return ok(ctx, &v1.CreatePartnerSettlementRuleResponse{Data: partnerSettlementRuleToAPI(created)}), nil
}

func (s *PartnerService) UpdatePartnerSettlementRule(ctx context.Context, request *v1.UpdatePartnerSettlementRuleRequest) (*v1.UpdatePartnerSettlementRuleResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
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
	return ok(ctx, &v1.UpdatePartnerSettlementRuleResponse{Data: partnerSettlementRuleToAPI(updated)}), nil
}

func partnerSettlementRuleFromAPI(value *v1.PartnerSettlementRuleInput) *biz.PartnerSettlementRule {
	if value == nil {
		return nil
	}
	result := &biz.PartnerSettlementRule{}
	if value.CreditLimitMinor != nil {
		item := value.GetCreditLimitMinor()
		result.CreditLimitMinor = &item
	}
	if value.CreditCurrency != nil {
		item := value.GetCreditCurrency()
		result.CreditCurrency = &item
	}
	if value.PaymentTermsDays != nil {
		item := int(value.GetPaymentTermsDays())
		result.PaymentTermsDays = &item
	}
	return result
}

func partnerSettlementRuleToAPI(value *biz.PartnerSettlementRule) *v1.PartnerSettlementRule {
	if value == nil {
		return nil
	}
	result := &v1.PartnerSettlementRule{
		Id: value.ID.String(), PartnerRoleId: value.PartnerRoleID.String(),
	}
	result.CreditLimitMinor = value.CreditLimitMinor
	result.CreditCurrency = value.CreditCurrency
	result.PaymentTermsDays = int32PtrFromInt(value.PaymentTermsDays)
	return result
}

func int32PtrFromInt(value *int) *int32 {
	if value == nil {
		return nil
	}
	item := int32(*value)
	return &item
}
