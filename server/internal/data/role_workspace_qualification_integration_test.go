package data

import (
	"context"
	"reflect"
	"sort"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
)

// 同一套公司边界同时用于审批、提醒与运行时授权；错绑角色不能跨工作台生效。
func TestRoleWorkspaceQualificationPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	client, err := data.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	newOrg := func(code string, kind organizationent.Kind, parentID *uuid.UUID, enabled bool) *ent.Organization {
		t.Helper()
		create := client.Organization.Create().SetCode(code).SetName(code).SetKind(kind).SetEnabled(enabled).SetNillableParentID(parentID)
		if kind == organizationent.KindCompany || kind == organizationent.KindSystem {
			create.SetBaseCurrency("CNY")
		}
		item, err := create.Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return item
	}
	system := newOrg("QUAL-SYSTEM", organizationent.KindSystem, nil, true)
	company := newOrg("QUAL-COMPANY", organizationent.KindCompany, nil, true)
	department := newOrg("QUAL-DEPT", organizationent.KindDepartment, &company.ID, true)
	disabledDepartment := newOrg("QUAL-DISABLED", organizationent.KindDepartment, &company.ID, false)
	disabledChild := newOrg("QUAL-DISABLED-CHILD", organizationent.KindTeam, &disabledDepartment.ID, true)
	nestedCompany := newOrg("QUAL-OTHER", organizationent.KindCompany, nil, true)
	nestedDepartment := newOrg("QUAL-NESTED-DEPT", organizationent.KindDepartment, &nestedCompany.ID, true)
	permissions := make([]*ent.Permission, 0, 5)
	for _, key := range []string{
		access.OrderPermission(access.OrderBusinessSI, access.OrderRead),
		access.OrderPermission(access.OrderBusinessSI, access.OrderUpdate),
		access.OrderPermission(access.OrderBusinessSI, access.OrderFeeRead),
		access.OrderPermission(access.OrderBusinessSI, access.OrderLock),
		access.FinanceExchangeRateCreate,
	} {
		permission, err := client.Permission.Query().Where(permissionent.KeyEQ(key)).Only(ctx)
		if ent.IsNotFound(err) {
			permission, err = client.Permission.Create().SetKey(key).SetName(key).SetGroup("测试").Save(ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
		permissions = append(permissions, permission)
	}
	newRole := func(code string, org *ent.Organization, enabled bool) *ent.Role {
		t.Helper()
		item, err := client.Role.Create().SetOrganizationID(org.ID).SetCode(code).SetName(code).SetEnabled(enabled).AddPermissions(permissions...).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return item
	}
	companyRole := newRole("qualified", company, true)
	systemRole := newRole("system", system, true)
	otherRole := newRole("other", nestedCompany, true)
	disabledRole := newRole("disabled", company, false)
	newMember := func(code string, org *ent.Organization, role *ent.Role, userEnabled, memberEnabled bool) *ent.User {
		t.Helper()
		account, err := client.User.Create().SetDisplayName(code).SetDingtalkUserid(code).SetEnabled(userEnabled).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		member, err := client.Membership.Create().SetOrganizationID(org.ID).SetUserID(account.ID).SetEnabled(memberEnabled).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := client.RoleAssignment.Create().SetMembershipID(member.ID).SetRoleID(role.ID).Save(ctx); err != nil {
			t.Fatal(err)
		}
		return account
	}
	direct := newMember("valid-direct", company, companyRole, true, true)
	dept := newMember("valid-dept", department, companyRole, true, true)
	invalid := []*ent.User{
		newMember("disabled-dept", disabledDepartment, companyRole, true, true),
		newMember("disabled-parent", disabledChild, companyRole, true, true),
		newMember("nested-company", nestedCompany, companyRole, true, true),
		newMember("nested-dept", nestedDepartment, companyRole, true, true),
		newMember("system-role", department, systemRole, true, true),
		newMember("other-role", department, otherRole, true, true),
		newMember("disabled-role", department, disabledRole, true, true),
		newMember("disabled-user", department, companyRole, false, true),
		newMember("disabled-member", department, companyRole, true, false),
	}
	assertUsers := func(label string, actual []uuid.UUID) {
		t.Helper()
		want := []uuid.UUID{direct.ID, dept.ID}
		sort.Slice(want, func(i, j int) bool { return want[i].String() < want[j].String() })
		sort.Slice(actual, func(i, j int) bool { return actual[i].String() < actual[j].String() })
		if !reflect.DeepEqual(actual, want) {
			t.Fatalf("%s 候选越过启用公司边界: got=%v want=%v", label, actual, want)
		}
	}
	locks, err := queryQualifiedBusinessLockCandidates(ctx, client, company.ID, access.OrderBusinessSI)
	if err != nil {
		t.Fatal(err)
	}
	lockIDs := make([]uuid.UUID, 0, len(locks))
	for _, item := range locks {
		lockIDs = append(lockIDs, item.UserID)
	}
	assertUsers("锁定", lockIDs)
	reminders, err := NewExchangeRateReminderRepo(data).ListFinanceReminderRecipients(ctx, company.ID)
	if err != nil {
		t.Fatal(err)
	}
	reminderIDs := make([]uuid.UUID, 0, len(reminders))
	for _, item := range reminders {
		reminderIDs = append(reminderIDs, item.UserID)
	}
	assertUsers("汇率提醒", reminderIDs)
	for _, account := range append([]*ent.User{direct, dept}, invalid...) {
		want := account.ID == direct.ID || account.ID == dept.ID
		locked, lockErr := isUserQualifiedBusinessLockRole(ctx, client, company.ID, account.ID, access.OrderBusinessSI)
		reviewed, reviewErr := hasRealtimeFeePermission(ctx, client, company.ID, account.ID, access.OrderPermission(access.OrderBusinessSI, access.OrderLock), false)
		if lockErr != nil || reviewErr != nil || locked != want || reviewed != want {
			t.Fatalf("%s 实时资格错误: lock=%v/%v review=%v/%v want=%v", account.DisplayName, locked, lockErr, reviewed, reviewErr, want)
		}
	}
	// 其他公司的成员即使被错误关联本公司角色，也不能成为本公司的授权来源。
	for _, account := range invalid[2:4] {
		profiles, err := actorRolesPrivilegeProfiles(ctx, client, company.ID, account.ID)
		if err != biz.ErrAdminPrivilegeEscalation || len(profiles) != 0 {
			t.Fatalf("其他公司成员进入本公司权限画像: profiles=%v err=%v", profiles, err)
		}
	}
	if _, err := client.Organization.Create().SetCode("QUAL-INVALID-NESTED").SetName("非法嵌套公司").SetKind(organizationent.KindCompany).SetBaseCurrency("CNY").SetParentID(company.ID).Save(ctx); !ent.IsConstraintError(err) {
		t.Fatalf("正式迁移应拒绝嵌套公司: %v", err)
	}
}
