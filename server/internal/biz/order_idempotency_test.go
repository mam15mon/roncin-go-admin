package biz

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// inlineTransactor 直接内联执行事务回调，供创建幂等用例单元测试使用。
type inlineTransactor struct{}

func (inlineTransactor) WithinTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func idempotentCreateInput(organizationID uuid.UUID) *Order {
	shippingLineID := uuid.Must(uuid.NewV7())
	directMode := SeaDocumentStructureDirect
	return &Order{
		OrganizationID: organizationID,
		IdempotencyKey: "replay-key-001",
		CustomerID:     uuid.Must(uuid.NewV7()), BusinessType: OrderBusinessSE,
		ShippingLineID: &shippingLineID,
		TradeDirection: OrderTradeExport, TradeTerm: OrderTradeFOB, PaymentTerm: OrderPaymentPrepaid,
		ServiceTypeIDs:     []uuid.UUID{uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())},
		CargoCategoryIDs:   []uuid.UUID{uuid.Must(uuid.NewV7())},
		Notes:              "重放保护",
		SeaMasterBillInput: &SeaMasterBillInput{MasterNo: "COSCO123456"},
		SeaDocumentInput:   &SeaOrderDocumentInput{DocumentStructure: &directMode},
	}
}

// buildExistingFromCreate 以创建请求为模板构造幂等键命中的既有订单，
// 载体字段与请求一致（ServiceTypeIDs 顺序打乱，验证集合比较与顺序无关）。
func buildExistingFromCreate(input *Order) *Order {
	existing := *input
	existing.ID = uuid.Must(uuid.NewV7())
	existing.OrderNo = "SE0009"
	existing.Version = 1
	existing.OrderDate = "2026-09-13T00:00:00Z"
	existing.ServiceTypeIDs = []uuid.UUID{input.ServiceTypeIDs[1], input.ServiceTypeIDs[0]}
	return &existing
}

func TestOrderCreateReplaysExistingOrderForSameIntent(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	existing := buildExistingFromCreate(idempotentCreateInput(organizationID))
	repo := &orderRepoStub{byIdempotencyKey: existing}
	usecase := NewOrderUsecase(repo, nil, &seaMasterBillRepoStub{}, nil, newReminderModeCreditControl(), inlineTransactor{})

	request := *existing
	request.OrderDate = ""
	request.SeaMasterBillInput = &SeaMasterBillInput{MasterNo: existing.SeaMasterBillInput.MasterNo}
	request.SeaDocumentInput = existing.SeaDocumentInput
	created, err := usecase.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), &request)
	if err != nil {
		t.Fatalf("同键同意图重放应返回既有订单: %v", err)
	}
	if created.ID != existing.ID {
		t.Fatalf("重放返回的订单 = %v, want 既有订单 %v", created.ID, existing.ID)
	}
	if repo.created != nil {
		t.Fatalf("重放不得再次创建订单: %#v", repo.created)
	}
}

func TestOrderCreateRejectsDifferentIntentWithSameKey(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	existing := buildExistingFromCreate(idempotentCreateInput(organizationID))
	repo := &orderRepoStub{byIdempotencyKey: existing}
	usecase := NewOrderUsecase(repo, nil, &seaMasterBillRepoStub{}, nil, newReminderModeCreditControl(), inlineTransactor{})

	request := *existing
	request.CustomerID = uuid.Must(uuid.NewV7())
	_, err := usecase.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), &request)
	if err != ErrOrderIdempotencyConflict {
		t.Fatalf("同键不同意图应返回幂等冲突, got %v", err)
	}
	if repo.created != nil {
		t.Fatalf("冲突请求不得创建订单: %#v", repo.created)
	}
}

