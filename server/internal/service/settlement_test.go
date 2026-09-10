package service

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	v1 "github.com/roncin/roncin-go-admin/server/api/finance/v1"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

type verificationCreationCandidateRepoStub struct {
	biz.VerificationRepo
	organizationID uuid.UUID
	filter         biz.VerificationCreationCandidateFilter
	result         *biz.VerificationCreationCandidates
	err            error
}

type invoiceCreationCandidateRepoStub struct {
	biz.FinanceInvoiceRepo
	organizationID uuid.UUID
	filter         biz.FinanceInvoiceCreationBillFilter
	billID         uuid.UUID
	profileScope   []uuid.UUID
	billResult     *biz.FinanceInvoiceCreationBillListResult
	profileResult  *biz.FinanceInvoiceProfilesForBill
	billErr        error
	profileErr     error
}

type billCreationCandidateServiceRepoStub struct {
	biz.FinanceBillRepo
	organizationID        uuid.UUID
	filter                biz.FinanceBillCreationCandidateFilter
	organizationIDs       []uuid.UUID
	fees                  []*biz.FinanceBillableFee
	createdOrganizationID uuid.UUID
	err                   error
	batch                 *biz.FinanceBillBatch
}

type settlementServiceTransactorStub struct{}

func (settlementServiceTransactorStub) WithinTransaction(ctx context.Context, operation func(context.Context) error) error {
	return operation(ctx)
}

type billSettlementAccountRepoStub struct {
	biz.PartnerAccountRepo
	listItems     []*biz.PartnerAccount
	listOrgID     uuid.UUID
	listPartnerID uuid.UUID
	listFilter    biz.PartnerAccountFilter
}

type billSettlementAccountUpdateRepoStub struct {
	biz.FinanceBillRepo
	bill *biz.FinanceBill
}

func (s *billSettlementAccountUpdateRepoStub) Get(context.Context, []uuid.UUID, uuid.UUID) (*biz.FinanceBill, error) {
	return s.bill, nil
}

func (s *billSettlementAccountRepoStub) List(_ context.Context, organizationID, partnerID uuid.UUID, filter biz.PartnerAccountFilter) ([]*biz.PartnerAccount, error) {
	s.listOrgID = organizationID
	s.listPartnerID = partnerID
	s.listFilter = filter
	return s.listItems, nil
}

func (s *billCreationCandidateServiceRepoStub) ListCreationCandidates(_ context.Context, organizationID uuid.UUID, filter biz.FinanceBillCreationCandidateFilter) (*biz.FinanceBillCreationCandidateResult, error) {
	s.organizationID = organizationID
	s.filter = filter
	return &biz.FinanceBillCreationCandidateResult{}, s.err
}

func (s *billCreationCandidateServiceRepoStub) LoadBillableFeesScoped(_ context.Context, organizationIDs []uuid.UUID, _ []uuid.UUID) ([]*biz.FinanceBillableFee, error) {
	s.organizationIDs = append([]uuid.UUID(nil), organizationIDs...)
	return s.fees, s.err
}

func (s *billCreationCandidateServiceRepoStub) LoadBillableFees(_ context.Context, _ uuid.UUID, _ []uuid.UUID) ([]*biz.FinanceBillableFee, error) {
	return s.fees, s.err
}

func (s *billCreationCandidateServiceRepoStub) GetBatchByIdempotencyKey(context.Context, uuid.UUID, string) (*biz.FinanceBillBatch, error) {
	return s.batch, nil
}

func (s *billCreationCandidateServiceRepoStub) CreateBatch(_ context.Context, batch *biz.FinanceBillBatch, _ string, _ *biz.AuditEvent) (*biz.FinanceBillBatch, error) {
	s.createdOrganizationID = batch.OrganizationID
	s.batch = batch
	return batch, s.err
}

func (*billCreationCandidateServiceRepoStub) ValidateBillCurrencies(context.Context, []string) error {
	return nil
}

