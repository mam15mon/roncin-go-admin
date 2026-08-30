package data

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	numbersequenceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type orderPostgresFixture struct {
	t              *testing.T
	data           *Data
	organizationID uuid.UUID
	partnerID      uuid.UUID
	actorID        uuid.UUID
	suffix         string
}

type orderWriteResult struct {
	order *biz.Order
	err   error
}

func TestOrderCreateTransactionPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}
	data, cleanup, err := NewData(&conf.Data{Database: &conf.Data_Database{
		Driver:             "postgres",
		Source:             source,
		AutoMigrate:        true,
		MaxOpenConnections: 8,
		MaxIdleConnections: 8,
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	defer cleanup()

	t.Run("并发创建订单时发号不重号不丢号", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		operations := make([]func(context.Context) (*biz.Order, error), 0, 4)
		for range 4 {
			input := fixture.validInput()
			operations = append(operations, func(ctx context.Context) (*biz.Order, error) {
				return usecase.Create(ctx, fixture.organizationID, fixture.actorID, input)
			})
		}
		results := runOrderWritesConcurrently(operations...)
		numbers := make(map[string]struct{}, len(results))
		for index, result := range results {
			if result.err != nil {
				t.Fatalf("第 %d 个并发请求创建订单: %v", index+1, result.err)
			}
			if result.order == nil || result.order.OrderNo == "" {
				t.Fatalf("第 %d 个并发请求未返回订单号: %#v", index+1, result.order)
			}
			if result.order.Version != 1 || result.order.FlowStatus != biz.OrderFlowDraft {
				t.Fatalf("第 %d 个并发订单初始状态异常: version=%d flowStatus=%s", index+1, result.order.Version, result.order.FlowStatus)
			}
			numbers[result.order.OrderNo] = struct{}{}
		}
		if len(numbers) != len(results) {
			t.Fatalf("并发创建出现重复订单号: %v", numbers)
		}
		fixture.requireCommittedOrders(4, 4)
	})

	t.Run("引用校验失败回滚订单与单号序列", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		input := fixture.validInput()
		input.CustomerID = uuid.New()

		_, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, input)
		if !errors.Is(err, biz.ErrOrderCustomerInvalid) {
			t.Fatalf("无效客户创建订单的错误 = %v，期望 ErrOrderCustomerInvalid", err)
		}
		fixture.requireRolledBackState()
	})

	t.Run("并发修改草稿时版本冲突只有一个成功", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		created, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.validInput())
		if err != nil {
			t.Fatalf("创建草稿订单: %v", err)
		}
		first, second := fixture.validInput(), fixture.validInput()
		first.GoodsDescription = "并发修改一"
		second.GoodsDescription = "并发修改二"
		results := runOrderWritesConcurrently(
			func(ctx context.Context) (*biz.Order, error) {
				return usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version, first)
			},
			func(ctx context.Context) (*biz.Order, error) {
				return usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version, second)
			},
		)
		successes, conflicts := 0, 0
		for _, result := range results {
			switch {
			case result.err == nil && result.order != nil:
				successes++
				if result.order.Version != 2 {
					t.Fatalf("并发更新成功方版本 = %d，期望 2", result.order.Version)
				}
			case errors.Is(result.err, biz.ErrOrderStatusConflict):
				conflicts++
			default:
				t.Fatalf("并发更新返回非预期结果: order=%#v error=%v", result.order, result.err)
			}
		}
		if successes != 1 || conflicts != 1 {
			t.Fatalf("并发更新结果 success=%d conflict=%d，期望各 1", successes, conflicts)
		}
		fixture.requireDraftUpdateResult()
	})
}

func newOrderPostgresFixture(t *testing.T, data *Data) *orderPostgresFixture {
	t.Helper()
	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	organization, err := data.db.Organization.Create().
		SetCode("ORDER-TX-" + suffix).
		SetName("订单事务集成测试组织-" + suffix).
		SetKind("headquarters").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	fixture := &orderPostgresFixture{
		t: t, data: data, organizationID: organization.ID, suffix: suffix,
	}
	t.Cleanup(fixture.cleanup)

	partner, err := data.db.Partner.Create().
		SetOrganizationID(organization.ID).
		SetCode("CUSTOMER-" + suffix).
		SetLegalName("订单事务测试客户-" + suffix).
		SetNormalizedName("订单事务测试客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试往来单位: %v", err)
	}
	fixture.partnerID = partner.ID
	if _, err = data.db.PartnerRole.Create().
		SetPartnerID(partner.ID).
		SetRoleType(partnerroleent.RoleTypeCustomer).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试客户角色: %v", err)
	}

	actor, err := data.db.User.Create().
		SetDisplayName("订单事务测试用户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试用户: %v", err)
	}
	fixture.actorID = actor.ID
	if _, err = data.db.Membership.Create().
		SetUserID(actor.ID).
		SetOrganizationID(organization.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试成员关系: %v", err)
	}

	if _, err = data.db.NumberRule.Create().
		SetOrganizationID(organization.ID).
		SetDocumentType(numberruleent.DocumentTypeOrder).
		SetPrefix("SE-").
		SetDateFormat(numberruleent.DateFormatYyyyMMdd).
		SetSequenceLength(5).
		SetResetPolicy(numberruleent.ResetPolicyDaily).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试订单编号规则: %v", err)
	}
	return fixture
}

func (f *orderPostgresFixture) newUsecase() *biz.OrderUsecase {
	return biz.NewOrderUsecase(NewOrderRepo(f.data), NewBusinessTagRepo(f.data))
}

func (f *orderPostgresFixture) validInput() *biz.Order {
	return &biz.Order{
		CustomerID:     f.partnerID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: biz.OrderTradeExport,
		TradeTerm:      biz.OrderTradeFOB,
		PaymentTerm:    biz.OrderPaymentPrepaid,
	}
}

