package service

import (
	"context"
	"time"

	v1 "github.com/roncin/roncin-go-admin/server/api/partner/v1"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/platform/requestmeta"

	"github.com/google/uuid"
)

func (s *PartnerService) ListPartnerContracts(ctx context.Context, request *v1.ListPartnerContractsRequest) (*v1.ListPartnerContractsResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil {
		return nil, biz.ErrPartnerContractInvalidArgument
	}
	var status *biz.PartnerContractStatus
	if request.Status != nil {
		value := partnerContractStatusFromAPI(request.GetStatus())
		status = &value
	}
	items, err := s.contractUsecase.List(ctx, principal.Organization.ID, partnerID, status)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.PartnerContract, 0, len(items))
	for _, item := range items {
		data = append(data, partnerContractToAPI(item))
	}
	return &v1.ListPartnerContractsResponse{Success: true, Code: 0, Message: "OK", Data: data, TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) CreatePartnerContract(ctx context.Context, request *v1.CreatePartnerContractRequest) (*v1.CreatePartnerContractResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	partnerID, err := uuid.Parse(request.GetPartnerId())
	if err != nil || request.GetContract() == nil {
		return nil, biz.ErrPartnerContractInvalidArgument
	}
	input := request.GetContract()
	startDate, endDate, err := parseContractDates(input.GetStartDate(), input.GetEndDate())
	if err != nil {
		return nil, err
	}
	created, err := s.contractUsecase.Create(ctx, principal.Organization.ID, principal.UserID, partnerID, &biz.PartnerContract{
		ContractNo: input.GetContractNo(), Name: input.GetName(), Status: partnerContractStatusFromAPI(input.GetStatus()),
		StartDate: startDate, EndDate: endDate, PaymentTerms: input.GetPaymentTerms(), DisputeResolution: input.GetDisputeResolution(), OtherNotes: input.GetOtherNotes(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.CreatePartnerContractResponse{Success: true, Code: 0, Message: "OK", Data: partnerContractToAPI(created), TraceId: requestmeta.TraceID(ctx)}, nil
}

func (s *PartnerService) UpdatePartnerContract(ctx context.Context, request *v1.UpdatePartnerContractRequest) (*v1.UpdatePartnerContractResponse, error) {
	principal, principalErr := biz.RequirePrincipal(ctx)
	if principalErr != nil {
		return nil, principalErr
	}
	partnerID, partnerErr := uuid.Parse(request.GetPartnerId())
	id, idErr := uuid.Parse(request.GetId())
	if partnerErr != nil || idErr != nil || request.GetContract() == nil {
		return nil, biz.ErrPartnerContractInvalidArgument
	}
	input := request.GetContract()
	startDate, endDate, err := parseContractDates(input.GetStartDate(), input.GetEndDate())
	if err != nil {
		return nil, err
	}
	updated, err := s.contractUsecase.Update(ctx, principal.Organization.ID, principal.UserID, partnerID, id, &biz.PartnerContract{
		Name: input.GetName(), Status: partnerContractStatusFromAPI(input.GetStatus()),
		StartDate: startDate, EndDate: endDate, PaymentTerms: input.GetPaymentTerms(), DisputeResolution: input.GetDisputeResolution(), OtherNotes: input.GetOtherNotes(),
	})
	if err != nil {
		return nil, err
	}
	return &v1.UpdatePartnerContractResponse{Success: true, Code: 0, Message: "OK", Data: partnerContractToAPI(updated), TraceId: requestmeta.TraceID(ctx)}, nil
}

func parseContractDates(start, end string) (time.Time, time.Time, error) {
	startDate, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return time.Time{}, time.Time{}, biz.ErrPartnerContractInvalidArgument
	}
	endDate, err := time.Parse(time.RFC3339, end)
	if err != nil {
		return time.Time{}, time.Time{}, biz.ErrPartnerContractInvalidArgument
	}
	return startDate, endDate, nil
}

func partnerContractStatusFromAPI(value v1.PartnerContractStatus) biz.PartnerContractStatus {
	switch value {
	case v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_PENDING:
		return biz.PartnerContractPending
	case v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_ACTIVE:
		return biz.PartnerContractActive
	case v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_EXPIRED:
		return biz.PartnerContractExpired
	case v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_TERMINATED:
		return biz.PartnerContractTerminated
	default:
		return ""
	}
}

func partnerContractStatusToAPI(value biz.PartnerContractStatus) v1.PartnerContractStatus {
	switch value {
	case biz.PartnerContractPending:
		return v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_PENDING
	case biz.PartnerContractActive:
		return v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_ACTIVE
	case biz.PartnerContractExpired:
		return v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_EXPIRED
	case biz.PartnerContractTerminated:
		return v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_TERMINATED
	default:
		return v1.PartnerContractStatus_PARTNER_CONTRACT_STATUS_UNSPECIFIED
	}
}

func partnerContractToAPI(value *biz.PartnerContract) *v1.PartnerContract {
	return &v1.PartnerContract{
		Id: value.ID.String(), PartnerId: value.PartnerID.String(), ContractNo: value.ContractNo, Name: value.Name,
		Status: partnerContractStatusToAPI(value.Status), StartDate: value.StartDate.Format(time.RFC3339), EndDate: value.EndDate.Format(time.RFC3339),
		PaymentTerms: value.PaymentTerms, DisputeResolution: value.DisputeResolution, OtherNotes: value.OtherNotes,
		CreatedAt: value.CreatedAt.Format(time.RFC3339), UpdatedAt: value.UpdatedAt.Format(time.RFC3339), AllowedStatuses: partnerContractStatusesToAPI(value.AllowedStatuses()),
	}
}

func partnerContractStatusesToAPI(statuses []biz.PartnerContractStatus) []v1.PartnerContractStatus {
	result := make([]v1.PartnerContractStatus, 0, len(statuses))
	for _, status := range statuses {
		result = append(result, partnerContractStatusToAPI(status))
	}
	return result
}
