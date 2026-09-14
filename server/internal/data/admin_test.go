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
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
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
	workspaceID := uuid.New()

	// 工作台范围先一次加载组织树（总部工作台 = 全树启用组织）。
	mock.ExpectQuery(`SELECT "organizations"."id", "organizations"."parent_id", "organizations"."kind", "organizations"."code", "organizations"."name", "organizations"."base_currency", "organizations"."enabled" FROM "organizations"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id", "kind", "code", "name", "base_currency", "enabled"}).
			AddRow(workspaceID, nil, "headquarters", "HQ", "总部", "CNY", true))
	mock.ExpectQuery(`SELECT COUNT\("users"\."id"\) FROM "users".*EXISTS.*"memberships".*"organization_id"`).
		WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
	mock.ExpectQuery(`SELECT "users"\."id".*FROM "users".*EXISTS.*ORDER BY "users"\."username", "users"\."id" LIMIT 20 OFFSET 20`).
		WillReturnRows(sqlmock.NewRows(userent.Columns))

	result, err := repo.ListUsers(context.Background(), workspaceID, biz.AdminUserListOptions{
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

func TestMembershipToUserBuildsOrganizationSummaries(t *testing.T) {
	primaryOrgID := uuid.New()
	secondaryOrgID := uuid.New()
	account := &ent.User{
		ID:          uuid.New(),
		Username:    "alice",
		DisplayName: "Alice",
		Edges: ent.UserEdges{
			Memberships: []*ent.Membership{
				{
					Primary: false,
					Edges: ent.MembershipEdges{
						Organization: &ent.Organization{ID: secondaryOrgID, Name: "B 公司", Enabled: true},
					},
				},
				{
					Primary: true,
					Edges: ent.MembershipEdges{
						Organization: &ent.Organization{ID: primaryOrgID, Name: "A 公司", Enabled: true},
					},
				},
			},
		},
	}
	item := &ent.Membership{
		Enabled: true,
		Edges:   ent.MembershipEdges{User: account},
	}

	result := membershipToUser(item)

	if len(result.Organizations) != 2 {
		t.Fatalf("组织摘要数量不符合预期: %d", len(result.Organizations))
	}
	if !result.Organizations[0].Primary || result.Organizations[0].Name != "A 公司" {
		t.Fatalf("主组织应排在首位: %#v", result.Organizations[0])
	}
	if result.Organizations[0].OrganizationID != primaryOrgID || result.Organizations[1].OrganizationID != secondaryOrgID {
		t.Fatalf("组织 ID 映射不符合预期: %#v", result.Organizations)
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
