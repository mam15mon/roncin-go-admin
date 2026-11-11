package data

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	partneraliasent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partneralias"
	partnercontactent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnercontact"
	partnerprofileent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerprofile"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
)

func TestPartnerUpdateRolesPreservesAggregatePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := t.Context()
	fixture := newOrderPostgresFixture(t, data)
	client, err := data.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	uc := biz.NewPartnerUsecase(NewPartnerRepo(data))
	created, err := uc.Create(ctx, fixture.organizationID, fixture.actorID, &biz.Partner{
		Code: "ROLE-ONLY", LegalName: "完整客商资料", IsCasual: true,
		RegisteredAddress: "原注册地址",
		Roles:             []*biz.PartnerRole{{Type: biz.PartnerRoleCustomer, Enabled: true}},
		Contacts:          []*biz.PartnerContact{{Name: "原联系人", Phone: "13800000000", IsPrimary: true}},
		Aliases:           []*biz.PartnerAlias{{AliasName: "原别名", SortOrder: 1}},
		Profile:           &biz.PartnerProfile{NameEN: "Original Customer", CountryCode: "US", AddressDetail: "原地址", Remark: "原备注"},
		Assignments: []*biz.PartnerAssignment{
			{Role: biz.PartnerAssignmentSales, UserID: fixture.actorID},
			{Role: biz.PartnerAssignmentOperator, UserID: fixture.actorID},
			{Role: biz.PartnerAssignmentCustomerService, UserID: fixture.actorID},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	customerRole, err := client.PartnerRole.Query().Where(partnerroleent.PartnerIDEQ(created.ID), partnerroleent.RoleTypeEQ(partnerroleent.RoleTypeCustomer)).Only(ctx)
	if err != nil {
		t.Fatal(err)
	}
	rule, err := client.PartnerSettlementRule.Create().SetPartnerRoleID(customerRole.ID).SetStatementMode("single").SetSettlementMethod("by_ticket").SetSettlementCurrency("CNY").SetPaymentTermsDays(30).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := client.PartnerSettlementRule.DeleteOneID(rule.ID).Exec(context.Background()); err != nil {
			t.Errorf("清理角色结算规则: %v", err)
		}
	})
	// 以数据库已保存精度作为保留基线，避免内存时间纳秒与 PostgreSQL 微秒比较。
	rule, err = client.PartnerSettlementRule.Get(ctx, rule.ID)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx := context.Background()
		if _, err := client.PartnerAlias.Delete().Where(partneraliasent.PartnerIDEQ(created.ID)).Exec(cleanupCtx); err != nil {
			t.Errorf("清理别名: %v", err)
		}
		if _, err := client.PartnerContact.Delete().Where(partnercontactent.PartnerIDEQ(created.ID)).Exec(cleanupCtx); err != nil {
			t.Errorf("清理联系人: %v", err)
		}
		if _, err := client.PartnerProfile.Delete().Where(partnerprofileent.PartnerIDEQ(created.ID)).Exec(cleanupCtx); err != nil {
			t.Errorf("清理档案: %v", err)
		}
	})
	if err := client.Partner.UpdateOneID(created.ID).SetEnabled(false).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	before, err := uc.Get(ctx, fixture.organizationID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertPreserved := func(got *biz.Partner) {
		t.Helper()
		if got.Enabled || !got.IsCasual || got.Code != before.Code || got.LegalName != before.LegalName || got.RegisteredAddress != before.RegisteredAddress {
			t.Fatalf("身份更新修改了主档: %+v", got)
		}
		if !reflect.DeepEqual(got.Contacts, before.Contacts) || !reflect.DeepEqual(got.Aliases, before.Aliases) || !reflect.DeepEqual(got.Profile, before.Profile) || !reflect.DeepEqual(got.Assignments, before.Assignments) {
			t.Fatalf("身份更新丢失或重建了子资料: before=%+v after=%+v", before, got)
		}
		retainedRule, err := client.PartnerSettlementRule.Get(ctx, rule.ID)
		retainedJSON, _ := json.Marshal(retainedRule)
		originalJSON, _ := json.Marshal(rule)
		if err != nil || string(retainedJSON) != string(originalJSON) {
			t.Fatalf("角色结算规则被改写: got=%s want=%s err=%v", retainedJSON, originalJSON, err)
		}
	}
	for _, roles := range [][]biz.PartnerRoleType{
		{biz.PartnerRoleCustomer, biz.PartnerRoleSupplier},
		{biz.PartnerRoleCustomer, biz.PartnerRoleSupplier},
		{biz.PartnerRoleSupplier},
		{biz.PartnerRoleCustomer, biz.PartnerRoleSupplier},
	} {
		updated, err := uc.UpdateRoles(ctx, fixture.organizationID, fixture.actorID, created.ID, roles)
		if err != nil {
			t.Fatal(err)
		}
		assertPreserved(updated)
		enabledRoles := map[biz.PartnerRoleType]bool{}
		for _, role := range updated.Roles {
			if role.Enabled {
				enabledRoles[role.Type] = true
			}
		}
		if len(enabledRoles) != len(roles) {
			t.Fatalf("启用身份不匹配: %+v", updated.Roles)
		}
		for _, role := range roles {
			if !enabledRoles[role] {
				t.Fatalf("缺少身份 %s", role)
			}
		}
	}
	if _, err := uc.UpdateRoles(ctx, uuid.New(), fixture.actorID, created.ID, []biz.PartnerRoleType{biz.PartnerRoleForeignAgent}); !errors.Is(err, biz.ErrPartnerNotFound) {
		t.Fatalf("跨公司身份更新应拒绝: %v", err)
	}
	if err := client.PartnerRole.UpdateOneID(customerRole.ID).SetBlacklisted(true).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := uc.UpdateRoles(ctx, fixture.organizationID, fixture.actorID, created.ID, []biz.PartnerRoleType{biz.PartnerRoleForeignAgent}); !errors.Is(err, biz.ErrPartnerBlacklistedRole) {
		t.Fatalf("移除黑名单身份应整笔拒绝: %v", err)
	}
	after, err := uc.Get(ctx, fixture.organizationID, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	assertPreserved(after)
	for _, role := range after.Roles {
		if role.Type == biz.PartnerRoleForeignAgent || !role.Enabled {
			t.Fatalf("失败写入不应残留新增或身份停用: %+v", after.Roles)
		}
	}
}
