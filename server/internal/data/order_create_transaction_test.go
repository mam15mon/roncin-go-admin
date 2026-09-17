package data

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerassignment"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
)

func TestOrderCommissionSnapshotRejectsMissingRoles(t *testing.T) {
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
	mock.ExpectBegin()
	tx, err := client.Tx(t.Context())
	if err != nil {
		t.Fatalf("开启事务失败: %v", err)
	}
	mock.ExpectQuery(`SELECT .* FROM "partner_assignments"`).
		WillReturnRows(sqlmock.NewRows(partnerassignment.Columns))
	err = snapshotOrderCommissionAttributions(t.Context(), tx, uuid.New(), uuid.New(), uuid.New(), time.Now())
	if err == nil || !strings.Contains(err.Error(), "销售、操作、客服") {
		t.Fatalf("缺失全部提成岗位时应显式拒绝并列出岗位，实际: %v", err)
	}
	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatalf("事务回滚失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("快照查询及回滚未符合预期: %v", err)
	}
}

func TestOrderCreateRollsBackAllocatedNumberWhenValidationFails(t *testing.T) {
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
	repo := &orderRepo{data: &Data{db: client, sqlDB: db}}
	organizationID, actorID, customerID := uuid.New(), uuid.New(), uuid.New()
	ruleID, sequenceID := uuid.New(), uuid.New()
	now := time.Now().UTC()

	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "organizations"\."id" FROM "organizations".*"kind".*"enabled"`).
		WillReturnRows(sqlmock.NewRows([]string{organization.FieldID}).AddRow(organizationID))
	mock.ExpectQuery(`SELECT "number_rules"\..*FROM "number_rules".*FOR UPDATE`).
		WillReturnRows(sqlmock.NewRows(numberrule.Columns).AddRow(
			ruleID, now, now, organizationID, "order", "", "yyyyMMdd", 5, "daily", true,
		))
	mock.ExpectQuery(`SELECT "number_sequences"\..*FROM "number_sequences"`).
		WillReturnRows(sqlmock.NewRows(numbersequence.Columns).AddRow(
			sequenceID, now, now, ruleID, now.Format("20060102"), 1,
		))
	mock.ExpectExec(`UPDATE "number_sequences" SET .*"current_value".*WHERE "id" =`).
		WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT .* FROM "number_sequences" WHERE "id" =`).
		WillReturnRows(sqlmock.NewRows(numbersequence.Columns).AddRow(
			sequenceID, now, now, ruleID, now.Format("20060102"), 2,
		))
	mock.ExpectQuery(`SELECT .* FROM "partners".*FOR SHARE`).
		WillReturnRows(sqlmock.NewRows(partner.Columns))
	mock.ExpectRollback()

	_, err = repo.Create(context.Background(), organizationID, actorID, &biz.Order{
		CustomerID:     customerID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: biz.OrderTradeExport,
		TradeTerm:      biz.OrderTradeFOB,
		PaymentTerm:    biz.OrderPaymentPrepaid,
	}, &biz.AuditEvent{Details: map[string]string{}})
	if err != biz.ErrOrderCustomerInvalid {
		t.Fatalf("创建订单错误 = %v，期望 ErrOrderCustomerInvalid", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("号码分配与订单校验未处于同一回滚事务: %v", err)
	}
}

func TestValidateOrderReferencesPreservesPartnerQueryError(t *testing.T) {
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

	mock.ExpectBegin()
	tx, err := client.Tx(t.Context())
	if err != nil {
		t.Fatalf("开启测试事务失败: %v", err)
	}
	databaseErr := errors.New("partner query failed")
	organizationID, customerID := uuid.New(), uuid.New()
	now := time.Now().UTC()
	mock.ExpectQuery(`SELECT .* FROM "partners".*FOR SHARE`).
		WillReturnRows(sqlmock.NewRows(partner.Columns).AddRow(
			customerID, now, now, organizationID, nil, "测试客户", "测试客户", nil, "", true, false, "测试客户",
		))
	mock.ExpectQuery(`SELECT .* FROM "partner_roles"`).WillReturnError(databaseErr)

	err = validateOrderReferences(t.Context(), tx, organizationID, &biz.Order{CustomerID: customerID}, nil)
	if !errors.Is(err, databaseErr) {
		t.Fatalf("合作方查询错误被改写: got %v, want %v", err, databaseErr)
	}

	mock.ExpectRollback()
	if err := tx.Rollback(); err != nil {
		t.Fatalf("回滚测试事务失败: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v", err)
	}
}

