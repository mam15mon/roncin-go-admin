package data

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

func TestCompanyPersonnelWorkspacePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	client, err := data.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fixture := newOrderPostgresFixture(t, data)
	companyID := fixture.organizationID
	newOrg := func(code, name string, kind organizationent.Kind, parentID uuid.UUID, enabled bool) *ent.Organization {
		t.Helper()
		create := client.Organization.Create().SetCode(code).SetName(name).SetKind(kind).SetEnabled(enabled)
		if kind == organizationent.KindCompany {
			create.SetBaseCurrency("CNY")
		} else {
			create.SetParentID(parentID)
		}
		org, err := create.Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return org
	}
	department := newOrg("PERSONNEL-DEPT", "财务部", organizationent.KindDepartment, companyID, true)
	team := newOrg("PERSONNEL-TEAM", "核算组", organizationent.KindTeam, department.ID, true)
	childCompany := newOrg("PERSONNEL-CHILD", "另一经营公司", organizationent.KindCompany, companyID, true)
	childDepartment := newOrg("PERSONNEL-CHILD-DEPT", "另一公司部门", organizationent.KindDepartment, childCompany.ID, true)
	t.Cleanup(func() {
		for _, id := range []uuid.UUID{team.ID, department.ID, childDepartment.ID} {
			if _, err := client.Membership.Delete().Where(membershipent.OrganizationIDEQ(id)).Exec(ctx); err != nil {
				t.Error(err)
			}
			if err := client.Organization.DeleteOneID(id).Exec(ctx); err != nil {
				t.Error(err)
			}
		}
	})
	disabledDepartment := newOrg("PERSONNEL-DISABLED", "停用部门", organizationent.KindDepartment, companyID, false)
	t.Cleanup(func() {
		if _, err := client.Membership.Delete().Where(membershipent.OrganizationIDEQ(disabledDepartment.ID)).Exec(ctx); err != nil {
			t.Error(err)
		}
		if err := client.Organization.DeleteOneID(disabledDepartment.ID).Exec(ctx); err != nil {
			t.Error(err)
		}
	})
	newUser := func(orgID uuid.UUID, enabled, membershipEnabled bool) *ent.User {
		t.Helper()
		person, err := client.User.Create().SetDisplayName("张三").SetEnabled(enabled).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.Membership.Create().SetUserID(person.ID).SetOrganizationID(orgID).SetEnabled(membershipEnabled).Save(ctx); err != nil {
			t.Fatal(err)
		}
		return person
	}
	multiDepartment := newUser(department.ID, true, true)
	if _, err := client.Membership.Create().SetUserID(multiDepartment.ID).SetOrganizationID(team.ID).SetEnabled(true).Save(ctx); err != nil {
		t.Fatal(err)
	}
	direct := newUser(companyID, true, true)
	outsider := newUser(childDepartment.ID, true, true)
	newUser(childCompany.ID, true, true)
	newUser(disabledDepartment.ID, true, true)
	newUser(department.ID, false, true)
	newUser(department.ID, true, false)
	partnerRepo := NewPartnerRepo(data)
	orderRepo := NewOrderRepo(data)
	options := biz.SelectorListOptions{Keyword: "张三", Page: 1, PageSize: 200}

	t.Run("伙伴与订单同名不合并且多部门聚合", func(t *testing.T) {
		partners, err := partnerRepo.ListAssignmentOptions(ctx, companyID, options)
		if err != nil {
			t.Fatal(err)
		}
		orders, err := orderRepo.ListPersonnelOptions(ctx, companyID, options)
		if err != nil {
			t.Fatal(err)
		}
		if partners.Total != 2 || orders.Total != 2 {
			t.Fatalf("候选数量应为两个不同张三: partner=%d order=%d", partners.Total, orders.Total)
		}
		want := map[uuid.UUID][]string{multiDepartment.ID: {"核算组", "财务部"}, direct.ID: {}}
		for _, item := range partners.Items {
			names, ok := want[item.UserID]
			if !ok || !reflect.DeepEqual(item.DepartmentNames, names) {
				t.Fatalf("客户候选部门不正确: %+v", item)
			}
		}
		for _, item := range orders.Items {
			names, ok := want[item.UserID]
			if !ok || !reflect.DeepEqual(item.DepartmentNames, names) {
				t.Fatalf("订单候选部门不正确: %+v", item)
			}
		}
	})
	t.Run("分页按用户计数且部门搜索保留全部部门", func(t *testing.T) {
		page := options
		page.PageSize = 1
		first, err := partnerRepo.ListAssignmentOptions(ctx, companyID, page)
		if err != nil {
			t.Fatal(err)
		}
		page.Page = 2
		second, err := partnerRepo.ListAssignmentOptions(ctx, companyID, page)
		if err != nil {
			t.Fatal(err)
		}
		if first.Total != 2 || second.Total != 2 || len(first.Items) != 1 || len(second.Items) != 1 || first.Items[0].UserID == second.Items[0].UserID {
			t.Fatal("分页应按不同用户返回")
		}
		page.Keyword = "财务"
		page.Page = 1
		matches, err := orderRepo.ListPersonnelOptions(ctx, companyID, page)
		if err != nil {
			t.Fatal(err)
		}
		if matches.Total != 1 || len(matches.Items[0].DepartmentNames) != 2 {
			t.Fatalf("按部门搜索丢失用户的其他部门: %+v", matches)
		}
	})
	t.Run("切换下属公司后候选不含原公司", func(t *testing.T) {
		result, err := partnerRepo.ListAssignmentOptions(ctx, childCompany.ID, options)
		if err != nil {
			t.Fatal(err)
		}
		if result.Total != 2 {
			t.Fatalf("下属公司应仅有本公司两人: %d", result.Total)
		}
		for _, item := range result.Items {
			if item.UserID == direct.ID || item.UserID == multiDepartment.ID {
				t.Fatal("原公司人员泄漏")
			}
		}
	})
	t.Run("伙伴拒绝下属公司人员且回滚", func(t *testing.T) {
		_, err := biz.NewPartnerUsecase(partnerRepo).Create(ctx, companyID, fixture.actorID, &biz.Partner{
			LegalName: "跨公司责任人客户", IsCasual: true,
			Roles:       []*biz.PartnerRole{{Type: biz.PartnerRoleCustomer, Enabled: true}},
			Assignments: []*biz.PartnerAssignment{{Role: biz.PartnerAssignmentSales, UserID: outsider.ID}},
		})
		if !errors.Is(err, biz.ErrPartnerInvalidArgument) {
			t.Fatalf("应拒绝下属公司责任人: %v", err)
		}
	})
	t.Run("订单创建拒绝下属公司人员并回滚", func(t *testing.T) {
		input := fixture.validInput()
		input.PersonnelAssignments = []*biz.OrderPersonnel{{UserID: outsider.ID, Role: biz.OrderPersonnelRoleOperator}}
		_, err := fixture.newUsecase().Create(ctx, companyID, fixture.actorID, input)
		if !errors.Is(err, biz.ErrOrderPersonnelUserInvalid) {
			t.Fatalf("应拒绝下属公司操作员: %v", err)
		}
		fixture.requireRolledBackState()
	})
	t.Run("订单分配接受本公司部门而拒绝下属公司", func(t *testing.T) {
		created, err := fixture.newUsecase().Create(ctx, companyID, fixture.actorID, fixture.validInput())
		if err != nil {
			t.Fatal(err)
		}
		personnelUsecase := biz.NewOrderPersonnelUsecase(NewOrderPersonnelRepo(data))
		if _, err := personnelUsecase.Assign(ctx, companyID, fixture.actorID, created.ID, outsider.ID, biz.OrderPersonnelRoleOperator); !errors.Is(err, biz.ErrOrderPersonnelUserInvalid) {
			t.Fatalf("应拒绝下属公司操作员: %v", err)
		}
		assigned, err := personnelUsecase.Assign(ctx, companyID, fixture.actorID, created.ID, multiDepartment.ID, biz.OrderPersonnelRoleOperator)
		if err != nil {
			t.Fatal(err)
		}
		if assigned.OrganizationID != companyID {
			t.Fatal("经营归属应固定公司")
		}
	})
}