func (*billCreationCandidateServiceRepoStub) HydrateBillSettlementAccounts(_ context.Context, bills []*biz.FinanceBill) error {
	for _, bill := range bills {
		bill.SettlementAccountName = "测试账户"
		bill.SettlementAccountHolder = "测试客户"
		bill.SettlementBankName = "测试银行"
		bill.SettlementBankAccount = "001"
		bill.SettlementAccountCurrency = bill.Currency
	}
	return nil
}

func (s *invoiceCreationCandidateRepoStub) ListCreationBills(_ context.Context, organizationID uuid.UUID, filter biz.FinanceInvoiceCreationBillFilter) (*biz.FinanceInvoiceCreationBillListResult, error) {
	s.organizationID = organizationID
	s.filter = filter
	return s.billResult, s.billErr
}

func (s *invoiceCreationCandidateRepoStub) ListProfilesForBill(_ context.Context, organizationIDs []uuid.UUID, billID uuid.UUID) (*biz.FinanceInvoiceProfilesForBill, error) {
	s.profileScope = append([]uuid.UUID(nil), organizationIDs...)
	s.billID = billID
	return s.profileResult, s.profileErr
}

func (s *verificationCreationCandidateRepoStub) ListCreationCandidates(_ context.Context, organizationID uuid.UUID, filter biz.VerificationCreationCandidateFilter) (*biz.VerificationCreationCandidates, error) {
	s.organizationID = organizationID
	s.filter = filter
	return s.result, s.err
}

func TestFeeLedgerRequestedOrganizationOnlyNarrowsMatchingPermissionScope(t *testing.T) {
	currentOrganizationID := uuid.New()
	allowedOrganizationID := uuid.New()
	deniedOrganizationID := uuid.New()
	principal := &biz.Principal{
		Organization:      biz.Organization{ID: currentOrganizationID},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: currentOrganizationID}, {ID: allowedOrganizationID}, {ID: deniedOrganizationID}},
		RoleGrants: []biz.RoleGrant{{
			RoleCode:  "fee-reader",
			DataScope: biz.DataScopeOrganization,
			Permissions: map[string]struct{}{
				access.FinanceFeeRead: {},
			},
			OrganizationAccesses: []biz.OrganizationAccess{{OrganizationID: allowedOrganizationID}},
		}},
	}

	requestedID := allowedOrganizationID.String()
	organizationIDs, err := organizationIDsForRequestedOrganization(principal, access.FinanceFeeRead, false, &requestedID)
	if err != nil || len(organizationIDs) != 1 || organizationIDs[0] != allowedOrganizationID {
		t.Fatalf("费用读取组织收窄失败: ids=%v err=%v", organizationIDs, err)
	}

	deniedID := deniedOrganizationID.String()
	if _, err := organizationIDsForRequestedOrganization(principal, access.FinanceFeeRead, false, &deniedID); err != biz.ErrPermissionDenied {
		t.Fatalf("越权费用组织错误 = %v，期望权限拒绝", err)
	}
	malformedID := "not-a-uuid"
	if _, err := organizationIDsForRequestedOrganization(principal, access.FinanceFeeRead, false, &malformedID); err != biz.ErrFinanceLedgerInvalidArgument {
		t.Fatalf("错误组织 ID = %v，期望参数错误", err)
	}
}

