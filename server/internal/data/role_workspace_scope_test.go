package data

import (
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
)

func TestWorkspaceRoleScopeDoesNotCrossWorkspace(t *testing.T) {
	system, company, department, nested, nestedDepartment := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	nodes := map[uuid.UUID]authOrgNode{
		system:           {ID: system, Kind: "system"},
		company:          {ID: company, Kind: "company", ParentID: &system},
		department:       {ID: department, Kind: "department", ParentID: &company},
		nested:           {ID: nested, Kind: "company", ParentID: &company},
		nestedDepartment: {ID: nestedDepartment, Kind: "department", ParentID: &nested},
	}
	ids := workspaceMembershipScopeIDs(nodes, company)
	if len(ids) != 2 || !containsUUID(ids, company) || !containsUUID(ids, department) {
		t.Fatalf("成员资格不应越过其他工作台: %v", ids)
	}
	for id, want := range map[uuid.UUID]biz.DataScope{system: biz.DataScopeAll, company: biz.DataScopeOrganization, department: "", uuid.New(): ""} {
		if got := workspaceDataScope(nodes, id); got != want {
			t.Fatalf("工作台范围推导错误: got=%q want=%q", got, want)
		}
	}
}
