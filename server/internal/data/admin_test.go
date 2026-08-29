package data

import (
	"context"
	"errors"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

func TestAdminRepoListUsersUsesDatabaseFilteringAndPagination(t *testing.T) {
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
	repo := NewAdminRepo(&Data{db: client, sqlDB: db})

	mock.ExpectQuery(`SELECT COUNT\("memberships"\."id"\) FROM "memberships".*EXISTS.*FROM "users".*"username"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT "memberships"\..*FROM "memberships".*ORDER BY.*LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(membership.Columns))

	result, err := repo.ListUsers(context.Background(), uuid.New(), biz.AdminUserListOptions{
		Page:     2,
		PageSize: 20,
		Keyword:  "Alice",
	})
	if err != nil {
		t.Fatalf("查询用户列表失败: %v", err)
	}
	if result.Total != 0 || len(result.Items) != 0 {
		t.Fatalf("空列表结果不符合预期: total=%d items=%d", result.Total, len(result.Items))
	}
	if result.Page != 2 || result.PageSize != 20 {
		t.Fatalf("分页信息不符合预期: page=%d pageSize=%d", result.Page, result.PageSize)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("数据库查询未按预期执行: %v", err)
	}
}

func TestAdminRepoUpdateOrganizationAuditErrorRollsBack(t *testing.T) {
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
	repo := NewAdminRepo(&Data{db: client, sqlDB: db})
	organizationID := uuid.New()
	now := time.Now()
	mock.ExpectBegin()
	mock.ExpectQuery(`SELECT "currencies"\."id" FROM "currencies"`).WithArgs("CNY").WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(uuid.New()))
	mock.ExpectExec(`UPDATE "organizations"`).WillReturnResult(sqlmock.NewResult(0, 1))
	mock.ExpectQuery(`SELECT "id", "created_at".*FROM "organizations"`).WithArgs(organizationID).WillReturnRows(
		sqlmock.NewRows(organizationent.Columns).AddRow(organizationID, now, now, "HQ", "新名称", "headquarters", nil, true, "CNY", "新名称"),
	)
	mock.ExpectExec(`INSERT INTO "audit_logs"`).WillReturnError(errors.New("写入审计失败"))
	mock.ExpectRollback()

	result, repoErr := repo.UpdateOrganization(context.Background(), organizationID, &biz.AdminOrganization{
		ID:           organizationID,
		Name:         "新名称",
		Kind:         biz.OrganizationKindHeadquarters,
		Enabled:      true,
		BaseCurrency: "CNY",
	}, &biz.AuditEvent{
		Action:  "admin.organization.update",
		Result:  "success",
		Details: map[string]string{"value": "HQ", "resource_id": organizationID.String()},
	})
	if repoErr == nil {
		t.Fatal("审计写入失败时应返回错误")
	}
	if result != nil {
		t.Fatalf("审计写入失败时不应返回组织: %#v", result)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("未满足 sqlmock 期望: %v；仓储错误: %v", err, repoErr)
	}
}