func TestFinanceStatusEnumConversions(t *testing.T) {
	progress := v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_INVOICED_PARTIALLY_VERIFIED
	if got := feeLedgerFinancialProgressFromAPI(&progress); got != biz.FeeLedgerInvoicedPartiallyVerified {
		t.Fatalf("财务进度 = %q", got)
	}
	if got := feeLedgerFinancialProgressToAPI(biz.FeeLedgerCompleted); got != v1.FeeLedgerFinancialProgress_FEE_LEDGER_FINANCIAL_PROGRESS_COMPLETED {
		t.Fatalf("财务进度 API 值 = %v", got)
	}
	if got := feeLedgerFinancialProgressFromAPI(nil); got != "" {
		t.Fatalf("空财务进度 = %q", got)
	}

	bill := v1.FinanceBillStatus_FINANCE_BILL_STATUS_CONFIRMED
	if got := financeBillStatusFromAPI(&bill); got != biz.FinanceBillConfirmed {
		t.Fatalf("账单状态 = %q", got)
	}
	if got := financeBillStatusToAPI(biz.FinanceBillCancelled); got != v1.FinanceBillStatus_FINANCE_BILL_STATUS_CANCELLED {
		t.Fatalf("账单 API 状态 = %v", got)
	}

	invoice := v1.FinanceInvoiceStatus_FINANCE_INVOICE_STATUS_RED_FLUSHED
	if got := financeInvoiceStatusFromAPI(&invoice); got != biz.FinanceInvoiceRedFlushed {
		t.Fatalf("发票状态 = %q", got)
	}
	if got := financeInvoiceStatusToAPI(biz.FinanceInvoiceIssued); got != v1.FinanceInvoiceStatus_FINANCE_INVOICE_STATUS_ISSUED {
		t.Fatalf("发票 API 状态 = %v", got)
	}

	cashflow := v1.FinanceCashflowStatus_FINANCE_CASHFLOW_STATUS_CONFIRMED
	if got := financeCashflowStatusFromAPI(&cashflow); got != biz.FinanceCashflowConfirmed {
		t.Fatalf("资金流水状态 = %q", got)
	}
	if got := financeCashflowStatusToAPI(biz.FinanceCashflowDraft); got != v1.FinanceCashflowStatus_FINANCE_CASHFLOW_STATUS_DRAFT {
		t.Fatalf("资金流水 API 状态 = %v", got)
	}

	verification := v1.FinanceVerificationStatus_FINANCE_VERIFICATION_STATUS_REVERSED
	if got := financeVerificationStatusFromAPI(&verification); got != biz.VerificationReversed {
		t.Fatalf("核销状态 = %q", got)
	}
	if got := financeVerificationStatusToAPI(biz.VerificationActive); got != v1.FinanceVerificationStatus_FINANCE_VERIFICATION_STATUS_ACTIVE {
		t.Fatalf("核销 API 状态 = %v", got)
	}

	commission := v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_PAID
	if got := financeCommissionStatusFromAPI(&commission); got != biz.CommissionPaid {
		t.Fatalf("提成状态 = %q", got)
	}
	if got := financeCommissionStatusToAPI(biz.CommissionConfirmed); got != v1.FinanceCommissionStatus_FINANCE_COMMISSION_STATUS_CONFIRMED {
		t.Fatalf("提成 API 状态 = %v", got)
	}
}

func TestFinanceStatusEnumConversionsKeepEmptyFilter(t *testing.T) {
	if got := financeBillStatusFromAPI(nil); got != "" {
		t.Fatalf("空账单筛选 = %q", got)
	}
	if got := financeInvoiceStatusFromAPI(nil); got != "" {
		t.Fatalf("空发票筛选 = %q", got)
	}
	if got := financeCashflowStatusFromAPI(nil); got != "" {
		t.Fatalf("空流水筛选 = %q", got)
	}
	if got := financeVerificationStatusFromAPI(nil); got != "" {
		t.Fatalf("空核销筛选 = %q", got)
	}
	if got := financeCommissionStatusFromAPI(nil); got != "" {
		t.Fatalf("空提成筛选 = %q", got)
	}
}

