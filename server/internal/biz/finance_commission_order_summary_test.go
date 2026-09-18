package biz

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/shopspring/decimal"
)

// orderSummaryRepoStub 只覆盖批量摘要查询：捕获用例构造的可见范围，验证
// 隐私裁剪发生在 SQL 作用域构造阶段，不触发数据库。
type orderSummaryRepoStub struct {
	CommissionRepo
	receivedScopes []OrderCommissionSummaryScope
	result         map[uuid.UUID]*OrderCommissionSummary
	resultErr      error
}

func (s *orderSummaryRepoStub) ListOrderSummaries(_ context.Context, scopes []OrderCommissionSummaryScope) (map[uuid.UUID]*OrderCommissionSummary, error) {
	s.receivedScopes = scopes
	if s.resultErr != nil {
		return nil, s.resultErr
	}
	return s.result, nil
}

// newOrderSummaryPrincipal 构造当前主体：withCommissionRead 决定是否持有
// system.finance.commission.read；readRootOrganizationID 为空时权限范围
// 覆盖全部节点组织，否则以组织树范围只覆盖该根。
func newOrderSummaryPrincipal(userID, currentOrganizationID uuid.UUID, withCommissionRead bool, nodes ...uuid.UUID) *Principal {
	principal := &Principal{UserID: userID, Organization: Organization{ID: currentOrganizationID}}
	for _, organizationID := range nodes {
		principal.OrganizationNodes = append(principal.OrganizationNodes, OrganizationScopeNode{ID: organizationID})
	}
	if withCommissionRead {
		principal.RoleGrants = []RoleGrant{{RoleCode: "commission-reader", DataScope: DataScopeOrganizationTree, Permissions: map[string]struct{}{access.FinanceCommissionRead: {}}}}
	}
	return principal
}

func orderSummaryStub(stub *orderSummaryRepoStub) *CommissionUsecase {
	return NewCommissionUsecase(stub, nil, nil)
}

