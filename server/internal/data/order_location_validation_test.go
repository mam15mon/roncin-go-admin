package data

import (
	"context"
	"database/sql/driver"
	"reflect"
	"regexp"
	"testing"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/DATA-DOG/go-sqlmock"
	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func TestOrderLocationValidationUsesUniqueReferences(t *testing.T) {
	first, second := uuid.New(), uuid.New()
	for _, location := range []struct {
		name     string
		table    string
		message  string
		validate func(context.Context, *ent.Tx, uuid.UUID, []uuid.UUID) error
	}{
		{"港口", "ports", "港口不存在或已停用，请刷新后重新选择", validatePortIDs},
		{"机场", "airports", "机场不存在或已停用，请刷新后重新选择", validateAirportIDs},
	} {
		t.Run(location.name, func(t *testing.T) {
			for _, tc := range []struct {
				name   string
				fields [4]*uuid.UUID
				unique []uuid.UUID
				count  int
				valid  bool
			}{
				{name: "空引用", valid: true},
				{name: "单个引用及nil", fields: [4]*uuid.UUID{nil, &first}, unique: []uuid.UUID{first}, count: 1, valid: true},
				{name: "起运与目的相同", fields: [4]*uuid.UUID{&first, &first}, unique: []uuid.UUID{first}, count: 1, valid: true},
				{name: "不同有效引用", fields: [4]*uuid.UUID{&first, &second}, unique: []uuid.UUID{first, second}, count: 2, valid: true},
				{name: "四字段相同", fields: [4]*uuid.UUID{&first, &first, &first, &first}, unique: []uuid.UUID{first}, count: 1, valid: true},
				{name: "重复中混入无效引用", fields: [4]*uuid.UUID{&first, &first, &second}, unique: []uuid.UUID{first, second}, count: 1},
				{name: "非nil零UUID仍拒绝", fields: [4]*uuid.UUID{&uuid.Nil}, unique: []uuid.UUID{uuid.Nil}},
			} {
				t.Run(tc.name, func(t *testing.T) {
					db, mock, err := sqlmock.New()
					if err != nil {
						t.Fatal(err)
					}
					client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
					t.Cleanup(func() { _ = client.Close() })
					data := &Data{db: client}
					mock.ExpectBegin()
					if len(tc.unique) > 0 {
						args := make([]driver.Value, 0, len(tc.unique)+1)
						for _, id := range tc.unique {
							args = append(args, id)
						}
						mock.ExpectQuery(regexp.QuoteMeta(`SELECT COUNT("` + location.table + `"."id") FROM "` + location.table + `" WHERE `)).
							WithArgs(args...).WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(tc.count))
					}
					if tc.valid {
						mock.ExpectCommit()
					} else {
						mock.ExpectRollback()
					}
					input := &biz.Order{OriginLocationID: tc.fields[0], DestinationLocationID: tc.fields[1], DischargeLocationID: tc.fields[2], TransitLocationID: tc.fields[3]}
					before := *input
					ids := nonNilUUIDs(input.OriginLocationID, input.DestinationLocationID, input.DischargeLocationID, input.TransitLocationID)
					beforeIDs := append([]uuid.UUID{}, ids...)
					err = data.WithTx(t.Context(), func(tx *ent.Tx) error {
						return location.validate(t.Context(), tx, uuid.New(), ids)
					})
					if tc.valid && err != nil {
						t.Fatalf("有效地点引用应通过: %v", err)
					}
					if !tc.valid {
						businessErr := kratoserrors.FromError(err)
						if businessErr.Code != 400 || businessErr.Reason != biz.ErrOrderInvalidArgument.Reason || businessErr.Message != location.message {
							t.Fatalf("无效地点提示不正确: %v", err)
						}
					}
					if !reflect.DeepEqual(input, &before) || !reflect.DeepEqual(ids, beforeIDs) {
						t.Fatal("地点校验不得修改原始字段或引用切片")
					}
					if err := mock.ExpectationsWereMet(); err != nil {
						t.Fatal(err)
					}
				})
			}
		})
	}
}
