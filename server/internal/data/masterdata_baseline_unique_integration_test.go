package data

// 集成测试覆盖存储三型的唯一性物理约束与业务错误映射（prd.md §6.2）：
// A 型 master_data_items 同 (kind, code) 第二行插入被数据库唯一索引拒绝并映射
// ErrMasterDataCodeExists；B 型 ports 两条 NULL 基线行同 un_locode 被部分唯一索引
// 物理拒绝并映射 ErrIndustryReferenceCodeExist。

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestMasterDataGlobalUniqueConflictPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	repo := NewMasterDataRepo(data)
	audit := func() *biz.AuditEvent {
		return &biz.AuditEvent{Action: "master_data.create", Result: "success", Details: map[string]string{}}
	}

	first, err := repo.Create(ctx, uuid.Nil, &biz.MasterDataItem{Kind: biz.MasterDataKindContainerSpec, Code: "XXGP", Name: "首个同码箱型", Source: "manual", SortOrder: 10}, audit())
	if err != nil {
		t.Fatalf("首个同码主数据创建失败: %v", err)
	}
	_, err = repo.Create(ctx, uuid.Nil, &biz.MasterDataItem{Kind: biz.MasterDataKindContainerSpec, Code: "XXGP", Name: "重复同码箱型", Source: "manual", SortOrder: 20}, audit())
	if err != biz.ErrMasterDataCodeExists {
		t.Fatalf("A 型同 (kind, code) 第二次插入应映射 ErrMasterDataCodeExists，实际 %v", err)
	}
	_ = first
}

func TestPortBaselineUniquePhysicalRejectPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	repo := NewIndustryReferenceRepo(data)
	// ports 约束要求 un_locode 前两位等于 country_code，夹具取合成码 CNXYP。
	audit := func() *biz.AuditEvent {
		return &biz.AuditEvent{Action: "port.create", Result: "success", Details: map[string]string{}}
	}
	branch := data.db.Organization.Create().
		SetCode("BR-UNIQ-" + uuid.NewString()[:8]).
		SetName("港口唯一性测试分公司").
		SetKind("company").
		SetBaseCurrency("CNY").
		SaveX(ctx)

	baseline := &biz.Port{UNLocode: "CNXYP", NameEN: "Baseline Xiny Port", CountryCode: "CN", TransportModes: []string{"SEA"}, Source: "manual", SortOrder: 10}
	if _, err := repo.CreatePort(ctx, uuid.Nil, baseline, audit()); err != nil {
		t.Fatalf("首个 NULL 基线港口创建失败: %v", err)
	}
	// 两条 NULL 同 Code：被部分唯一索引 ports_baseline_locode_unique 物理拒绝。
	if _, err := repo.CreatePort(ctx, uuid.Nil, &biz.Port{UNLocode: "CNXYP", NameEN: "Duplicate Baseline", CountryCode: "CN", TransportModes: []string{"SEA"}, Source: "manual", SortOrder: 20}, audit()); err != biz.ErrIndustryReferenceCodeExist {
		t.Fatalf("B 型两条 NULL 同 un_locode 应被物理拒绝并映射 ErrIndustryReferenceCodeExist，实际 %v", err)
	}
	// NULL 基线行与 org 行同 Code 允许并存（B 型遮蔽前置条件）。
	if _, err := repo.CreatePort(ctx, branch.ID, &biz.Port{OrganizationID: &branch.ID, UNLocode: "CNXYP", NameEN: "Branch Xiny Port", CountryCode: "CN", TransportModes: []string{"SEA"}, Source: "manual", SortOrder: 30}, audit()); err != nil {
		t.Fatalf("基线行与本组织行同 Code 并存应成功: %v", err)
	}
}