// 用例必须按目标组织逐个解析 system.finance.commission.read：持有则组织级
// 全员汇总，否则 SQL 层固定本人；普通员工作用域不得缺失 EmployeeID。
func TestBuildOrderListSummariesVisibilityScopes(t *testing.T) {
	organization := uuid.New()
	otherOrganization := uuid.New()
	order := uuid.New()
	otherOrder := uuid.New()
	callerID := uuid.New()

	t.Run("普通员工仅本人", func(t *testing.T) {
		stub := &orderSummaryRepoStub{result: map[uuid.UUID]*OrderCommissionSummary{}}
		caller := newOrderSummaryPrincipal(callerID, organization, false, organization)
		summaries, err := orderSummaryStub(stub).BuildOrderListSummaries(context.Background(), caller, []OrderCommissionSummaryTarget{{OrderID: order, OrganizationID: organization}})
		if err != nil {
			t.Fatalf("构建本人摘要失败: %v", err)
		}
		if len(stub.receivedScopes) != 1 {
			t.Fatalf("作用域数量 = %d，期望 1", len(stub.receivedScopes))
		}
		scope := stub.receivedScopes[0]
		if scope.Visibility != OrderCommissionVisibilityEmployee {
			t.Fatalf("无组织级读取权限应得到本人视图: %s", scope.Visibility)
		}
		if scope.EmployeeID != callerID {
			t.Fatalf("本人作用域必须固定调用者 employee_id: %s", scope.EmployeeID)
		}
		if scope.OrganizationID != organization || len(scope.OrderIDs) != 1 || scope.OrderIDs[0] != order {
			t.Fatalf("作用域范围不符: %+v", scope)
		}
		summary := summaries[order]
		if summary == nil || summary.Visibility != OrderCommissionVisibilityEmployee {
			t.Fatalf("本人视图摘要缺失或可见模式不符: %+v", summary)
		}
	})

	t.Run("组织级财务整票", func(t *testing.T) {
		stub := &orderSummaryRepoStub{result: map[uuid.UUID]*OrderCommissionSummary{}}
		caller := newOrderSummaryPrincipal(callerID, organization, true, organization)
		if _, err := orderSummaryStub(stub).BuildOrderListSummaries(context.Background(), caller, []OrderCommissionSummaryTarget{{OrderID: order, OrganizationID: organization}}); err != nil {
			t.Fatalf("构建组织级摘要失败: %v", err)
		}
		scope := stub.receivedScopes[0]
		if scope.Visibility != OrderCommissionVisibilityOrganization {
			t.Fatalf("持有组织级读取权限应得到全员视图: %s", scope.Visibility)
		}
		if scope.EmployeeID != uuid.Nil {
			t.Fatalf("组织级视图不得限定 employee_id: %s", scope.EmployeeID)
		}
	})

	t.Run("同一主体在不同组织权限不同", func(t *testing.T) {
		stub := &orderSummaryRepoStub{result: map[uuid.UUID]*OrderCommissionSummary{}}
		// 当前工作区在授权组织，权限树只覆盖该组织：另一组织退回本人视图。
		caller := newOrderSummaryPrincipal(callerID, organization, true, organization, otherOrganization)
		_, err := orderSummaryStub(stub).BuildOrderListSummaries(context.Background(), caller, []OrderCommissionSummaryTarget{
			{OrderID: order, OrganizationID: organization},
			{OrderID: otherOrder, OrganizationID: otherOrganization},
		})
		if err != nil {
			t.Fatalf("构建混合权限摘要失败: %v", err)
		}
		if len(stub.receivedScopes) != 2 {
			t.Fatalf("作用域数量 = %d，期望 2", len(stub.receivedScopes))
		}
		scopeByOrganization := map[uuid.UUID]OrderCommissionSummaryScope{}
		for _, scope := range stub.receivedScopes {
			scopeByOrganization[scope.OrganizationID] = scope
		}
		if scopeByOrganization[organization].Visibility != OrderCommissionVisibilityOrganization {
			t.Fatalf("授权组织应为组织级视图: %+v", scopeByOrganization[organization])
		}
		otherScope := scopeByOrganization[otherOrganization]
		if otherScope.Visibility != OrderCommissionVisibilityEmployee || otherScope.EmployeeID != callerID {
			t.Fatalf("未授权组织应退回仅本人视图: %+v", otherScope)
		}
	})
}

// 本人无记录时必须返回本人空态，而不是缺失；仓储返回的事实集合原样透传，
// 用例不得捏造或放大任何事实。
func TestBuildOrderListSummariesEmptyStateAndPassthrough(t *testing.T) {
	organization := uuid.New()
	order := uuid.New()
	caller := newOrderSummaryPrincipal(uuid.New(), organization, false, organization)

	t.Run("仓储无记录回填本人空态", func(t *testing.T) {
		stub := &orderSummaryRepoStub{result: map[uuid.UUID]*OrderCommissionSummary{}}
		summaries, err := orderSummaryStub(stub).BuildOrderListSummaries(context.Background(), caller, []OrderCommissionSummaryTarget{{OrderID: order, OrganizationID: organization}})
		if err != nil {
			t.Fatalf("构建摘要失败: %v", err)
		}
		summary := summaries[order]
		if summary == nil {
			t.Fatal("本人无记录也应返回空态摘要")
		}
		if summary.Visibility != OrderCommissionVisibilityEmployee {
			t.Fatalf("空态可见模式不符: %s", summary.Visibility)
		}
		if summary.HasDraftCommission || summary.HasConfirmedCommission || summary.HasPaidCommission || summary.HasPendingDecrease || summary.HasExpectedOpportunity {
			t.Fatalf("空态不得携带任何事实: %+v", summary)
		}
	})

	t.Run("仓储事实原样透传", func(t *testing.T) {
		stub := &orderSummaryRepoStub{result: map[uuid.UUID]*OrderCommissionSummary{order: {
			Visibility: OrderCommissionVisibilityEmployee, BaseCurrency: "CNY",
			HasDraftCommission: true, DraftCommissionCount: 2, DraftCommissionAmount: decimal.RequireFromString("120.00000000"),
		}}}
		summaries, err := orderSummaryStub(stub).BuildOrderListSummaries(context.Background(), caller, []OrderCommissionSummaryTarget{{OrderID: order, OrganizationID: organization}})
		if err != nil {
			t.Fatalf("构建摘要失败: %v", err)
		}
		summary := summaries[order]
		if summary.DraftCommissionCount != 2 || !summary.DraftCommissionAmount.Equal(decimal.RequireFromString("120.00000000")) {
			t.Fatalf("仓储事实被篡改: %+v", summary)
		}
	})
}

