package data

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

func TestChengduCompanyNameEnMigrationPostgres(t *testing.T) {
	blank, manual := " \t\n", "MANUAL CHENGDU CO., LTD"
	cases := []struct {
		name       string
		value      *string
		nonCompany bool
	}{
		{name: "补齐空值"},
		{name: "补齐空白", value: &blank},
		{name: "保留人工名称", value: &manual},
		{name: "非公司不补齐", nonCompany: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data, cleanup := getIntegrationDataBeforeMigration(t, "20261104110000")
			t.Cleanup(cleanup)
			ctx := context.Background()
			stamp := time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)
			create := data.db.Organization.Create().SetCode("CD").SetName("成都公司").SetNillableNameEn(tc.value).SetCreatedAt(stamp).SetUpdatedAt(stamp)
			if tc.nonCompany {
				parent, err := data.db.Organization.Create().SetCode("PARENT").SetName("部门上级公司").SetKind(organizationent.KindCompany).SetBaseCurrency("CNY").Save(ctx)
				if err != nil {
					t.Fatal(err)
				}
				create.SetKind(organizationent.KindDepartment).SetParentID(parent.ID)
			} else {
				create.SetKind(organizationent.KindCompany).SetBaseCurrency("CNY")
			}
			initial, err := create.Save(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if err = migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
				t.Fatal(err)
			}
			after, err := data.db.Organization.Get(ctx, initial.ID)
			if err != nil {
				t.Fatal(err)
			}
			if tc.nonCompany {
				if after.NameEn != nil {
					t.Fatal("非公司被写入成都抬头")
				}
			} else {
				expected := "RONCIN (CHENGDU) SUPPLY CHAIN MANAGEMENT CO., LTD"
				if tc.value != nil && *tc.value == manual {
					expected = manual
				}
				if after.NameEn == nil || *after.NameEn != expected {
					t.Fatalf("英文名=%v，期望=%s", after.NameEn, expected)
				}
				if !strings.Contains(after.SearchKeywords, expected) || !strings.Contains(after.SearchKeywords, initial.SearchKeywords) {
					t.Fatal("英文检索缺失或覆盖已有检索键")
				}
			}
			if !after.UpdatedAt.Equal(stamp) {
				t.Fatal("补名改变已有业务更新时间")
			}
			if err = migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
				t.Fatal(err)
			}
			repeated, err := data.db.Organization.Get(ctx, initial.ID)
			if err != nil {
				t.Fatal(err)
			}
			if repeated.SearchKeywords != after.SearchKeywords || !repeated.UpdatedAt.Equal(after.UpdatedAt) {
				t.Fatal("重复迁移重写检索或更新时间")
			}
			if (repeated.NameEn == nil) != (after.NameEn == nil) || (after.NameEn != nil && *repeated.NameEn != *after.NameEn) {
				t.Fatal("重复迁移改写英文名")
			}
		})
	}
}
