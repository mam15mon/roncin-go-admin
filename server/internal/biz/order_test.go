package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderRepoStub struct {
	created        *Order
	createdNumber  string
	referenceCheck *OrderReferenceCheck
	referenceMatch *OrderReferenceMatch
	transitioned   *Order
	expectedStatus string
	targetStatus   string
	statusEvent    *OrderStatusChangedEvent
}

func (s *orderRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*Order, error) {
	return nil, ErrOrderNotFound
}

func (s *orderRepoStub) Find(context.Context, uuid.UUID) (*Order, error) {
	return nil, ErrOrderNotFound
}

func (s *orderRepoStub) List(_ context.Context, _ []uuid.UUID, options OrderListOptions) (*OrderList, error) {
	return &OrderList{Page: options.Page, PageSize: options.PageSize}, nil
}

func (s *orderRepoStub) FindReferenceDuplicate(_ context.Context, _ uuid.UUID, check OrderReferenceCheck) (*OrderReferenceMatch, error) {
	s.referenceCheck = &check
	return s.referenceMatch, nil
}

func (s *orderRepoStub) ListPersonnelOptions(_ context.Context, _ uuid.UUID) ([]*OrderPersonnelOption, error) {
	return nil, nil
}

func (s *orderRepoStub) Create(_ context.Context, organizationID, _ uuid.UUID, number string, input *Order) (*Order, error) {
	s.created = input
	s.createdNumber = number
	input.ID = uuid.New()
	input.OrganizationID = organizationID
	input.OrderNo = number
	input.Status = "DRAFT"
	return input, nil
}

func (s *orderRepoStub) UpdateDraft(_ context.Context, organizationID, id uuid.UUID, expectedStatus string, input *Order) (*Order, error) {
	input.ID = id
	input.OrganizationID = organizationID
	input.Status = expectedStatus
	return input, nil
}

func (s *orderRepoStub) TransitionStatus(_ context.Context, organizationID, id uuid.UUID, expectedStatus, targetStatus, _ string, _ uuid.UUID, event *OrderStatusChangedEvent) (*Order, error) {
	s.expectedStatus = expectedStatus
	s.targetStatus = targetStatus
	s.statusEvent = event
	s.transitioned = &Order{ID: id, OrganizationID: organizationID, Status: targetStatus}
	return s.transitioned, nil
}