func TestOrderCreateRejectsSameKeyWithDriftedContent(t *testing.T) {
	organizationID := uuid.Must(uuid.NewV7())
	existing := buildExistingFromCreate(idempotentCreateInput(organizationID))
	repo := &orderRepoStub{byIdempotencyKey: existing}
	usecase := NewOrderUsecase(repo, nil, &seaMasterBillRepoStub{}, nil, newReminderModeCreditControl(), inlineTransactor{})

	// 全量请求哈希覆盖此前清单式比对遗漏的字段：仅改货物描述也必须判为冲突。
	request := *existing
	request.GoodsDescription = "漂移后的货物描述"
	_, err := usecase.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), &request)
	if err != ErrOrderIdempotencyConflict {
		t.Fatalf("同键仅改货物描述应返回幂等冲突, got %v", err)
	}

	driftedContainers := *existing
	driftedContainers.ContainerRequests = []*OrderContainerRequest{{
		ContainerSpecID: uuid.Must(uuid.NewV7()), Quantity: 2,
	}}
	_, err = usecase.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), &driftedContainers)
	if err != ErrOrderIdempotencyConflict {
		t.Fatalf("同键仅改箱型箱量应返回幂等冲突, got %v", err)
	}

	driftedPersonnel := *existing
	driftedPersonnel.PersonnelAssignments = []*OrderPersonnel{{
		UserID: uuid.Must(uuid.NewV7()), OrganizationID: organizationID, Role: OrderPersonnelRoleOperator,
	}}
	_, err = usecase.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), &driftedPersonnel)
	if err != ErrOrderIdempotencyConflict {
		t.Fatalf("同键仅改岗位人员应返回幂等冲突, got %v", err)
	}

	driftedDestination := *existing
	destinationID := uuid.Must(uuid.NewV7())
	driftedDestination.DestinationLocationID = &destinationID
	_, err = usecase.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), &driftedDestination)
	if err != ErrOrderIdempotencyConflict {
		t.Fatalf("同键仅改目的地应返回幂等冲突, got %v", err)
	}

	driftedCutoff := *existing
	driftedCutoff.SICutoff = "2026-10-01T12:00:00Z"
	_, err = usecase.Create(context.Background(), organizationID, uuid.Must(uuid.NewV7()), &driftedCutoff)
	if err != ErrOrderIdempotencyConflict {
		t.Fatalf("同键仅改截关时间应返回幂等冲突, got %v", err)
	}
	if repo.created != nil {
		t.Fatalf("冲突请求不得创建订单: %#v", repo.created)
	}
}

func TestOrderCreateWithoutIdempotencyKeyKeepsDirectCreate(t *testing.T) {
	repo := &orderRepoStub{}
	usecase := NewOrderUsecase(repo, nil, &seaMasterBillRepoStub{}, nil, newReminderModeCreditControl(), nil)

	input := idempotentCreateInput(uuid.Must(uuid.NewV7()))
	input.IdempotencyKey = ""
	created, err := usecase.Create(context.Background(), uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7()), input)
	if err != nil {
		t.Fatalf("未提供幂等键时应保持直接创建行为: %v", err)
	}
	if repo.created == nil || created.ID != repo.created.ID {
		t.Fatalf("未提供幂等键时应走仓储直建: %#v", repo.created)
	}
}

func TestSameOrderCreateIntentIgnoresServerDefaultsAndSelectionOrder(t *testing.T) {
	input := idempotentCreateInput(uuid.Must(uuid.NewV7()))
	existing := buildExistingFromCreate(input)

	// 服务端默认值（order_date）与选择集合顺序不影响意图判定。
	if !sameOrderCreateIntent(existing, input) {
		t.Fatal("载体字段一致（选择集合顺序不同）应判定为同一意图")
	}

	otherCustomer := *input
	otherCustomer.CustomerID = uuid.Must(uuid.NewV7())
	if sameOrderCreateIntent(existing, &otherCustomer) {
		t.Fatal("委托客户不同应判定为意图冲突")
	}

	otherSelection := *input
	otherSelection.ServiceTypeIDs = []uuid.UUID{uuid.Must(uuid.NewV7()), uuid.Must(uuid.NewV7())}
	if sameOrderCreateIntent(existing, &otherSelection) {
		t.Fatal("服务类型集合不同应判定为意图冲突")
	}

	otherCount := *input
	otherCount.CargoCategoryIDs = nil
	if sameOrderCreateIntent(existing, &otherCount) {
		t.Fatal("货物类别数量不同应判定为意图冲突")
	}

	if sameOrderCreateIntent(nil, input) || sameOrderCreateIntent(existing, nil) {
		t.Fatal("空指针应判定为意图不一致")
	}
}

func TestNormalizeOrderTrimsAndValidatesIdempotencyKey(t *testing.T) {
	input := idempotentCreateInput(uuid.Must(uuid.NewV7()))
	input.IdempotencyKey = "  replay-key-001  "
	normalized, err := normalizeOrder(input, true)
	if err != nil {
		t.Fatalf("normalizeOrder() error = %v", err)
	}
	if normalized.IdempotencyKey != "replay-key-001" {
		t.Fatalf("幂等键应去空白, got %q", normalized.IdempotencyKey)
	}

	tooLong := *input
	tooLong.IdempotencyKey = strings.Repeat("k", 129)
	if _, err := normalizeOrder(&tooLong, true); err != ErrOrderInvalidArgument {
		t.Fatalf("超长幂等键应拒绝, got %v", err)
	}
}
