package data

import (
	"reflect"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
)

func TestPersonnelOrganizationIDs(t *testing.T) {
	company := &ent.Organization{ID: uuid.New(), Kind: organizationent.KindCompany, Enabled: true}
	department := &ent.Organization{ID: uuid.New(), Kind: organizationent.KindDepartment, ParentID: &company.ID, Enabled: true}
	team := &ent.Organization{ID: uuid.New(), Kind: organizationent.KindTeam, ParentID: &department.ID, Enabled: true}
	childCompany := &ent.Organization{ID: uuid.New(), Kind: organizationent.KindCompany, ParentID: &company.ID, Enabled: true}
	childDepartment := &ent.Organization{ID: uuid.New(), Kind: organizationent.KindDepartment, ParentID: &childCompany.ID, Enabled: true}
	disabled := &ent.Organization{ID: uuid.New(), Kind: organizationent.KindDepartment, ParentID: &company.ID, Enabled: false}
	disabledChild := &ent.Organization{ID: uuid.New(), Kind: organizationent.KindTeam, ParentID: &disabled.ID, Enabled: true}
	organizations := []*ent.Organization{team, childCompany, disabledChild, department, childDepartment, company, disabled}
	got := personnelOrganizationIDs(organizations, company.ID)
	if !reflect.DeepEqual(got, []uuid.UUID{company.ID, department.ID, team.ID}) {
		t.Fatalf("公司边界错误: %v", got)
	}
	if got := personnelOrganizationIDs(organizations, department.ID); len(got) != 0 {
		t.Fatal("部门不能作为经营工作台")
	}
	company.Enabled = false
	if got := personnelOrganizationIDs(organizations, company.ID); len(got) != 0 {
		t.Fatal("停用公司不能返回人员范围")
	}
}
