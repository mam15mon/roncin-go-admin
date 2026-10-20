package data

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

// 使用正式迁移前态，验证时间戳差异不再被当成人工修改，成功修改审计仍被保留。
func TestInvoiceDefaultNumberRuleRepairPostgres(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, "20261020140000")
	t.Cleanup(cleanup)
	ctx := context.Background()
	createdAt := time.Date(2026, 8, 22, 0, 0, 0, 0, time.UTC)
	updatedAt := createdAt.AddDate(0, 0, 2)

	type testRule struct {
		name        string
		rule        *ent.NumberRule
		wantEnabled bool
	}
	var rules []testRule
	for _, tc := range []struct {
		name              string
		prefix            string
		enabled           bool
		auditResult       auditlog.Result
		otherOrganization bool
		wantEnabled       bool
	}{
		{name: "历史迁移改过时间的默认规则", wantEnabled: true},
		{name: "应用创建时间存在微小差异", wantEnabled: true},
		{name: "人工关闭的默认规则", auditResult: auditlog.ResultSuccess, wantEnabled: false},
		{name: "失败修改审计不影响默认规则", auditResult: auditlog.ResultFailure, wantEnabled: true},
		{name: "其他组织审计不得影响本组织", auditResult: auditlog.ResultSuccess, otherOrganization: true, wantEnabled: true},
		{name: "自定义规则保持关闭", prefix: "CUSTOM", wantEnabled: false},
		{name: "已启用规则保持原样", enabled: true, wantEnabled: true},
	} {
		org, err := data.db.Organization.Create().SetCode("INV-" + uuid.NewString()).SetName(tc.name).SetKind("company").SetBaseCurrency("CNY").Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		ruleUpdatedAt := updatedAt
		if tc.name == "应用创建时间存在微小差异" {
			ruleUpdatedAt = createdAt.Add(time.Microsecond)
		}
		rule, err := data.db.NumberRule.Create().SetOrganizationID(org.ID).SetDocumentType(numberrule.DocumentTypeInvoice).SetPrefix(tc.prefix).SetDateFormat(numberrule.DateFormatYyyyMMdd).SetSequenceLength(5).SetResetPolicy(numberrule.ResetPolicyDaily).SetEnabled(tc.enabled).SetCreatedAt(createdAt).SetUpdatedAt(ruleUpdatedAt).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if tc.auditResult != "" {
			auditOrganizationID := org.ID
			if tc.otherOrganization {
				auditOrganizationID = uuid.New()
			}
			details, err := json.Marshal(map[string]string{"number_rule.id": rule.ID.String()})
			if err != nil {
				t.Fatal(err)
			}
			if _, err := data.db.AuditLog.Create().SetOrganizationID(auditOrganizationID).SetAction("number_rule.update").SetResult(tc.auditResult).SetDetails(details).Save(ctx); err != nil {
				t.Fatal(err)
			}
		}
		rules = append(rules, testRule{name: tc.name, rule: rule, wantEnabled: tc.wantEnabled})
	}
	if err := migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	for _, tc := range rules {
		t.Run(tc.name, func(t *testing.T) {
			got, err := data.db.NumberRule.Get(ctx, tc.rule.ID)
			if err != nil {
				t.Fatal(err)
			}
			if got.Enabled != tc.wantEnabled {
				t.Fatalf("启用状态 = %v，期望 %v", got.Enabled, tc.wantEnabled)
			}
			if !got.CreatedAt.Equal(tc.rule.CreatedAt) || got.Prefix != tc.rule.Prefix || got.SequenceLength != tc.rule.SequenceLength || got.DateFormat != tc.rule.DateFormat || got.ResetPolicy != tc.rule.ResetPolicy {
				t.Fatal("迁移改写了启用状态以外的编号配置")
			}
			if tc.rule.Enabled == tc.wantEnabled && !got.UpdatedAt.Equal(tc.rule.UpdatedAt) {
				t.Fatal("非目标规则的更新时间被改写")
			}
		})
	}
	beforeRepeat, err := data.db.NumberRule.Get(ctx, rules[0].rule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if err := migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations")); err != nil {
		t.Fatal(err)
	}
	afterRepeat, err := data.db.NumberRule.Get(ctx, rules[0].rule.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !beforeRepeat.UpdatedAt.Equal(afterRepeat.UpdatedAt) {
		t.Fatal("重复迁移不应再次改写规则")
	}
}
