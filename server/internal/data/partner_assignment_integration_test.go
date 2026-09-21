package data

import (
	"context"
	"errors"
	"testing"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
)

func TestPartnerCreateAssignmentEligibility(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()
	ctx := context.Background()
	client, err := data.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	company, err := client.Organization.Create().SetCode("PARTNER-COMPANY").SetName("客户测试公司").SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	department, err := client.Organization.Create().SetCode("PARTNER-DEPARTMENT").SetName("客户测试部门").SetKind("department").SetParentID(company.ID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := client.User.Create().SetDisplayName("无成员关系的初始化管理员").SetIsBootstrapAdmin(true).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	employee, err := client.User.Create().SetDisplayName("部门业务员").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	membership, err := client.Membership.Create().SetUserID(employee.ID).SetOrganizationID(department.ID).SetEnabled(true).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	uc := biz.NewPartnerUsecase(NewPartnerRepo(data))

	t.Run("快捷创建记录无成员关系的创建人", func(t *testing.T) {
		created, err := uc.Create(ctx, company.ID, actor.ID, &biz.Partner{
			LegalName: "测试的", IsCasual: true,
			Roles: []*biz.PartnerRole{{Type: biz.PartnerRoleCustomer, Enabled: true}},
		})
		if err != nil {
			t.Fatalf("快捷创建客户失败: %v", err)
		}
		if !created.IsCasual || len(created.Roles) != 1 || created.Roles[0].Type != biz.PartnerRoleCustomer || !created.Roles[0].Enabled {
			t.Fatalf("散客或客户角色未保存: %+v", created)
		}
		if len(created.Assignments) != 1 || created.Assignments[0].Role != biz.PartnerAssignmentCreator || created.Assignments[0].UserID != actor.ID || created.Assignments[0].OrganizationID != company.ID {
			t.Fatalf("创建人事实或公司归属不正确: %+v", created.Assignments)
		}
	})

	for _, tc := range []struct {
		name   string
		reject bool
		setup  func() error
		actor  bool
	}{
		{name: "无成员关系不能担任业务员", reject: true, actor: true},
		{name: "启用部门成员可担任业务员"},
		{name: "停用用户不能担任业务员", reject: true, setup: func() error { return client.User.UpdateOne(employee).SetEnabled(false).Exec(ctx) }},
		{name: "停用成员关系不能担任业务员", reject: true, setup: func() error {
			if err := client.User.UpdateOne(employee).SetEnabled(true).Exec(ctx); err != nil {
				return err
			}
			return client.Membership.UpdateOne(membership).SetEnabled(false).Exec(ctx)
		}},
		{name: "停用部门不能担任业务员", reject: true, setup: func() error {
			if err := client.Membership.UpdateOne(membership).SetEnabled(true).Exec(ctx); err != nil {
				return err
			}
			return client.Organization.UpdateOne(department).SetEnabled(false).Exec(ctx)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if tc.setup != nil {
				if err := tc.setup(); err != nil {
					t.Fatal(err)
				}
			}
			userID := employee.ID
			if tc.actor {
				userID = actor.ID
			}
			created, err := uc.Create(ctx, company.ID, actor.ID, &biz.Partner{
				LegalName: tc.name, IsCasual: true,
				Roles:       []*biz.PartnerRole{{Type: biz.PartnerRoleCustomer, Enabled: true}},
				Assignments: []*biz.PartnerAssignment{{Role: biz.PartnerAssignmentSales, UserID: userID}},
			})
			if tc.reject {
				if !errors.Is(err, biz.ErrPartnerInvalidArgument) {
					t.Fatalf("应拒绝无资格业务员，实际: %v", err)
				}
				exists, queryErr := client.Partner.Query().Where(partnerent.LegalNameEQ(tc.name)).Exist(ctx)
				if queryErr != nil || exists {
					t.Fatalf("拒绝后不应留下客户记录，exists=%v err=%v", exists, queryErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("合法部门业务员创建失败: %v", err)
			}
			if len(created.Assignments) != 2 {
				t.Fatalf("责任人数量错误: %d", len(created.Assignments))
			}
			for _, assignment := range created.Assignments {
				if assignment.OrganizationID != company.ID {
					t.Fatalf("责任人经营归属应为公司: %+v", assignment)
				}
				if assignment.Role == biz.PartnerAssignmentSales && assignment.UserID != employee.ID {
					t.Fatalf("业务员错误: %+v", assignment)
				}
			}
		})
	}
}
