package data

import (
	"context"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
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
