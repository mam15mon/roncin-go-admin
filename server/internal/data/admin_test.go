package data

import (
	"context"
	"errors"
	"regexp"
	"strings"
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

// TestAdminRepoListUsersEnabledFilter 验证 enabled 过滤三种取值（不传/true/false）
// 与关键字搜索的 AND 组合：设置时追加启用状态谓词（ent 将 = true/= false 优化为
// 裸列 / NOT 裸列，无绑定参数），不传时查询不含该谓词，关键字条件在两种情况下
// 均生效。
func TestAdminRepoListUsersEnabledFilter(t *testing.T) {
	enabledTrue := true
	enabledFalse := false
	tests := []struct {
		name          string
		enabled       *bool
		wantPredicate string
	}{
		{name: "不传enabled不过滤但关键字仍生效", enabled: nil, wantPredicate: ""},
		{name: "enabled=true与关键字AND过滤在职用户", enabled: &enabledTrue, wantPredicate: `) AND "users"."enabled"`},
		{name: "enabled=false与关键字AND过滤离职用户", enabled: &enabledFalse, wantPredicate: `) AND NOT "users"."enabled"`},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			// 记录实际执行的 SQL，用于断言「不传时无 enabled 过滤」的否定场景。
			var executedSQLs []string
			recorder := sqlmock.QueryMatcherFunc(func(expectedSQL, actualSQL string) error {
				executedSQLs = append(executedSQLs, actualSQL)
				return sqlmock.QueryMatcherRegexp.Match(expectedSQL, actualSQL)
			})
			db, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(recorder))
			if err != nil {
				t.Fatalf("创建 sqlmock 失败: %v", err)
			}
			driverConn := entsql.OpenDB(dialect.Postgres, db)
			client := ent.NewClient(ent.Driver(driverConn))
			t.Cleanup(func() {
				_ = client.Close()
				_ = db.Close()
			})
			repo := NewAdminRepo(&Data{db: client, sqlDB: db})
			workspaceID := uuid.New()

			mock.ExpectQuery(`SELECT "organizations"."id", "organizations"."parent_id", "organizations"."kind", "organizations"."code", "organizations"."name", "organizations"."base_currency", "organizations"."enabled" FROM "organizations"`).
				WillReturnRows(sqlmock.NewRows([]string{"id", "parent_id", "kind", "code", "name", "base_currency", "enabled"}).
					AddRow(workspaceID, nil, "headquarters", "HQ", "总部", "CNY", true))
			if test.wantPredicate != "" {
				mock.ExpectQuery(`SELECT COUNT\("users"\."id"\) FROM "users".*ILIKE.*`+regexp.QuoteMeta(test.wantPredicate)).
					WithArgs(workspaceID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectQuery(`SELECT "users"\."id".*ILIKE.*`+regexp.QuoteMeta(test.wantPredicate)).
					WithArgs(workspaceID, sqlmock.AnyArg(), sqlmock.AnyArg(), sqlmock.AnyArg()).
					WillReturnRows(sqlmock.NewRows(userent.Columns))
			} else {
				mock.ExpectQuery(`SELECT COUNT\("users"\."id"\) FROM "users".*ILIKE`).
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
				mock.ExpectQuery(`SELECT "users"\."id".*ILIKE`).
					WillReturnRows(sqlmock.NewRows(userent.Columns))
			}

			result, listErr := repo.ListUsers(context.Background(), workspaceID, biz.AdminUserListOptions{
				Page:     1,
				PageSize: 20,
				Keyword:  "Alice",
				Enabled:  test.enabled,
			})
			if listErr != nil {
				t.Fatalf("查询用户列表失败: %v", listErr)
			}
			if result.Total != 0 || len(result.Items) != 0 {
				t.Fatalf("空列表结果不符合预期: total=%d items=%d", result.Total, len(result.Items))
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatalf("数据库查询未按预期执行: %v", err)
			}
			// SELECT 列表固定包含 enabled 列，只有 AND 接在 WHERE 尾部的谓词才代表过滤。
			enabledPredicate := regexp.MustCompile(regexp.QuoteMeta(`) AND `) + `(NOT )?"users"\."enabled"`)
			hasFilteredQuery := false
			for _, query := range executedSQLs {
				if strings.Contains(query, "ILIKE") && enabledPredicate.MatchString(query) {
					hasFilteredQuery = true
					break
				}
			}
			if test.wantPredicate != "" && !hasFilteredQuery {
				t.Fatalf("enabled 过滤应与关键字以 AND 组合出现在同一查询中: %v", executedSQLs)
			}
			if test.wantPredicate == "" && hasFilteredQuery {
				t.Fatalf("不传 enabled 时查询不应包含启用状态过滤: %v", executedSQLs)
			}
		})
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