func TestVerificationCreateOrganizationPurposeUsesWritableScope(t *testing.T) {
	permission, writable, ok := financeOrganizationPurposePermission(v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_VERIFICATION_CREATE)
	if !ok || permission != access.FinanceVerificationCreate || !writable {
		t.Fatalf("核销创建组织用途映射错误: permission=%q writable=%t ok=%t", permission, writable, ok)
	}
	allowedOrganizationID := uuid.New()
	deniedOrganizationID := uuid.New()
	principal := &biz.Principal{
		Organization: biz.Organization{ID: allowedOrganizationID},
		RoleGrants: []biz.RoleGrant{{
			RoleCode:  "verification-creator",
			DataScope: biz.DataScopeOrganization,
			Permissions: map[string]struct{}{
				access.FinanceVerificationCreate: {},
			},
		}},
	}
	allowed := allowedOrganizationID.String()
	organizationIDs, err := organizationIDsForRequestedOrganization(principal, permission, writable, &allowed)
	if err != nil || len(organizationIDs) != 1 || organizationIDs[0] != allowedOrganizationID {
		t.Fatalf("核销创建允许组织收窄失败: ids=%v err=%v", organizationIDs, err)
	}
	denied := deniedOrganizationID.String()
	if _, err := organizationIDsForRequestedOrganization(principal, permission, writable, &denied); err != biz.ErrPermissionDenied {
		t.Fatalf("核销创建越权组织错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
	}
}

func TestInvoiceCreateCandidatesUseWritableOrganizationScope(t *testing.T) {
	organizationID := uuid.New()
	deniedOrganizationID := uuid.New()
	billID := uuid.New()
	principal := &biz.Principal{
		Organization:      biz.Organization{ID: uuid.New()},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: organizationID}},
		RoleGrants: []biz.RoleGrant{{
			RoleCode:  "invoice-creator",
			DataScope: biz.DataScopeOrganization,
			Permissions: map[string]struct{}{
				access.FinanceInvoiceCreate: {},
			},
			OrganizationAccesses: []biz.OrganizationAccess{{OrganizationID: organizationID, Writable: true}},
		}},
	}
	repo := &invoiceCreationCandidateRepoStub{
		billResult:    &biz.FinanceInvoiceCreationBillListResult{},
		profileResult: &biz.FinanceInvoiceProfilesForBill{OrganizationID: organizationID, SettlementPartyID: uuid.New()},
	}
	service := &SettlementService{invoiceUsecase: biz.NewFinanceInvoiceUsecase(repo, nil, nil)}
	ctx := biz.WithPrincipal(context.Background(), principal)
	direction := string(biz.OrderFeeReceivable)
	currency := "usd"

	permission, writable, ok := financeOrganizationPurposePermission(v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_INVOICE_CREATE)
	if !ok || permission != access.FinanceInvoiceCreate || !writable {
		t.Fatalf("发票创建组织用途映射错误: permission=%q writable=%t ok=%t", permission, writable, ok)
	}
	_, err := service.ListInvoiceCreationBills(ctx, &v1.ListInvoiceCreationBillsRequest{
		OrganizationId: organizationID.String(), Page: 1, PageSize: 20, Direction: &direction, Currency: &currency,
	})
	if err != nil {
		t.Fatalf("读取发票创建账单候选失败: %v", err)
	}
	if repo.organizationID != organizationID || repo.filter.Direction != biz.OrderFeeReceivable || repo.filter.Currency != "USD" {
		t.Fatalf("账单候选未使用发票创建可写范围: org=%s filter=%+v", repo.organizationID, repo.filter)
	}
	if _, err := service.ListInvoiceCreationBills(ctx, &v1.ListInvoiceCreationBillsRequest{OrganizationId: deniedOrganizationID.String(), Page: 1, PageSize: 20}); err != biz.ErrPermissionDenied {
		t.Fatalf("越权所属公司错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
	}
	_, err = service.ListInvoiceProfilesForBill(ctx, &v1.ListInvoiceProfilesForBillRequest{BillId: billID.String()})
	if err != nil {
		t.Fatalf("读取发票创建资料失败: %v", err)
	}
	if repo.billID != billID || len(repo.profileScope) != 1 || repo.profileScope[0] != organizationID {
		t.Fatalf("开票资料未按发票创建可写范围定位账单: bill=%s scope=%v", repo.billID, repo.profileScope)
	}
	databaseErr := errors.New("发票候选数据库故障")
	repo.billErr = databaseErr
	if _, err := service.ListInvoiceCreationBills(ctx, &v1.ListInvoiceCreationBillsRequest{OrganizationId: organizationID.String(), Page: 1, PageSize: 20}); !errors.Is(err, databaseErr) {
		t.Fatalf("账单候选数据库错误 = %v，期望原样返回 %v", err, databaseErr)
	}
	repo.billErr = nil
	repo.profileErr = databaseErr
	if _, err := service.ListInvoiceProfilesForBill(ctx, &v1.ListInvoiceProfilesForBillRequest{BillId: billID.String()}); !errors.Is(err, databaseErr) {
		t.Fatalf("开票资料数据库错误 = %v，期望原样返回 %v", err, databaseErr)
	}
}

