package data

import (
	"context"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionrule"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
	"github.com/shopspring/decimal"
)

func setupTestCommissionRepo(t *testing.T) (*commissionRepo, sqlmock.Sqlmock) {
	t.Helper()
	db, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("创建 sqlmock 失败: %v", err)
	}
	driver := entsql.OpenDB(dialect.Postgres, db)
	client := ent.NewClient(ent.Driver(driver))
	t.Cleanup(func() {
		_ = client.Close()
		_ = db.Close()
	})
	return &commissionRepo{data: &Data{db: client, sqlDB: db}}, mock
}

func TestCommissionCandidateEmployeePredicatesUseDatabaseFilteringAndPagination(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	source := &commissionCalculationSource{
		rule:     &ent.FinanceCommissionRule{PersonnelRole: financecommissionrule.PersonnelRoleSALES},
		orderIDs: []uuid.UUID{uuid.New()},
	}
	query := repo.data.db.User.Query().Where(commissionCandidateEmployeePredicates(uuid.New(), source, "zhangsan")...)
	mock.ExpectQuery(`SELECT COUNT\("users"\."id"\) FROM "users".*order_commission_attributions.*orders.*order_fees.*"username".*"display_name".*"search_keywords"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT "users"\..*FROM "users".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(user.Columns))

	if _, err := query.Clone().Count(context.Background()); err != nil {
		t.Fatalf("统计提成候选失败: %v", err)
	}
	if _, err := query.Order(user.ByDisplayName(), user.ByUsername(), user.ByID()).Offset(20).Limit(20).All(context.Background()); err != nil {
		t.Fatalf("分页查询提成候选失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("提成候选未使用数据库筛选和分页: %v", err)
	}
}

func TestCalculateCommissionFromSourceUsesSharedBatchData(t *testing.T) {
	organizationID := uuid.New()
	orderID := uuid.New()
	employeeID := uuid.New()
	customer := &ent.Partner{ID: uuid.New(), Code: "CUS001", LegalName: "测试客户"}
	settlementParty := &ent.Partner{ID: uuid.New(), LegalName: "测试结算单位"}
	orderItem := &ent.Order{
		ID: orderID, OrderNo: "SE20260001", OrderDate: "2026-08-28", CustomerID: customer.ID,
		Edges: ent.OrderEdges{Customer: customer},
	}
	receivable := &ent.OrderFee{
		ID: uuid.New(), OrderID: orderID, SettlementPartyID: settlementParty.ID,
		Direction: orderfee.DirectionRECEIVABLE, Status: orderfee.StatusCONFIRMED,
		FeeCode: "FREIGHT", FeeName: "海运费", Currency: "CNY", TotalAmount: "1000", ExchangeRate: "1",
		BaseCurrency: "CNY", BaseCurrencyAmount: "1000", ExpenseDate: "2026-08-28", Version: 1,
		Edges: ent.OrderFeeEdges{SettlementParty: settlementParty},
	}
	payable := &ent.OrderFee{
		ID: uuid.New(), OrderID: orderID, SettlementPartyID: settlementParty.ID,
		Direction: orderfee.DirectionPAYABLE, Status: orderfee.StatusCONFIRMED,
		FeeCode: "COST", FeeName: "成本", Currency: "CNY", TotalAmount: "400", ExchangeRate: "1",
		BaseCurrency: "CNY", BaseCurrencyAmount: "400", ExpenseDate: "2026-08-28", Version: 1,
		Edges: ent.OrderFeeEdges{SettlementParty: settlementParty},
	}
	source := &commissionCalculationSource{
		organizationID: organizationID,
		verification:   &ent.FinanceVerification{ID: uuid.New(), VerificationNo: "VR20260001"},
		rule: &ent.FinanceCommissionRule{
			ID: uuid.New(), Name: "销售提成", PersonnelRole: financecommissionrule.PersonnelRoleSALES,
			CalculationBasis: financecommissionrule.CalculationBasisREALIZED_PROFIT, Version: 1,
		},
		rate:          decimal.NewFromInt(10),
		baseCurrency:  "CNY",
		orderIDs:      []uuid.UUID{orderID},
		orderRealized: map[uuid.UUID]decimal.Decimal{orderID: decimal.NewFromInt(500)},
		orderByID:     map[uuid.UUID]*ent.Order{orderID: orderItem},
		feesByOrder:   map[uuid.UUID][]*ent.OrderFee{orderID: {receivable, payable}},
	}
	employee := &ent.User{ID: employeeID, DisplayName: "张三", Enabled: true}
	attributions := []*ent.OrderCommissionAttribution{{
		ID: uuid.New(), OrganizationID: organizationID, OrderID: orderID, CustomerID: customer.ID,
		SourceAssignmentID: uuid.New(), EmployeeID: employeeID, EmployeeName: "张三",
		PersonnelRole: ordercommissionattribution.PersonnelRoleSALES, AttributedAt: time.Now(),
	}}

	calculation, err := calculateCommissionFromSource(source, employee, attributions)
	if err != nil {
		t.Fatalf("使用批量上下文计算提成失败: %v", err)
	}
	if calculation.RealizedRevenue.StringFixed(8) != "500.00000000" || calculation.AllocatedCost.StringFixed(8) != "200.00000000" || calculation.RealizedProfit.StringFixed(8) != "300.00000000" || calculation.CommissionAmount.StringFixed(8) != "30.00000000" {
		t.Fatalf("批量提成计算结果不符合预期: %#v", calculation)
	}
	if calculation.CustomerCount != 1 || calculation.OrderCount != 1 || calculation.FeeCount != 2 || calculation.SourceFingerprint == "" {
		t.Fatalf("批量提成计算摘要不符合预期: %#v", calculation)
	}
}

func TestCommissionRepoListEmployeesUsesKeywordAndDatabasePagination(t *testing.T) {
	repo, mock := setupTestCommissionRepo(t)
	mock.ExpectQuery(`SELECT COUNT\("users"\."id"\) FROM "users".*EXISTS.*FROM "memberships".*"username".*"display_name".*"search_keywords"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT "users"\..*FROM "users".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(user.Columns))

	result, err := repo.ListEmployees(context.Background(), uuid.New(), biz.SelectorListOptions{
		Page: 2, PageSize: 20, Keyword: "zhangsan",
	})
	if err != nil {
		t.Fatalf("查询提成员工失败: %v", err)
	}
	if result.Total != 0 || len(result.Items) != 0 || result.Page != 2 || result.PageSize != 20 {
		t.Fatalf("提成员工分页结果不符合预期: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("提成员工未使用数据库搜索和分页: %v", err)
	}
}