// 输入校验与可见模式一致性：重复目标去重、非法参数拒绝、仓储返回的可见模式
// 与请求不符时整体失败，防止越权聚合数据混入本人视图。
func TestBuildOrderListSummariesValidation(t *testing.T) {
	organization := uuid.New()
	order := uuid.New()
	caller := newOrderSummaryPrincipal(uuid.New(), organization, false, organization)

	t.Run("非法参数拒绝", func(t *testing.T) {
		stub := &orderSummaryRepoStub{}
		usecase := orderSummaryStub(stub)
		if _, err := usecase.BuildOrderListSummaries(context.Background(), nil, []OrderCommissionSummaryTarget{{OrderID: order, OrganizationID: organization}}); err != ErrCommissionInvalid {
			t.Fatalf("缺少主体应拒绝: %v", err)
		}
		if _, err := usecase.BuildOrderListSummaries(context.Background(), caller, nil); err != ErrCommissionInvalid {
			t.Fatalf("空目标应拒绝: %v", err)
		}
		oversized := make([]OrderCommissionSummaryTarget, MaxListPageSize+1)
		for index := range oversized {
			oversized[index] = OrderCommissionSummaryTarget{OrderID: uuid.New(), OrganizationID: organization}
		}
		if _, err := usecase.BuildOrderListSummaries(context.Background(), caller, oversized); err != ErrCommissionInvalid {
			t.Fatalf("超过列表分页上限应拒绝: %v", err)
		}
		if _, err := usecase.BuildOrderListSummaries(context.Background(), caller, []OrderCommissionSummaryTarget{{OrderID: uuid.Nil, OrganizationID: organization}}); err != ErrCommissionInvalid {
			t.Fatalf("空订单 ID 应拒绝: %v", err)
		}
	})

	t.Run("重复目标去重", func(t *testing.T) {
		stub := &orderSummaryRepoStub{result: map[uuid.UUID]*OrderCommissionSummary{}}
		if _, err := orderSummaryStub(stub).BuildOrderListSummaries(context.Background(), caller, []OrderCommissionSummaryTarget{
			{OrderID: order, OrganizationID: organization},
			{OrderID: order, OrganizationID: organization},
		}); err != nil {
			t.Fatalf("构建摘要失败: %v", err)
		}
		if len(stub.receivedScopes) != 1 || len(stub.receivedScopes[0].OrderIDs) != 1 {
			t.Fatalf("重复目标应去重: %+v", stub.receivedScopes)
		}
	})

	t.Run("仓储可见模式漂移拒绝", func(t *testing.T) {
		stub := &orderSummaryRepoStub{result: map[uuid.UUID]*OrderCommissionSummary{order: {
			Visibility: OrderCommissionVisibilityOrganization, HasPaidCommission: true, PaidCommissionCount: 1,
		}}}
		if _, err := orderSummaryStub(stub).BuildOrderListSummaries(context.Background(), caller, []OrderCommissionSummaryTarget{{OrderID: order, OrganizationID: organization}}); err != ErrCommissionInvalid {
			t.Fatalf("本人请求混入组织级聚合应整体失败: %v", err)
		}
	})
}
