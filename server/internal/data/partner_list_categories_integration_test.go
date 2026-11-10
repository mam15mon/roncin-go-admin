package data

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
)

func TestPartnerListMutuallyExclusiveCategories(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := t.Context()
	client, err := d.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	company, err := client.Organization.Create().SetCode("CATEGORY-COMPANY").SetName("分类验证公司").SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	otherCompany, err := client.Organization.Create().SetCode("CATEGORY-OTHER").SetName("其他公司").SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	type roleState struct{ enabled, blacklisted bool }
	type fixture struct {
		id      uuid.UUID
		enabled bool
		casual  bool
		roles   map[biz.PartnerRoleType]roleState
	}
	var fixtures []fixture
	createPartner := func(companyID uuid.UUID, enabled, casual bool, roles map[biz.PartnerRoleType]roleState) {
		t.Helper()
		name := fmt.Sprintf("分类验证-%d-%s", len(fixtures), uuid.New())
		partner, err := client.Partner.Create().SetOrganizationID(companyID).SetLegalName(name).SetNormalizedName(name).SetEnabled(enabled).SetIsCasual(casual).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		for role, state := range roles {
			if _, err := client.PartnerRole.Create().SetPartnerID(partner.ID).SetRoleType(partnerroleent.RoleType(role)).SetEnabled(state.enabled).SetBlacklisted(state.blacklisted).Save(ctx); err != nil {
				t.Fatal(err)
			}
		}
		if companyID == company.ID {
			fixtures = append(fixtures, fixture{id: partner.ID, enabled: enabled, casual: casual, roles: roles})
		}
	}
	roles := []biz.PartnerRoleType{biz.PartnerRoleCustomer, biz.PartnerRoleSupplier, biz.PartnerRoleForeignAgent}
	for _, role := range roles {
		for _, enabled := range []bool{true, false} {
			for _, roleEnabled := range []bool{true, false} {
				for _, casual := range []bool{true, false} {
					for _, blacklisted := range []bool{true, false} {
						createPartner(company.ID, enabled, casual, map[biz.PartnerRoleType]roleState{role: {roleEnabled, blacklisted}})
					}
				}
			}
		}
	}
	// 同一档案在不同业务角色中的黑名单状态互不影响。
	createPartner(company.ID, true, true, map[biz.PartnerRoleType]roleState{
		biz.PartnerRoleCustomer:     {true, true},
		biz.PartnerRoleSupplier:     {true, false},
		biz.PartnerRoleForeignAgent: {false, false},
	})
	createPartner(otherCompany.ID, true, false, map[biz.PartnerRoleType]roleState{biz.PartnerRoleCustomer: {true, false}})
	createPartner(otherCompany.ID, true, true, map[biz.PartnerRoleType]roleState{biz.PartnerRoleCustomer: {true, true}})
	repo := NewPartnerRepo(d)
	for _, role := range roles {
		t.Run(string(role), func(t *testing.T) {
			type category struct {
				name        string
				blacklisted bool
				casual      *bool
			}
			categories := []category{{name: "正常合作"}, {name: "黑名单", blacklisted: true}}
			if role == biz.PartnerRoleCustomer {
				categories = append(categories, category{name: "散客", casual: new(true)})
				categories[0].casual = new(false)
			}
			seen := make(map[uuid.UUID]bool)
			for _, category := range categories {
				options := biz.PartnerListOptions{Page: 1, PageSize: biz.MaxListPageSize, Role: role, Blacklisted: &category.blacklisted, IsCasual: category.casual}
				result, err := repo.List(ctx, []uuid.UUID{company.ID}, options)
				if err != nil {
					t.Fatal(err)
				}
				want := make(map[uuid.UUID]bool)
				for _, f := range fixtures {
					state, exists := f.roles[role]
					if exists && state.blacklisted == category.blacklisted && (category.casual == nil || f.casual == *category.casual) {
						want[f.id] = true
					}
				}
				if result.Total != len(want) || len(result.Items) != len(want) {
					t.Fatalf("%s 结果不完整: total=%d items=%d want=%d", category.name, result.Total, len(result.Items), len(want))
				}
				for _, item := range result.Items {
					if !want[item.ID] || seen[item.ID] {
						t.Fatalf("%s 出现错误分组、跨公司或重复档案: %s", category.name, item.ID)
					}
					seen[item.ID] = true
				}
				options.Enabled = new(false)
				filtered, err := repo.List(ctx, []uuid.UUID{company.ID}, options)
				if err != nil {
					t.Fatal(err)
				}
				wantDisabled := 0
				for _, f := range fixtures {
					if want[f.id] && !f.enabled {
						wantDisabled++
					}
				}
				if filtered.Total != wantDisabled || len(filtered.Items) != wantDisabled {
					t.Fatalf("%s 停用筛选改变了分类: total=%d want=%d", category.name, filtered.Total, wantDisabled)
				}
			}
			for _, f := range fixtures {
				if _, exists := f.roles[role]; exists && !seen[f.id] {
					t.Fatalf("档案未归入任何页签: %s", f.id)
				}
			}
			selector, err := repo.List(ctx, []uuid.UUID{company.ID}, biz.PartnerListOptions{Page: 1, PageSize: biz.MaxListPageSize, Role: role})
			if err != nil {
				t.Fatal(err)
			}
			wantSelector := 0
			for _, f := range fixtures {
				if state, exists := f.roles[role]; exists && state.enabled {
					wantSelector++
				}
			}
			if selector.Total != wantSelector {
				t.Fatalf("业务选择器未指定黑名单条件时只能查询启用角色: got=%d want=%d", selector.Total, wantSelector)
			}
		})
	}
}