func runOrderWritesConcurrently(operations ...func(context.Context) (*biz.Order, error)) []orderWriteResult {
	start := make(chan struct{})
	results := make(chan orderWriteResult, len(operations))
	for _, operation := range operations {
		operation := operation
		go func() {
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			order, err := operation(ctx)
			results <- orderWriteResult{order: order, err: err}
		}()
	}
	close(start)
	collected := make([]orderWriteResult, 0, len(operations))
	for range operations {
		collected = append(collected, <-results)
	}
	return collected
}

func (f *orderPostgresFixture) requireCommittedOrders(wantOrders int, wantSequence int64) {
	f.t.Helper()
	ctx := context.Background()
	orders, err := f.data.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).All(ctx)
	if err != nil || len(orders) != wantOrders {
		f.t.Fatalf("已提交订单数 = %d，期望 %d，error=%v", len(orders), wantOrders, err)
	}
	lifecycleCount, err := f.data.db.OrderLifecycleEvent.Query().Where(orderlifecycleeventent.HasOrderWith(orderent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || lifecycleCount != wantOrders {
		f.t.Fatalf("已提交生命周期事件数 = %d，期望 %d，error=%v", lifecycleCount, wantOrders, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("order.create")).Count(ctx)
	if err != nil || auditCount != wantOrders {
		f.t.Fatalf("已提交订单审计数 = %d，期望 %d，error=%v", auditCount, wantOrders, err)
	}
	personnelCount, err := f.data.db.OrderPersonnel.Query().Where(orderpersonnelent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || personnelCount != wantOrders {
		f.t.Fatalf("已提交订单人员数 = %d，期望 %d（仅创建人），error=%v", personnelCount, wantOrders, err)
	}
	sequences, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).All(ctx)
	if err != nil || len(sequences) != 1 || sequences[0].CurrentValue != wantSequence {
		f.t.Fatalf("已提交订单序列 = %#v，期望当前值 %d，error=%v", sequences, wantSequence, err)
	}
}

func (f *orderPostgresFixture) requireRolledBackState() {
	f.t.Helper()
	ctx := context.Background()
	orderCount, err := f.data.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || orderCount != 0 {
		f.t.Fatalf("回滚后订单数 = %d，期望 0，error=%v", orderCount, err)
	}
	lifecycleCount, err := f.data.db.OrderLifecycleEvent.Query().Where(orderlifecycleeventent.HasOrderWith(orderent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || lifecycleCount != 0 {
		f.t.Fatalf("回滚后生命周期事件数 = %d，期望 0，error=%v", lifecycleCount, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || auditCount != 0 {
		f.t.Fatalf("回滚后审计数 = %d，期望 0，error=%v", auditCount, err)
	}
	sequenceCount, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || sequenceCount != 0 {
		f.t.Fatalf("回滚后订单序列数 = %d，期望 0，error=%v", sequenceCount, err)
	}
}

func (f *orderPostgresFixture) requireDraftUpdateResult() {
	f.t.Helper()
	ctx := context.Background()
	stored, err := f.data.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).Only(ctx)
	if err != nil {
		f.t.Fatalf("查询并发更新后的订单: %v", err)
	}
	if stored.Version != 2 {
		f.t.Fatalf("并发更新后订单版本 = %d，期望 2", stored.Version)
	}
	if stored.GoodsDescription != "并发修改一" && stored.GoodsDescription != "并发修改二" {
		f.t.Fatalf("并发更新后货物描述 = %q，期望为其中一个并发值", stored.GoodsDescription)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("order.update")).Count(ctx)
	if err != nil || auditCount != 1 {
		f.t.Fatalf("并发更新后 order.update 审计数 = %d，期望 1，error=%v", auditCount, err)
	}
}

func (f *orderPostgresFixture) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	steps := []struct {
		name string
		run  func() error
	}{
		{name: "订单人员", run: func() error {
			_, err := f.data.db.OrderPersonnel.Delete().Where(orderpersonnelent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "订单生命周期事件", run: func() error {
			_, err := f.data.db.OrderLifecycleEvent.Delete().Where(orderlifecycleeventent.HasOrderWith(orderent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "订单提成归属快照", run: func() error {
			_, err := f.data.db.OrderCommissionAttribution.Delete().Where(ordercommissionattributionent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "订单", run: func() error {
			_, err := f.data.db.Order.Delete().Where(orderent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "订单审计", run: func() error {
			_, err := f.data.db.AuditLog.Delete().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "订单序列", run: func() error {
			_, err := f.data.db.NumberSequence.Delete().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "订单编号规则", run: func() error {
			_, err := f.data.db.NumberRule.Delete().Where(numberruleent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "成员关系", run: func() error {
			_, err := f.data.db.Membership.Delete().Where(membershipent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "用户", run: func() error {
			_, err := f.data.db.User.Delete().Where(userent.IDEQ(f.actorID)).Exec(ctx)
			return err
		}},
		{name: "客户角色", run: func() error {
			_, err := f.data.db.PartnerRole.Delete().Where(partnerroleent.HasPartnerWith(partnerent.IDEQ(f.partnerID))).Exec(ctx)
			return err
		}},
		{name: "往来单位", run: func() error {
			_, err := f.data.db.Partner.Delete().Where(partnerent.IDEQ(f.partnerID)).Exec(ctx)
			return err
		}},
		{name: "组织", run: func() error {
			_, err := f.data.db.Organization.Delete().Where(organizationent.IDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			f.t.Errorf("清理%s失败: %v", step.name, err)
			return
		}
	}
}
