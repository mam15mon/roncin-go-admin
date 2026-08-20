package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type orderRepoStub struct {
	created        *Order
	createdNumber  string
	transitioned   *Order
	expectedStatus string
	targetStatus   string
}

func (s *orderRepoStub) Get(context.Context, uuid.UUID, uuid.UUID) (*Order, error) {
	return nil, ErrOrderNotFound
}

func (s *orderRepoStub) List(_ context.Context, _ uuid.UUID, options OrderListOptions) (*OrderList, error) {
	return &OrderList{Page: options.Page, PageSize: options.PageSize}, nil
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

func (s *orderRepoStub) TransitionStatus(_ context.Context, organizationID, id uuid.UUID, expectedStatus, targetStatus, _ string, _ uuid.UUID) (*Order, error) {
	s.expectedStatus = expectedStatus
	s.targetStatus = targetStatus
	s.transitioned = &Order{ID: id, OrganizationID: organizationID, Status: targetStatus}
	return s.transitioned, nil
}

func TestOrderCreateUsesNumberRuleAndAudits(t *testing.T) {
	repo := &orderRepoStub{}
	configRepo := &orderConfigRepoStub{allocatedRule: &NumberRule{Prefix: "ORD", DateFormat: DateFormatNone, SequenceLength: 4, ResetPolicy: ResetPolicyNever}, allocatedSequence: 7}
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
	if created.OrderNo != "ORD0007" || repo.createdNumber != "ORD0007" || configRepo.lastAllocDocType != DocumentTypeOrder {
		t.Fatalf("created order = %#v, allocated document type = %s", created, configRepo.lastAllocDocType)
	}
	if len(audit.events) != 1 || audit.events[0].Action != "order.create" || audit.events[0].Details["order.no"] != "ORD0007" {
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
	if len(audit.events) != 1 || audit.events[0].Action != "order.status.transition" || audit.events[0].Details["to_status"] != "BOOKED" {
		t.Fatalf("audit events = %#v", audit.events)
	}
}

var _ OrderRepo = (*orderRepoStub)(nil)