func TestBillCreationCandidatesUseCreateWritableOrganization(t *testing.T) {
	allowed, denied := uuid.New(), uuid.New()
	p := &biz.Principal{Organization: biz.Organization{ID: uuid.New()}, OrganizationNodes: []biz.OrganizationScopeNode{{ID: allowed}}, RoleGrants: []biz.RoleGrant{{RoleCode: "creator", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.FinanceBillCreate: {}}, OrganizationAccesses: []biz.OrganizationAccess{{OrganizationID: allowed, Writable: true}}}}}
	repo := &billCreationCandidateServiceRepoStub{}
	service := &SettlementService{billUsecase: biz.NewFinanceBillUsecase(repo, nil, nil)}
	ctx := biz.WithPrincipal(context.Background(), p)
	permission, writable, ok := financeOrganizationPurposePermission(v1.FinanceOrganizationPurpose_FINANCE_ORGANIZATION_PURPOSE_BILL_CREATE)
	if !ok || permission != access.FinanceBillCreate || !writable {
		t.Fatalf("建账用途错误:%q %t", permission, writable)
	}
	if _, err := service.ListBillCreationCandidates(ctx, &v1.ListBillCreationCandidatesRequest{OrganizationId: allowed.String(), Page: 1, PageSize: 20}); err != nil {
		t.Fatal(err)
	}
	if repo.organizationID != allowed {
		t.Fatalf("候选组织=%s", repo.organizationID)
	}
	if _, err := service.ListBillCreationCandidates(ctx, &v1.ListBillCreationCandidatesRequest{OrganizationId: denied.String(), Page: 1, PageSize: 20}); err != biz.ErrPermissionDenied {
		t.Fatalf("越权=%v", err)
	}
	repo.err = errors.New("db")
	if _, err := service.ListBillCreationCandidates(ctx, &v1.ListBillCreationCandidatesRequest{OrganizationId: allowed.String(), Page: 1, PageSize: 20}); !errors.Is(err, repo.err) {
		t.Fatalf("错误未透传:%v", err)
	}
}