func TestOrderCreateUsesNumberRuleAndAudits(t *testing.T) {
	repo := &orderRepoStub{}
	configRepo := &orderConfigRepoStub{allocatedRule: &NumberRule{DateFormat: DateFormatNone, SequenceLength: 4, ResetPolicy: ResetPolicyNever}, allocatedSequence: 7}
	audit := &auditRepoStub{}
	usecase := NewOrderUsecase(repo, NewOrderConfigUsecase(configRepo, audit), audit)
	organizationID := uuid.New()
	actorID := uuid.New()
	customerID := uuid.New()
	templateID := uuid.New()

	created, err := usecase.Create(context.Background(), organizationID, actorID, &Order{
		CustomerID: customerID, StatusTemplateID: templateID, BusinessType: OrderBusinessSE,
		TradeDirection: OrderTradeExport, TradeTerm: OrderTradeFOB, PaymentTerm: OrderPaymentPrepaid,
		ServiceTypeIDs: []uuid.UUID{uuid.New()}, CargoCategoryIDs: []uuid.UUID{uuid.New()},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.OrderNo != "SE0007" || repo.createdNumber != "SE0007" || configRepo.lastAllocDocType != DocumentTypeOrder {
		t.Fatalf("created order = %#v, allocated document type = %s", created, configRepo.lastAllocDocType)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "order.create" || audit.events[0].Details["order.no"] != "SE0007" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

func TestOrderRejectsInvalidAggregateAndDraftRollback(t *testing.T) {
	audit := &auditRepoStub{}
	configRepo := &orderConfigRepoStub{allocatedRule: &NumberRule{Prefix: "ORD", DateFormat: DateFormatNone, SequenceLength: 4, ResetPolicy: ResetPolicyNever}, allocatedSequence: 1}
	usecase := NewOrderUsecase(&orderRepoStub{}, NewOrderConfigUsecase(configRepo, audit), audit)
	organizationID := uuid.New()
	actorID := uuid.New()
	duplicateID := uuid.New()

	_, err := usecase.Create(context.Background(), organizationID, actorID, &Order{
		CustomerID: uuid.New(), StatusTemplateID: uuid.New(), BusinessType: OrderBusinessSE,
		TradeDirection: OrderTradeExport, TradeTerm: OrderTradeFOB, PaymentTerm: OrderPaymentPrepaid,
		CargoCategoryIDs: []uuid.UUID{duplicateID, duplicateID},
	})
	if err != ErrOrderInvalidArgument {
		t.Fatalf("duplicate cargo categories error = %v, want ErrOrderInvalidArgument", err)
	}
	_, err = usecase.TransitionStatus(context.Background(), organizationID, actorID, uuid.New(), "BOOKED", "DRAFT", "rollback")
	if err != ErrOrderStatusInvalid {
		t.Fatalf("draft rollback error = %v, want ErrOrderStatusInvalid", err)
	}
}

func TestOrderNormalizesBusinessFieldsAndRequiresCompleteCargoValue(t *testing.T) {
	input := &Order{
		CustomerID: uuid.New(), StatusTemplateID: uuid.New(), BusinessType: OrderBusinessSE,
		TradeDirection: OrderTradeExport, TradeTerm: OrderTradeFOB, PaymentTerm: OrderPaymentPrepaid,
		CustomerReferenceNo: "  CUST-001  ", InternalReferenceNo: "  INTERNAL-001  ", ContractNo: "  CONTRACT-001  ",
		CargoValue: "100000.25", CargoCurrency: " usd ", InsurancePremium: "100.50", InsuranceCurrency: " cny ",
		UNNumber: "1234", HazardClass: "3", FactoryName: "  测试工厂  ", CargoReadyAt: "2026-08-23T12:00:00+08:00", LoadingTerms: "  CY-CY  ", ReceivedAt: "2026-08-23T10:00:00+08:00",
	}
	normalized, err := normalizeOrder(input, false)
	if err != nil {
		t.Fatalf("normalizeOrder() error = %v", err)
	}
	if normalized.CustomerReferenceNo != "CUST-001" || normalized.InternalReferenceNo != "INTERNAL-001" || normalized.ContractNo != "CONTRACT-001" || normalized.CargoValue != "100000.25" || normalized.CargoCurrency != "USD" || normalized.InsurancePremium != "100.50" || normalized.InsuranceCurrency != "CNY" || normalized.UNNumber != "1234" || normalized.HazardClass != "3" || normalized.FactoryName != "测试工厂" || normalized.LoadingTerms != "CY-CY" || normalized.ReceivedAt != "2026-08-23T10:00:00+08:00" {
		t.Fatalf("normalized business fields = %#v", normalized)
	}

	invalidValues := []struct {
		value    string
		currency string
	}{
		{value: "100"},
		{currency: "USD"},
		{value: "-1", currency: "USD"},
		{value: "1.12345", currency: "USD"},
		{value: "100", currency: "US"},
	}
	for _, testCase := range invalidValues {
		invalid := *input
		invalid.CargoValue = testCase.value
		invalid.CargoCurrency = testCase.currency
		if _, err := normalizeOrder(&invalid, false); err != ErrOrderInvalidArgument {
			t.Fatalf("normalizeOrder(%q, %q) error = %v, want ErrOrderInvalidArgument", testCase.value, testCase.currency, err)
		}
	}

	invalidInsurance := *input
	invalidInsurance.InsuranceCurrency = ""
	if _, err := normalizeOrder(&invalidInsurance, false); err != ErrOrderInvalidArgument {
		t.Fatalf("incomplete insurance premium error = %v, want ErrOrderInvalidArgument", err)
	}
	invalidUNNumber := *input
	invalidUNNumber.UNNumber = "123"
	if _, err := normalizeOrder(&invalidUNNumber, false); err != ErrOrderInvalidArgument {
		t.Fatalf("invalid UN number error = %v, want ErrOrderInvalidArgument", err)
	}
}

func TestOrderNormalizesOneMasterWithMultipleHousesAndContainerRequests(t *testing.T) {
	container20GP := uuid.New()
	container40HQ := uuid.New()
	masterDocumentType := "ORIGINAL_BL"
	masterReleaseMethod := "TELEX_RELEASE"
	input := &Order{
		CustomerID: uuid.New(), StatusTemplateID: uuid.New(), BusinessType: OrderBusinessSE,
		TradeDirection: OrderTradeExport, TradeTerm: OrderTradeFOB, PaymentTerm: OrderPaymentPrepaid,
		ShippingDocuments: []*OrderShippingDocument{
			{MasterNo: " MBL-001 ", MasterDocumentType: &masterDocumentType, MasterReleaseMethod: &masterReleaseMethod, HouseNo: " HBL-001 "},
			{MasterNo: " MBL-001 ", MasterDocumentType: &masterDocumentType, MasterReleaseMethod: &masterReleaseMethod, HouseNo: " HBL-002 "},
		},
		ContainerRequests: []*OrderContainerRequest{
			{ContainerSpecID: container20GP, Quantity: 2},
			{ContainerSpecID: container40HQ, Quantity: 3},
		},
	}

	normalized, err := normalizeOrder(input, false)
	if err != nil {
		t.Fatalf("normalizeOrder() error = %v", err)
	}
	if normalized.ShippingDocuments[0].MasterNo != "MBL-001" || normalized.ShippingDocuments[1].HouseNo != "HBL-002" {
		t.Fatalf("normalized shipping documents = %#v", normalized.ShippingDocuments)
	}
	if stringPointerValue(normalized.ShippingDocuments[0].MasterDocumentType) != masterDocumentType || stringPointerValue(normalized.ShippingDocuments[1].MasterReleaseMethod) != masterReleaseMethod {
		t.Fatalf("normalized master attributes = %#v", normalized.ShippingDocuments)
	}

	inconsistentMaster := *input
	differentReleaseMethod := "ORIGINAL"
	inconsistentMaster.ShippingDocuments = []*OrderShippingDocument{
		{MasterNo: "MBL-001", MasterDocumentType: &masterDocumentType, MasterReleaseMethod: &masterReleaseMethod, HouseNo: "HBL-001"},
		{MasterNo: "mbl-001", MasterDocumentType: &masterDocumentType, MasterReleaseMethod: &differentReleaseMethod, HouseNo: "HBL-002"},
	}
	if _, err := normalizeOrder(&inconsistentMaster, false); err != ErrOrderInvalidArgument {
		t.Fatalf("inconsistent master attributes error = %v, want ErrOrderInvalidArgument", err)
	}

	duplicateHouse := *input
	duplicateHouse.ShippingDocuments = []*OrderShippingDocument{
		{MasterNo: "MBL-001", HouseNo: "HBL-001"},
		{MasterNo: "MBL-002", HouseNo: " hbl-001 "},
	}
	if _, err := normalizeOrder(&duplicateHouse, false); err != ErrOrderShippingDocumentExists {
		t.Fatalf("duplicate house error = %v, want ErrOrderShippingDocumentExists", err)
	}

	duplicateContainer := *input
	duplicateContainer.ContainerRequests = []*OrderContainerRequest{
		{ContainerSpecID: container20GP, Quantity: 1},
		{ContainerSpecID: container20GP, Quantity: 2},
	}
	if _, err := normalizeOrder(&duplicateContainer, false); err != ErrOrderInvalidArgument {
		t.Fatalf("duplicate container request error = %v, want ErrOrderInvalidArgument", err)
	}
}

func TestOrderCheckReferenceNormalizesScopeAndReturnsMatch(t *testing.T) {
	organizationID := uuid.New()
	customerID := uuid.New()
	match := &OrderReferenceMatch{OrderID: uuid.New(), OrderNo: "SE0001"}
	repo := &orderRepoStub{referenceMatch: match}
	usecase := NewOrderUsecase(repo, nil, nil)

	result, err := usecase.CheckReference(context.Background(), organizationID, OrderReferenceCheck{
		ReferenceType: OrderReferenceCustomer,
		ReferenceNo:   "  Customer-001  ",
		CustomerID:    &customerID,
	})
	if err != nil {
		t.Fatalf("CheckReference() error = %v", err)
	}
	if result != match || repo.referenceCheck == nil || repo.referenceCheck.ReferenceNo != "Customer-001" || repo.referenceCheck.CustomerID == nil || *repo.referenceCheck.CustomerID != customerID {
		t.Fatalf("reference result = %#v, check = %#v", result, repo.referenceCheck)
	}

	_, err = usecase.CheckReference(context.Background(), organizationID, OrderReferenceCheck{
		ReferenceType: OrderReferenceCustomer,
		ReferenceNo:   "Customer-001",
	})
	if err != ErrOrderInvalidArgument {
		t.Fatalf("customer reference without customer error = %v, want ErrOrderInvalidArgument", err)
	}
}

func TestOrderTransitionNormalizesStatusAndAudits(t *testing.T) {
	repo := &orderRepoStub{}
	audit := &auditRepoStub{}
	usecase := NewOrderUsecase(repo, NewOrderConfigUsecase(&orderConfigRepoStub{}, audit), audit)
	organizationID := uuid.New()
	actorID := uuid.New()
	id := uuid.New()

	updated, err := usecase.TransitionStatus(context.Background(), organizationID, actorID, id, " draft ", " booked ", "  已订舱  ")
	if err != nil {
		t.Fatalf("TransitionStatus() error = %v", err)
	}
	if updated.Status != "BOOKED" || repo.expectedStatus != "DRAFT" || repo.targetStatus != "BOOKED" {
		t.Fatalf("transition result = %#v, from = %s, to = %s", updated, repo.expectedStatus, repo.targetStatus)
	}
	if repo.statusEvent == nil || repo.statusEvent.AuditEvent().Action != "order.status.transition" || repo.statusEvent.ToStatus != "BOOKED" {
		t.Fatalf("status event = %#v", repo.statusEvent)
	}
}

func TestOrderStatusChangedEventFields(t *testing.T) {
	organizationID := uuid.New()
	actorID := uuid.New()
	id := uuid.New()

	event := &OrderStatusChangedEvent{
		OrganizationID: organizationID,
		OrderID:        id,
		ActorID:        actorID,
		FromStatus:     "BOOKED",
		ToStatus:       "SHIPPED",
		OccurredAt:     time.Now().UTC(),
	}
	auditEvent := event.AuditEvent()
	if auditEvent.Action != "order.status.transition" ||
		auditEvent.OrganizationID == nil || *auditEvent.OrganizationID != organizationID ||
		auditEvent.UserID == nil || *auditEvent.UserID != actorID ||
		auditEvent.Details["order.id"] != id.String() ||
		auditEvent.Details["from_status"] != "BOOKED" ||
		auditEvent.Details["to_status"] != "SHIPPED" {
		t.Fatalf("audit event = %#v", auditEvent)
	}
}

var _ OrderRepo = (*orderRepoStub)(nil)