func TestValidateOrderPartnerReferencesBlacklistMatrix(t *testing.T) {
	for _, testCase := range []struct {
		name        string
		build       func(customerID, targetID uuid.UUID) (*biz.Order, *ent.Order)
		targetRole  partnerrole.RoleType
		targetLabel string
		wantErr     bool
	}{
		{
			name: "新建订单拒绝黑名单客户",
			build: func(_, targetID uuid.UUID) (*biz.Order, *ent.Order) {
				return &biz.Order{CustomerID: targetID}, nil
			},
			targetRole: partnerrole.RoleTypeCustomer, targetLabel: "客户", wantErr: true,
		},
		{
			name: "新建订单拒绝黑名单订舱代理",
			build: func(customerID, targetID uuid.UUID) (*biz.Order, *ent.Order) {
				return &biz.Order{CustomerID: customerID, BookingAgentID: &targetID}, nil
			},
			targetRole: partnerrole.RoleTypeSupplier, targetLabel: "供应商", wantErr: true,
		},
		{
			name: "新建订单拒绝黑名单国外代理",
			build: func(customerID, targetID uuid.UUID) (*biz.Order, *ent.Order) {
				return &biz.Order{CustomerID: customerID, ForeignAgentID: &targetID}, nil
			},
			targetRole: partnerrole.RoleTypeForeignAgent, targetLabel: "国外代理", wantErr: true,
		},
		{
			name: "新建订单拒绝黑名单船务代理",
			build: func(customerID, targetID uuid.UUID) (*biz.Order, *ent.Order) {
				return &biz.Order{CustomerID: customerID, ShippingAgentID: &targetID}, nil
			},
			targetRole: partnerrole.RoleTypeSupplier, targetLabel: "供应商", wantErr: true,
		},
		{
			name: "草稿换入黑名单订舱代理时拒绝",
			build: func(customerID, targetID uuid.UUID) (*biz.Order, *ent.Order) {
				originalID := uuid.New()
				return &biz.Order{CustomerID: customerID, BookingAgentID: &targetID}, &ent.Order{CustomerID: customerID, BookingAgentID: &originalID}
			},
			targetRole: partnerrole.RoleTypeSupplier, targetLabel: "供应商", wantErr: true,
		},
		{
			name: "草稿保留原黑名单订舱代理时允许",
			build: func(customerID, targetID uuid.UUID) (*biz.Order, *ent.Order) {
				return &biz.Order{CustomerID: customerID, BookingAgentID: &targetID}, &ent.Order{CustomerID: customerID, BookingAgentID: &targetID}
			},
			targetRole: partnerrole.RoleTypeSupplier,
		},
		{
			name: "草稿清空原黑名单订舱代理时允许",
			build: func(customerID, targetID uuid.UUID) (*biz.Order, *ent.Order) {
				return &biz.Order{CustomerID: customerID}, &ent.Order{CustomerID: customerID, BookingAgentID: &targetID}
			},
			targetRole: partnerrole.RoleTypeSupplier,
		},
		{
			name: "同一档案客户正常但供应商黑名单时仍拒绝订舱代理",
			build: func(_, targetID uuid.UUID) (*biz.Order, *ent.Order) {
				return &biz.Order{CustomerID: targetID, BookingAgentID: &targetID}, nil
			},
			targetRole: partnerrole.RoleTypeSupplier, targetLabel: "供应商", wantErr: true,
		},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatalf("创建 sqlmock 失败: %v", err)
			}
			driver := entsql.OpenDB(dialect.Postgres, db)
			client := ent.NewClient(ent.Driver(driver))
			t.Cleanup(func() { _ = client.Close(); _ = db.Close() })
			mock.ExpectBegin()
			tx, err := client.Tx(t.Context())
			if err != nil {
				t.Fatalf("开启测试事务失败: %v", err)
			}
			organizationID, customerID, targetID := uuid.New(), uuid.New(), uuid.New()
			input, existing := testCase.build(customerID, targetID)
			now := time.Now().UTC()
			partnerRows := sqlmock.NewRows(partner.Columns)
			seenPartners := map[uuid.UUID]bool{}
			for _, partnerID := range []uuid.UUID{input.CustomerID, targetID} {
				if partnerID == uuid.Nil || seenPartners[partnerID] || (input.BookingAgentID == nil && input.ForeignAgentID == nil && input.ShippingAgentID == nil && partnerID == targetID && input.CustomerID != targetID) {
					continue
				}
				seenPartners[partnerID] = true
				partnerRows.AddRow(partnerID, now, now, organizationID, nil, "测试单位", "测试单位", nil, "", true, false, "测试单位")
			}
			mock.ExpectQuery(`SELECT .* FROM "partners".*FOR SHARE`).WillReturnRows(partnerRows)

			roleRows := sqlmock.NewRows(partnerrole.Columns).
				AddRow(uuid.New(), now, now, input.CustomerID, "customer", true, false, nil, nil, nil)
			if input.BookingAgentID != nil || input.ForeignAgentID != nil || input.ShippingAgentID != nil {
				roleRows.AddRow(uuid.New(), now, now, targetID, string(testCase.targetRole), true, true, "严重违约", now, uuid.New())
			} else if input.CustomerID == targetID && testCase.targetRole == partnerrole.RoleTypeCustomer {
				roleRows = sqlmock.NewRows(partnerrole.Columns).
					AddRow(uuid.New(), now, now, targetID, "customer", true, true, "严重违约", now, uuid.New())
			}
			mock.ExpectQuery(`SELECT .* FROM "partner_roles"`).WillReturnRows(roleRows)

			err = validateOrderPartnerReferences(t.Context(), tx, organizationID, input, existing)
			if testCase.wantErr {
				if kratoserrors.Reason(err) != "ORDER_PARTNER_ROLE_BLACKLISTED" || !strings.Contains(err.Error(), testCase.targetLabel+"已列入黑名单") {
					t.Fatalf("黑名单错误 = %v", err)
				}
			}
			if !testCase.wantErr && err != nil {
				t.Fatalf("原关联保留或清空不应被拒绝: %v", err)
			}
			mock.ExpectRollback()
			if err := tx.Rollback(); err != nil {
				t.Fatalf("回滚测试事务失败: %v", err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("未满足 sqlmock 期望: %v", err)
			}
		})
	}
}