func TestBillSettlementAccountCandidatesUseBillCreateScopeAndDirection(t *testing.T) {
	organizationID, partyID := uuid.New(), uuid.New()
	principal := &biz.Principal{
		Organization:      biz.Organization{ID: uuid.New()},
		OrganizationNodes: []biz.OrganizationScopeNode{{ID: organizationID}},
		RoleGrants: []biz.RoleGrant{{
			RoleCode: "bill-creator", DataScope: biz.DataScopeOrganization,
			Permissions:          map[string]struct{}{access.FinanceBillCreate: {}},
			OrganizationAccesses: []biz.OrganizationAccess{{OrganizationID: organizationID, Writable: true}},
		}},
	}
	repo := &billSettlementAccountRepoStub{listItems: []*biz.PartnerAccount{
		{ID: uuid.New(), Name: "应收账户", AccountHolder: "结算单位", BankName: "银行", AccountNo: "001", Currency: "USD", Usage: biz.PartnerAccountUsageReceivable, Enabled: true, IsDefaultReceivable: true},
		{ID: uuid.New(), Name: "双向账户", AccountHolder: "结算单位", BankName: "银行", AccountNo: "002", Currency: "USD", Usage: biz.PartnerAccountUsageBoth, Enabled: true},
		{ID: uuid.New(), Name: "应付账户", AccountHolder: "结算单位", BankName: "银行", AccountNo: "003", Currency: "USD", Usage: biz.PartnerAccountUsagePayable, Enabled: true},
	}}
	service := &SettlementService{accountUsecase: biz.NewPartnerAccountUsecase(repo)}
	response, err := service.ListBillSettlementAccountCandidates(biz.WithPrincipal(context.Background(), principal), &v1.ListBillSettlementAccountCandidatesRequest{
		OrganizationId: organizationID.String(), SettlementPartyId: partyID.String(), Direction: string(biz.OrderFeeReceivable), Currency: "usd",
	})
	if err != nil {
		t.Fatalf("仅建账创建权限查询账户候选失败: %v", err)
	}
	if repo.listOrgID != organizationID || repo.listPartnerID != partyID || repo.listFilter.Enabled == nil || !*repo.listFilter.Enabled || repo.listFilter.Currency != "USD" {
		t.Fatalf("账户候选查询未按建账写范围与固定事实过滤: org=%s partner=%s filter=%+v", repo.listOrgID, repo.listPartnerID, repo.listFilter)
	}
	if len(response.Data) != 2 || response.Data[0].Name != "应收账户" || response.Data[1].Name != "双向账户" || !response.Data[0].IsDefault {
		t.Fatalf("应收候选必须排除用途不符账户并保留默认标记: %#v", response.Data)
	}
}

func TestBillSettlementAccountUpdateCandidatesRejectNonDraftBill(t *testing.T) {
	organizationID, billID := uuid.New(), uuid.New()
	principal := &biz.Principal{Organization: biz.Organization{ID: organizationID}, RoleGrants: []biz.RoleGrant{{
		RoleCode: "bill-editor", DataScope: biz.DataScopeOrganization,
		Permissions: map[string]struct{}{access.FinanceBillUpdate: {}},
	}}}
	billRepo := &billSettlementAccountUpdateRepoStub{bill: &biz.FinanceBill{
		ID: billID, OrganizationID: organizationID, SettlementPartyID: uuid.New(), Direction: biz.OrderFeeReceivable,
		Currency: "CNY", Status: biz.FinanceBillConfirmed,
	}}
	service := &SettlementService{
		billUsecase:    biz.NewFinanceBillUsecase(billRepo, nil, nil),
		accountUsecase: biz.NewPartnerAccountUsecase(&billSettlementAccountRepoStub{}),
	}
	if _, err := service.ListBillSettlementAccountUpdateCandidates(biz.WithPrincipal(context.Background(), principal), &v1.ListBillSettlementAccountUpdateCandidatesRequest{BillId: billID.String()}); !errors.Is(err, biz.ErrFinanceBillInvalidTransition) {
		t.Fatalf("已确认账单不能加载草稿改选账户候选: %v", err)
	}
}

func TestPreviewBillBatchConfigsRejectEstimatedRateWithoutCurrency(t *testing.T) {
	rate := "1.2"
	configs, err := previewBillBatchConfigsFromAPI(&v1.PreviewBillBatchRequest{GroupConfigs: []*v1.BillBatchPreviewGroupConfigInput{{GroupKey: "group", EstimatedInvoiceRate: &rate}}})
	if err != biz.ErrFinanceBillInvalidArgument || len(configs.groups) != 0 {
		t.Fatalf("预计开票汇率缺币种错误: configs=%#v err=%v", configs, err)
	}
}

