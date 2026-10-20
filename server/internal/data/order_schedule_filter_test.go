package data

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

func TestOrderScheduleFilterUsesTimestampBoundaries(t *testing.T) {
	from := time.Date(2026, time.October, 19, 16, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)
	for _, field := range []string{orderent.FieldEtd, orderent.FieldEta} {
		t.Run(field, func(t *testing.T) {
			db, mock, err := sqlmock.New()
			if err != nil {
				t.Fatal(err)
			}
			client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
			t.Cleanup(func() { _ = client.Close() })
			// 必须比较完整时刻；仅按文本日期比较会漏掉北京时间零点落在 UTC 前一天的记录。
			mock.ExpectQuery(`WHERE NULLIF\("orders"\."`+field+`", ''\)::timestamptz >= \$1 AND NULLIF\("orders"\."`+field+`", ''\)::timestamptz < \$2`).
				WithArgs(from, to).
				WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
			query := client.Order.Query()
			applyOrderStringDateRange(query, field, biz.OrderDateRange{From: &from, ToExclusive: &to})
			if count, err := query.Count(context.Background()); err != nil || count != 1 {
				t.Fatalf("船期筛选没有使用完整时间边界: count=%d，err=%v", count, err)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestOrderScheduleTimestampBoundsPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("需要专用PostgreSQL连接")
	}
	db, err := sql.Open("pgx", source)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	from := time.Date(2026, time.October, 19, 16, 0, 0, 0, time.UTC)
	to := from.AddDate(0, 0, 1)
	// 仅查询常量，不创建表、写入业务数据或依赖连接的会话时区。
	rows, err := db.QueryContext(context.Background(), `SELECT id FROM (VALUES
		(1, '2026-10-19T15:59:59Z'),
		(2, '2026-10-19T16:00:00Z'),
		(3, '2026-10-20T00:00:00+08:00'),
		(4, '2026-10-20T15:59:59Z'),
		(5, '2026-10-20T16:00:00Z'),
		(6, ''),
		(7, NULL)
	) AS schedule(id, value)
	WHERE NULLIF(value, '')::timestamptz >= $1
	AND NULLIF(value, '')::timestamptz < $2 ORDER BY id`, from, to)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	want := []int{2, 3, 4}
	var actual []int
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		actual = append(actual, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}
	if len(actual) != len(want) {
		t.Fatalf("北京时间当天筛选结果=%v，期望%v", actual, want)
	}
	for i, id := range want {
		if actual[i] != id {
			t.Fatalf("北京时间当天筛选结果=%v，期望%v", actual, want)
		}
	}
}
