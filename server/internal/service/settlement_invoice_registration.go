package service

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func (s *SettlementService) RegisterInvoices(ctx context.Context, request *v1.RegisterInvoicesRequest) (*v1.RegisterInvoicesResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := organizationIDsForPermission(principal, access.FinanceInvoiceCreate, true)
	if err != nil {
		return nil, err
	}
	records, err := invoiceRegistrationInputsFromAPI(request.GetRecords())
	if err != nil {
		return nil, err
	}
	items, err := s.invoiceUsecase.RegisterScoped(ctx, scope, principal.UserID, biz.RegisterFinanceInvoicesInput{IdempotencyKey: request.GetIdempotencyKey(), Records: records})
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceInvoice, 0, len(items))
	for _, item := range items {
		data = append(data, financeInvoiceToAPI(item))
	}
	return ok(ctx, &v1.RegisterInvoicesResponse{Data: data}), nil
}
func (s *SettlementService) ReviseInvoiceRegistrations(ctx context.Context, request *v1.ReviseInvoiceRegistrationsRequest) (*v1.ReviseInvoiceRegistrationsResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := organizationIDsForPermission(principal, access.FinanceInvoiceUpdate, true)
	if err != nil {
		return nil, err
	}
	// 修订中的新增票据必须同时具备登记权限，不能借更新权限扩大写入。
	for _, entry := range request.GetEntries() {
		if entry != nil && strings.TrimSpace(entry.GetId()) == "" {
			createScope, err := organizationIDsForPermission(principal, access.FinanceInvoiceCreate, true)
			if err != nil {
				return nil, err
			}
			if len(createScope) != 1 || len(scope) != 1 || createScope[0] != scope[0] {
				return nil, biz.ErrFinanceInvoiceInvalidArgument
			}
		}
	}
	input := biz.ReviseFinanceInvoiceRegistrationsInput{IdempotencyKey: request.GetIdempotencyKey(), Reason: request.GetReason()}
	for _, raw := range request.GetBillVersions() {
		if raw == nil {
			return nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		id, err := uuid.Parse(strings.TrimSpace(raw.GetBillId()))
		if err != nil {
			return nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		input.BillVersions = append(input.BillVersions, biz.FinanceInvoiceBillVersion{BillID: id, ExpectedVersion: raw.GetExpectedVersion()})
	}
	for _, raw := range request.GetEntries() {
		if raw == nil {
			return nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		entry := biz.FinanceInvoiceRevisionEntry{ExpectedVersion: raw.GetExpectedVersion(), Remove: raw.GetRemove()}
		if raw.GetId() != "" {
			entry.ID, err = uuid.Parse(strings.TrimSpace(raw.GetId()))
			if err != nil {
				return nil, biz.ErrFinanceInvoiceInvalidArgument
			}
		}
		if raw.GetRegistration() != nil {
			parsed, err := invoiceRegistrationInputsFromAPI([]*v1.InvoiceRegistrationInput{raw.GetRegistration()})
			if err != nil {
				return nil, err
			}
			entry.Registration = &parsed[0]
		}
		input.Entries = append(input.Entries, entry)
	}
	items, err := s.invoiceUsecase.ReviseRegistrations(ctx, scope, principal.UserID, input)
	if err != nil {
		return nil, err
	}
	data := make([]*v1.FinanceInvoice, 0, len(items))
	for _, item := range items {
		data = append(data, financeInvoiceToAPI(item))
	}
	return ok(ctx, &v1.ReviseInvoiceRegistrationsResponse{Data: data}), nil
}
func (s *SettlementService) GetInvoiceRevisionImpact(ctx context.Context, request *v1.GetInvoiceRevisionImpactRequest) (*v1.GetInvoiceRevisionImpactResponse, error) {
	principal, id, err := financePrincipalAndID(ctx, request.GetId())
	if err != nil {
		return nil, err
	}
	scope, err := organizationIDsForPermission(principal, access.FinanceInvoiceUpdate, true)
	if err != nil {
		return nil, err
	}
	impact, err := s.invoiceUsecase.GetRevisionImpact(ctx, scope, id)
	if err != nil {
		return nil, err
	}
	data := &v1.FinanceInvoiceRevisionImpact{Invoice: financeInvoiceToAPI(impact.Invoice), CanRevise: impact.CanRevise}
	for _, b := range impact.Bills {
		data.Bills = append(data.Bills, financeBillToAPI(b))
	}
	for _, d := range impact.Dependencies {
		data.Dependencies = append(data.Dependencies, &v1.FinanceInvoiceSettlementDependency{Id: d.ID.String(), BillId: d.BillID.String(), Kind: d.Kind, DocumentNo: d.DocumentNo, BillNo: d.BillNo, Currency: d.Currency, Amount: d.Amount.StringFixed(8)})
	}
	return ok(ctx, &v1.GetInvoiceRevisionImpactResponse{Data: data}), nil
}

// invoiceRegistrationInputsFromAPI 只解析明确十进制值与来源身份，金额关系与余额由领域/事务校验。
func invoiceRegistrationInputsFromAPI(records []*v1.InvoiceRegistrationInput) ([]biz.FinanceInvoiceRegistrationInput, error) {
	out := make([]biz.FinanceInvoiceRegistrationInput, 0, len(records))
	for _, raw := range records {
		if raw == nil {
			return nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		profile, err := uuid.Parse(strings.TrimSpace(raw.GetInvoiceProfileId()))
		if err != nil {
			return nil, biz.ErrFinanceInvoiceInvalidArgument
		}
		total, err := invoiceRegistrationDecimal(raw.GetTotalAmount())
		if err != nil {
			return nil, err
		}
		record := biz.FinanceInvoiceRegistrationInput{InvoiceProfileID: profile, InvoiceType: biz.FinanceInvoiceType(strings.ToUpper(raw.GetInvoiceType())), TaxInvoiceNo: raw.GetTaxInvoiceNo(), InvoiceDate: raw.GetInvoiceDate(), TotalAmount: total, Note: raw.Note}
		for _, a := range raw.GetAllocations() {
			if a == nil {
				return nil, biz.ErrFinanceInvoiceInvalidArgument
			}
			bill, err := uuid.Parse(strings.TrimSpace(a.GetBillId()))
			if err != nil {
				return nil, biz.ErrFinanceInvoiceInvalidArgument
			}
			amount, err := invoiceRegistrationDecimal(a.GetAmount())
			if err != nil {
				return nil, err
			}
			allocation := biz.FinanceInvoiceAllocationInput{BillID: bill, ExpectedVersion: a.GetExpectedVersion(), Amount: amount}
			for _, f := range a.GetFees() {
				if f == nil {
					return nil, biz.ErrFinanceInvoiceInvalidArgument
				}
				fee, err := uuid.Parse(strings.TrimSpace(f.GetOrderFeeId()))
				if err != nil {
					return nil, biz.ErrFinanceInvoiceInvalidArgument
				}
				amount, err := invoiceRegistrationDecimal(f.GetAmount())
				if err != nil {
					return nil, err
				}
				tax, err := invoiceRegistrationDecimal(f.GetTaxAmount())
				if err != nil {
					return nil, err
				}
				allocation.Fees = append(allocation.Fees, biz.FinanceInvoiceFeeInput{OrderFeeID: fee, Amount: amount, TaxAmount: tax})
			}
			record.Allocations = append(record.Allocations, allocation)
		}
		out = append(out, record)
	}
	return out, nil
}
func invoiceRegistrationDecimal(raw string) (decimal.Decimal, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return decimal.Zero, biz.ErrFinanceInvoiceInvalidArgument
	}
	v, err := decimal.NewFromString(raw)
	if err != nil || strings.ContainsAny(raw, "eE") {
		return decimal.Zero, biz.ErrFinanceInvoiceInvalidArgument
	}
	return v, nil
}

func invoiceAllocationToAPI(a *biz.FinanceInvoiceFeeAllocation) *v1.FinanceInvoiceFeeAllocation {
	return &v1.FinanceInvoiceFeeAllocation{Id: a.ID.String(), InvoiceBillId: a.InvoiceBillID.String(), BillId: a.BillID.String(), BillLineId: a.BillLineID.String(), OrderFeeId: a.OrderFeeID.String(), FeeCode: a.FeeCode, FeeName: a.FeeName, Currency: a.Currency, SourceTaxRate: financeDecimalPointer(a.SourceTaxRate, 4), Amount: a.Amount.StringFixed(8), NetAmount: a.NetAmount.StringFixed(8), TaxAmount: a.TaxAmount.StringFixed(8)}
}
func invoiceRevisionToAPI(r *biz.FinanceInvoiceRevision) *v1.FinanceInvoiceRevision {
	return &v1.FinanceInvoiceRevision{Id: r.ID.String(), RevisionNumber: r.RevisionNumber, Reason: r.Reason, ActorName: r.ActorName, CreatedAt: r.CreatedAt.UTC().Format(time.RFC3339), BeforeSnapshot: string(r.Before), AfterSnapshot: string(r.After)}
}

func (s *SettlementService) ListInvoiceProfilesForParty(ctx context.Context, request *v1.ListInvoiceProfilesForPartyRequest) (*v1.ListInvoiceProfilesForPartyResponse, error) {
	principal, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil, err
	}
	scope, err := organizationIDsForPermission(principal, access.FinanceInvoiceCreate, true)
	if err != nil {
		return nil, err
	}
	party, err := uuid.Parse(strings.TrimSpace(request.GetSettlementPartyId()))
	if err != nil {
		return nil, biz.ErrFinanceInvoiceInvalidArgument
	}
	result, err := s.invoiceUsecase.ListProfilesForParty(ctx, scope, party)
	if err != nil {
		return nil, err
	}
	data := &v1.FinanceInvoiceProfilesForBill{OrganizationId: result.OrganizationID.String(), SettlementPartyId: result.SettlementPartyID.String()}
	for _, item := range result.Items {
		data.Data = append(data.Data, &v1.FinanceInvoiceProfileOption{Id: item.ID.String(), InvoiceTitle: item.InvoiceTitle, TaxpayerIdentificationNo: item.TaxpayerIdentificationNo, DefaultInvoiceType: string(item.DefaultInvoiceType), IsDefault: item.IsDefault})
	}
	return okList(ctx, &v1.ListInvoiceProfilesForPartyResponse{Data: data}), nil
}