func TestBillBatchPreviewAndCreateRequireDeclaredSourceOrganization(t *testing.T) {
	organizationID, otherOrganizationID := uuid.New(), uuid.New()
	feeID, orderID, partyID := uuid.New(), uuid.New(), uuid.New()
	taxRate := decimal.RequireFromString("6")
	fee := &biz.FinanceBillableFee{
		OrganizationID: organizationID,
		OrderNo:        "SE202609100001",
		BusinessType:   "SE",
		Fee: &biz.OrderFee{
			ID: feeID, OrderID: orderID, Direction: biz.OrderFeeReceivable, Status: biz.OrderFeeConfirmed,
			SettlementPartyID: partyID, SettlementPartyName: "测试客户", FeeCode: "OCEAN", FeeName: "海运费",
			Currency: "CNY", BaseCurrency: "CNY", TaxRate: &taxRate, ExchangeRate: decimal.NewFromInt(1),
			Quantity: decimal.NewFromInt(1), UnitPrice: decimal.NewFromInt(100), TotalAmount: decimal.NewFromInt(100),
			NetAmount: decimal.RequireFromString("94.33962264"), TaxAmount: decimal.RequireFromString("5.66037736"),
			BaseCurrencyAmount: decimal.NewFromInt(100), Version: 1,
		},
	}
	principal := &biz.Principal{
		UserID: uuid.New(), Organization: biz.Organization{ID: organizationID},
		RoleGrants: []biz.RoleGrant{{RoleCode: "bill-creator", DataScope: biz.DataScopeOrganization, Permissions: map[string]struct{}{access.FinanceBillCreate: {}}}},
	}
	repo := &billCreationCandidateServiceRepoStub{fees: []*biz.FinanceBillableFee{fee}}
	rateSettingID := uuid.New()
	rateRepo := &exchangeRateStub{
		context:  &biz.ExchangeRateContext{OwnerOrganizationID: organizationID, BaseCurrency: "CNY"},
		resolved: &biz.ResolvedExchangeRate{Rate: decimal.NewFromInt(1), Source: "SYSTEM", RateDate: "2026-09-10", SettingID: &rateSettingID},
	}
	accountID := uuid.New()
	service := &SettlementService{billUsecase: biz.NewFinanceBillUsecase(repo, biz.NewExchangeRateUsecase(rateRepo), settlementServiceTransactorStub{})}
	ctx := biz.WithPrincipal(context.Background(), principal)

	policy := &v1.BillGroupingPolicy{Mode: v1.BillGroupingMode_BILL_GROUPING_MODE_NORMAL}
	preview, err := service.PreviewBillBatch(ctx, &v1.PreviewBillBatchRequest{
		FeeIds: []string{feeID.String()}, GroupingPolicy: policy, OrganizationId: organizationID.String(),
	})
	if err != nil || len(preview.GetData()) != 1 || len(repo.organizationIDs) != 1 || repo.organizationIDs[0] != organizationID {
		t.Fatalf("同组织预览失败: response=%#v scope=%v err=%v", preview, repo.organizationIDs, err)
	}
	preview, err = service.PreviewBillBatch(ctx, &v1.PreviewBillBatchRequest{
		FeeIds: []string{feeID.String()}, GroupingPolicy: policy, OrganizationId: organizationID.String(),
		GroupConfigs: []*v1.BillBatchPreviewGroupConfigInput{{GroupKey: preview.Data[0].GetGroupKey(), BillDate: "2026-09-10", SettlementAccountId: accountID.String()}},
	})
	if err != nil || preview.GetPreviewToken() == "" {
		t.Fatalf("完整配置预览失败: response=%#v err=%v", preview, err)
	}
	created, err := service.CreateBillBatch(ctx, &v1.CreateBillBatchRequest{
		FeeIds: []string{feeID.String()}, GroupingPolicy: policy, OrganizationId: organizationID.String(),
		PreviewToken: preview.GetPreviewToken(), IdempotencyKey: "bill-batch-org-assertion",
		Groups: []*v1.CreateBillBatchGroupInput{{GroupKey: preview.Data[0].GetGroupKey(), StatementTitle: "测试客户", BillDate: "2026-09-10", SettlementAccountId: accountID.String()}},
	})
	if err != nil || created.GetData() == nil || repo.createdOrganizationID != organizationID {
		t.Fatalf("同组织创建失败: response=%#v err=%v", created, err)
	}

	fee.OrganizationID = otherOrganizationID
	if _, err := service.PreviewBillBatch(ctx, &v1.PreviewBillBatchRequest{
		FeeIds: []string{feeID.String()}, GroupingPolicy: policy, OrganizationId: organizationID.String(),
	}); !errors.Is(err, biz.ErrFinanceBillInvalidArgument) {
		t.Fatalf("来源声明为 A、费用解析为 B 时预览错误 = %v", err)
	}
	if _, err := service.CreateBillBatch(ctx, &v1.CreateBillBatchRequest{
		FeeIds: []string{feeID.String()}, GroupingPolicy: policy, OrganizationId: organizationID.String(),
	}); !errors.Is(err, biz.ErrFinanceBillInvalidArgument) {
		t.Fatalf("来源声明为 A、费用解析为 B 时创建错误 = %v", err)
	}
	if _, err := service.PreviewBillBatch(ctx, &v1.PreviewBillBatchRequest{
		FeeIds: []string{feeID.String()}, GroupingPolicy: policy, OrganizationId: otherOrganizationID.String(),
	}); !errors.Is(err, biz.ErrPermissionDenied) {
		t.Fatalf("无 B 公司建账权限时错误 = %v", err)
	}
}

