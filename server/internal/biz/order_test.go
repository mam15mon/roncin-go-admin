package biz

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type orderRepoStub struct {
	created                    *Order
	createdNumber              string
	createdAudit               *AuditEvent
	updatedAudit               *AuditEvent
	referenceCheck             *OrderReferenceCheck
	referenceMatch             *OrderReferenceMatch
	transitioned               *Order
	expectedVersion            uint64
	targetStatus               OrderFlowStatus
	statusEvent                *OrderStatusChangedEvent
	terminationExpectedVersion uint64
	terminationTarget          OrderTerminationStatus
	terminationType            *OrderTerminationType
	terminationReason          string
	terminationEvent           *OrderLifecycleChangedEvent
	closureReadiness           *OrderClosureReadiness
	closureExpectedVersion     uint64
	closureTarget              OrderClosureStatus
	closureReason              string
	closureEvent               *OrderLifecycleChangedEvent
	hasContainers              bool
	current                    *Order
}

func (s *orderRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*Order, error) {
	if s.current != nil {
		return s.current, nil
	}
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

func (s *orderRepoStub) ListPersonnelOptions(_ context.Context, _ uuid.UUID, options SelectorListOptions) (*PagedList[*OrderPersonnelOption], error) {
	return &PagedList[*OrderPersonnelOption]{Page: options.Page, PageSize: options.PageSize}, nil
}

func (s *orderRepoStub) HasContainers(context.Context, uuid.UUID, uuid.UUID) (bool, error) {
	return s.hasContainers, nil
}

func (s *orderRepoStub) ListConsolidationSummaries(context.Context, uuid.UUID, uuid.UUID) ([]*OrderConsolidationSummary, error) {
	return nil, nil
}

func (s *orderRepoStub) Create(_ context.Context, organizationID, _ uuid.UUID, number string, input *Order, audit *AuditEvent) (*Order, error) {
	s.created = input
	s.createdNumber = number
	s.createdAudit = audit
	input.ID = uuid.New()
	input.OrganizationID = organizationID
	input.OrderNo = number
	input.FlowStatus = OrderFlowDraft
	input.TerminationStatus = OrderTerminationActive
	input.ClosureStatus = OrderClosureOpen
	input.Version = 1
	return input, nil
}

func (s *orderRepoStub) UpdateDraft(_ context.Context, organizationID, id uuid.UUID, expectedVersion uint64, input *Order, audit *AuditEvent) (*Order, error) {
	s.updatedAudit = audit
	input.ID = id
	input.OrganizationID = organizationID
	input.Version = expectedVersion + 1
	return input, nil
}

func (s *orderRepoStub) TransitionStatus(_ context.Context, organizationID, id uuid.UUID, expectedVersion uint64, targetStatus OrderFlowStatus, _ string, _ uuid.UUID, event *OrderStatusChangedEvent) (*Order, error) {
	s.expectedVersion = expectedVersion
	s.targetStatus = targetStatus
	s.statusEvent = event
	s.transitioned = &Order{ID: id, OrganizationID: organizationID, FlowStatus: targetStatus, Version: expectedVersion + 1}
	return s.transitioned, nil
}

func (s *orderRepoStub) TransitionTermination(_ context.Context, organizationID, id uuid.UUID, expectedVersion uint64, target OrderTerminationStatus, terminationType *OrderTerminationType, reason string, _ uuid.UUID, event *OrderLifecycleChangedEvent) (*Order, error) {
	s.terminationExpectedVersion = expectedVersion
	s.terminationTarget = target
	s.terminationType = terminationType
	s.terminationReason = reason
	s.terminationEvent = event
	return &Order{ID: id, OrganizationID: organizationID, TerminationStatus: target, ClosureStatus: OrderClosureOpen, Version: expectedVersion + 1}, nil
}

func (s *orderRepoStub) ClosureReadiness(context.Context, uuid.UUID, uuid.UUID) (*OrderClosureReadiness, error) {
	if s.closureReadiness != nil {
		return s.closureReadiness, nil
	}
	return &OrderClosureReadiness{}, nil
}

func (s *orderRepoStub) TransitionClosure(_ context.Context, organizationID, id uuid.UUID, expectedVersion uint64, target OrderClosureStatus, reason string, _ uuid.UUID, event *OrderLifecycleChangedEvent) (*Order, error) {
	s.closureExpectedVersion = expectedVersion
	s.closureTarget = target
	s.closureReason = reason
	s.closureEvent = event
	return &Order{ID: id, OrganizationID: organizationID, ClosureStatus: target, Version: expectedVersion + 1}, nil
}

func TestOrderCreateUsesNumberRuleAndAudits(t *testing.T) {
	repo := &orderRepoStub{}
	configRepo := &orderConfigRepoStub{allocatedRule: &NumberRule{DateFormat: DateFormatNone, SequenceLength: 4, ResetPolicy: ResetPolicyNever}, allocatedSequence: 7}
	audit := &auditRepoStub{}
	usecase := NewOrderUsecase(repo, NewOrderConfigUsecase(configRepo, audit), audit)
	organizationID := uuid.New()
	actorID := uuid.New()
	customerID := uuid.New()
	personnelUserID := uuid.New()
	created, err := usecase.Create(context.Background(), organizationID, actorID, &Order{
		CustomerID: customerID, BusinessType: OrderBusinessSE,
		TradeDirection: OrderTradeExport, TradeTerm: OrderTradeFOB, PaymentTerm: OrderPaymentPrepaid,
		ServiceTypeIDs: []uuid.UUID{uuid.New()}, CargoCategoryIDs: []uuid.UUID{uuid.New()},
		PersonnelAssignments: []*OrderPersonnel{{UserID: personnelUserID, OrganizationID: organizationID, Role: OrderPersonnelRoleOperator}},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if created.OrderNo != "SE0007" || repo.createdNumber != "SE0007" || configRepo.lastAllocDocType != DocumentTypeOrder {
		t.Fatalf("created order = %#v, allocated document type = %s", created, configRepo.lastAllocDocType)
	}
	if repo.createdAudit == nil || repo.createdAudit.Action != "order.create" || repo.createdAudit.Details["order.no"] != "SE0007" {
		t.Fatalf("create audit = %#v", repo.createdAudit)
	}
	if len(repo.created.PersonnelAssignments) != 1 || repo.created.PersonnelAssignments[0].Notification == nil || repo.created.PersonnelAssignments[0].Notification.RecipientUserID != personnelUserID {
		t.Fatalf("personnel notification = %#v", repo.created.PersonnelAssignments)
	}
}

func TestOrderRejectsInvalidAggregateAndDraftRollback(t *testing.T) {
	audit := &auditRepoStub{}
	configRepo := &orderConfigRepoStub{allocatedRule: &NumberRule{Prefix: "ORD", DateFormat: DateFormatNone, SequenceLength: 4, ResetPolicy: ResetPolicyNever}, allocatedSequence: 1}
	usecase := NewOrderUsecase(&orderRepoStub{current: &Order{BusinessType: OrderBusinessSE, FlowStatus: OrderFlowBooked, TerminationStatus: OrderTerminationActive, ClosureStatus: OrderClosureOpen, Version: 1}}, NewOrderConfigUsecase(configRepo, audit), audit)
	organizationID := uuid.New()
	actorID := uuid.New()
	duplicateID := uuid.New()

	_, err := usecase.Create(context.Background(), organizationID, actorID, &Order{
		CustomerID: uuid.New(), BusinessType: OrderBusinessSE,
		TradeDirection: OrderTradeExport, TradeTerm: OrderTradeFOB, PaymentTerm: OrderPaymentPrepaid,
		CargoCategoryIDs: []uuid.UUID{duplicateID, duplicateID},
	})
	if err != ErrOrderInvalidArgument {
		t.Fatalf("duplicate cargo categories error = %v, want ErrOrderInvalidArgument", err)
	}
	_, err = usecase.TransitionStatus(context.Background(), organizationID, actorID, uuid.New(), 1, OrderFlowDraft, "rollback")
	if err != ErrOrderStatusInvalid {
		t.Fatalf("draft rollback error = %v, want ErrOrderStatusInvalid", err)
	}
}

func TestOrderRejectsNegativeEntrustedCargoMeasurement(t *testing.T) {
	negative := -0.001
	input := &Order{
		CustomerID: uuid.New(), BusinessType: OrderBusinessSE,
		TradeDirection: OrderTradeExport, TradeTerm: OrderTradeFOB, PaymentTerm: OrderPaymentPrepaid,
		TotalGrossWeightKg: &negative,
	}
	if _, err := normalizeOrder(input, false); err != ErrOrderInvalidArgument {
		t.Fatalf("负数委托毛重 error = %v, want ErrOrderInvalidArgument", err)
	}
	input.TotalGrossWeightKg = nil
	input.TotalVolumeCbm = &negative
	if _, err := normalizeOrder(input, false); err != ErrOrderInvalidArgument {
		t.Fatalf("负数委托体积 error = %v, want ErrOrderInvalidArgument", err)
	}
}

func TestOrderNormalizesBusinessFieldsAndRequiresCompleteCargoValue(t *testing.T) {
	input := &Order{
		CustomerID: uuid.New(), BusinessType: OrderBusinessSE,
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
		CustomerID: uuid.New(), BusinessType: OrderBusinessSE,
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

func TestOrderBreakBulkRejectsContainerPlanAndVGM(t *testing.T) {
	breakBulk := OrderShipmentBreakBulk
	base := Order{
		CustomerID: uuid.New(), BusinessType: OrderBusinessSE,
		TradeDirection: OrderTradeExport, TradeTerm: OrderTradeFOB, PaymentTerm: OrderPaymentPrepaid,
		ShipmentType: &breakBulk,
	}

	withContainerPlan := base
	withContainerPlan.ContainerRequests = []*OrderContainerRequest{{ContainerSpecID: uuid.New(), Quantity: 1}}
	if _, err := normalizeOrder(&withContainerPlan, false); err != ErrOrderInvalidArgument {
		t.Fatalf("散杂订单携带箱量计划 error = %v, want ErrOrderInvalidArgument", err)
	}

	withVGM := base
	withVGM.VGMCutoff = "2026-08-26T12:00:00+08:00"
	if _, err := normalizeOrder(&withVGM, false); err != ErrOrderInvalidArgument {
		t.Fatalf("散杂订单携带 VGM 截止时间 error = %v, want ErrOrderInvalidArgument", err)
	}

	if _, err := normalizeOrder(&base, false); err != nil {
		t.Fatalf("不含集装箱数据的散杂订单 error = %v", err)
	}
}

func TestOrderUpdateRejectsChangingContainerOrderToNonFCL(t *testing.T) {
	repo := &orderRepoStub{hasContainers: true}
	usecase := NewOrderUsecase(repo, nil, nil)
	breakBulk := OrderShipmentBreakBulk
	input := &Order{
		CustomerID: uuid.New(), BusinessType: OrderBusinessSE,
		TradeDirection: OrderTradeExport, TradeTerm: OrderTradeFOB, PaymentTerm: OrderPaymentPrepaid,
		ShipmentType: &breakBulk,
	}

	_, err := usecase.UpdateDraft(context.Background(), uuid.New(), uuid.New(), uuid.New(), 1, input)
	if err != ErrOrderContainerShipmentType {
		t.Fatalf("已有实际箱时切换到散杂 error = %v, want ErrOrderContainerShipmentType", err)
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

func TestOrderTransitionValidatesEdgeAndAudits(t *testing.T) {
	repo := &orderRepoStub{current: &Order{BusinessType: OrderBusinessSE, FlowStatus: OrderFlowDraft, TerminationStatus: OrderTerminationActive, ClosureStatus: OrderClosureOpen, Version: 1}}
	audit := &auditRepoStub{}
	usecase := NewOrderUsecase(repo, NewOrderConfigUsecase(&orderConfigRepoStub{}, audit), audit)
	organizationID := uuid.New()
	actorID := uuid.New()
	id := uuid.New()

	updated, err := usecase.TransitionStatus(context.Background(), organizationID, actorID, id, 1, OrderFlowBooked, "  已订舱  ")
	if err != nil {
		t.Fatalf("TransitionStatus() error = %v", err)
	}
	if updated.FlowStatus != OrderFlowBooked || repo.expectedVersion != 1 || repo.targetStatus != OrderFlowBooked {
		t.Fatalf("transition result = %#v, version = %d, to = %s", updated, repo.expectedVersion, repo.targetStatus)
	}
	if repo.statusEvent == nil || repo.statusEvent.AuditEvent().Action != "order.flow.transition" || repo.statusEvent.ToStatus != OrderFlowBooked {
		t.Fatalf("status event = %#v", repo.statusEvent)
	}
}

func TestOrderTerminationTransitionRequiresReasonAndValidEdge(t *testing.T) {
	terminationType := OrderTerminationCustomsReturn
	repo := &orderRepoStub{current: &Order{BusinessType: OrderBusinessSE, FlowStatus: OrderFlowSpaceAllocated, TerminationStatus: OrderTerminationActive, ClosureStatus: OrderClosureOpen, Version: 4}}
	usecase := NewOrderUsecase(repo, nil, nil)
	organizationID := uuid.New()
	actorID := uuid.New()
	id := uuid.New()

	updated, err := usecase.TransitionTermination(context.Background(), organizationID, actorID, id, 4, OrderTerminationTerminating, &terminationType, "  海关要求退关  ")
	if err != nil {
		t.Fatalf("TransitionTermination() error = %v", err)
	}
	if updated.TerminationStatus != OrderTerminationTerminating || repo.terminationExpectedVersion != 4 || repo.terminationTarget != OrderTerminationTerminating || repo.terminationReason != "海关要求退关" || repo.terminationType == nil || *repo.terminationType != terminationType {
		t.Fatalf("termination result = %#v, repo = %#v", updated, repo)
	}
	if repo.terminationEvent == nil || repo.terminationEvent.AuditEvent().Action != "order.termination.transition" || repo.terminationEvent.FromStatus != string(OrderTerminationActive) {
		t.Fatalf("termination event = %#v", repo.terminationEvent)
	}

	_, err = usecase.TransitionTermination(context.Background(), organizationID, actorID, id, 4, OrderTerminationTerminated, &terminationType, "跳过处理中")
	if err != ErrOrderTerminationInvalid {
		t.Fatalf("direct termination error = %v, want ErrOrderTerminationInvalid", err)
	}
	_, err = usecase.TransitionTermination(context.Background(), organizationID, actorID, id, 4, OrderTerminationTerminating, &terminationType, " ")
	if err != ErrOrderTerminationInvalid {
		t.Fatalf("empty reason error = %v, want ErrOrderTerminationInvalid", err)
	}
}

func TestOrderClosureRequiresTerminalBusinessAndNoBlockers(t *testing.T) {
	organizationID := uuid.New()
	actorID := uuid.New()
	id := uuid.New()

	blockedCases := []struct {
		name      string
		readiness *OrderClosureReadiness
	}{
		{name: "主流程未到终点", readiness: &OrderClosureReadiness{FlowStatus: OrderFlowCustomsDeclarationArranged, TerminationStatus: OrderTerminationActive, ClosureStatus: OrderClosureOpen}},
		{name: "存在活动异常", readiness: &OrderClosureReadiness{FlowStatus: OrderFlowDocumentReleased, TerminationStatus: OrderTerminationActive, ClosureStatus: OrderClosureOpen, HasActiveException: true}},
		{name: "存在未出账费用", readiness: &OrderClosureReadiness{FlowStatus: OrderFlowDocumentReleased, TerminationStatus: OrderTerminationActive, ClosureStatus: OrderClosureOpen, HasUnbilledOrderFees: true}},
	}
	for _, testCase := range blockedCases {
		t.Run(testCase.name, func(t *testing.T) {
			usecase := NewOrderUsecase(&orderRepoStub{closureReadiness: testCase.readiness}, nil, nil)
			_, err := usecase.TransitionClosure(context.Background(), organizationID, actorID, id, 8, OrderClosureClosed, "确认结案")
			if err != ErrOrderClosureBlocked {
				t.Fatalf("TransitionClosure() error = %v, want ErrOrderClosureBlocked", err)
			}
		})
	}

	repo := &orderRepoStub{closureReadiness: &OrderClosureReadiness{FlowStatus: OrderFlowSpaceAllocated, TerminationStatus: OrderTerminationTerminated, ClosureStatus: OrderClosureOpen}}
	usecase := NewOrderUsecase(repo, nil, nil)
	updated, err := usecase.TransitionClosure(context.Background(), organizationID, actorID, id, 8, OrderClosureClosed, "  退关费用已处理  ")
	if err != nil {
		t.Fatalf("terminated order closure error = %v", err)
	}
	if updated.ClosureStatus != OrderClosureClosed || repo.closureExpectedVersion != 8 || repo.closureTarget != OrderClosureClosed || repo.closureReason != "退关费用已处理" {
		t.Fatalf("closure result = %#v, repo = %#v", updated, repo)
	}
	if repo.closureEvent == nil || repo.closureEvent.AuditEvent().Action != "order.closure.transition" || repo.closureEvent.FromStatus != string(OrderClosureOpen) {
		t.Fatalf("closure event = %#v", repo.closureEvent)
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
		FromStatus:     OrderFlowBooked,
		ToStatus:       OrderFlowSpaceAllocated,
		OccurredAt:     time.Now().UTC(),
	}
	auditEvent := event.AuditEvent()
	if auditEvent.Action != "order.flow.transition" ||
		auditEvent.OrganizationID == nil || *auditEvent.OrganizationID != organizationID ||
		auditEvent.UserID == nil || *auditEvent.UserID != actorID ||
		auditEvent.Details["order.id"] != id.String() ||
		auditEvent.Details["from_status"] != "BOOKED" ||
		auditEvent.Details["to_status"] != "SPACE_ALLOCATED" {
		t.Fatalf("audit event = %#v", auditEvent)
	}
}

var _ OrderRepo = (*orderRepoStub)(nil)