func TestListVerificationCreationCandidatesUsesCreateWritableOrganization(t *testing.T) {
	organizationID := uuid.New()
	partyID := uuid.New()
	principal := &biz.Principal{
		Organization: biz.Organization{ID: uuid.New()},
		OrganizationNodes: []biz.OrganizationScopeNode{
			{ID: organizationID},
		},
		RoleGrants: []biz.RoleGrant{{
			RoleCode:  "verification-creator",
			DataScope: biz.DataScopeOrganization,
			Permissions: map[string]struct{}{
				access.FinanceVerificationCreate: {},
			},
			OrganizationAccesses: []biz.OrganizationAccess{{OrganizationID: organizationID, Writable: true}},
		}},
	}
	repo := &verificationCreationCandidateRepoStub{result: &biz.VerificationCreationCandidates{}}
	service := &SettlementService{verificationUsecase: biz.NewVerificationUsecase(repo, nil, nil)}
	ctx := biz.WithPrincipal(context.Background(), principal)
	_, err := service.ListVerificationCreationCandidates(ctx, &v1.ListVerificationCreationCandidatesRequest{
		OrganizationId:    organizationID.String(),
		Direction:         "RECEIVABLE",
		SettlementPartyId: partyID.String(),
		Currency:          "usd",
	})
	if err != nil {
		t.Fatalf("读取核销创建候选失败: %v", err)
	}
	if repo.organizationID != organizationID || repo.filter.Direction != biz.OrderFeeReceivable || repo.filter.SettlementPartyID != partyID || repo.filter.Currency != "USD" {
		t.Fatalf("候选未使用核销创建权限范围: org=%s filter=%+v", repo.organizationID, repo.filter)
	}
	if _, err := service.ListVerificationCreationCandidates(ctx, &v1.ListVerificationCreationCandidatesRequest{
		OrganizationId:    uuid.NewString(),
		Direction:         "RECEIVABLE",
		SettlementPartyId: partyID.String(),
		Currency:          "USD",
	}); err != biz.ErrPermissionDenied {
		t.Fatalf("越权组织错误 = %v，期望 %v", err, biz.ErrPermissionDenied)
	}

	expected := errors.New("候选数据库故障")
	repo.err = expected
	if _, err := service.ListVerificationCreationCandidates(ctx, &v1.ListVerificationCreationCandidatesRequest{
		OrganizationId:    organizationID.String(),
		Direction:         "RECEIVABLE",
		SettlementPartyId: partyID.String(),
		Currency:          "USD",
	}); !errors.Is(err, expected) {
		t.Fatalf("候选错误 = %v，期望原样返回 %v", err, expected)
	}
}
