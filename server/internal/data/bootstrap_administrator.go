package data

import (
	"context"
	"fmt"

	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// BootstrapAdministratorInput 的账号资料仅用于首次创建，重复执行只核对非空用户名。
type BootstrapAdministratorInput struct {
	Username, DisplayName, PlainPassword, OrganizationCode, OrganizationName string
	// HashPassword 由命令层提供，只有首次创建路径调用，数据层不依赖密码算法。
	HashPassword func(string) (string, error)
}

type AdministratorProvisioningSkip struct {
	OrganizationID   uuid.UUID
	OrganizationCode string
	Reason           string
}

// AdministratorProvisioningSummary 仅在整个事务提交成功后向调用者报告。
type AdministratorProvisioningSummary struct {
	Username                                                         string
	AccountCreated                                                   bool
	CompaniesCreated, MembershipsCreated, BindingsCreated, Satisfied int
	Skipped                                                          []AdministratorProvisioningSkip
}

// lockInitialAdministrator 是公司创建和补齐的共同串行化点，不能按普通同名账号识别身份。
func lockInitialAdministrator(ctx context.Context, client *ent.Client) (*ent.User, error) {
	accounts, err := client.User.Query().Where(user.IsBootstrapAdminEQ(true)).Order(user.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	if len(accounts) != 1 {
		return nil, biz.ErrAdminBootstrapIdentityInvalid
	}
	if !accounts[0].Enabled || accounts[0].PendingAssignment {
		return nil, biz.ErrAdminBootstrapInactive
	}
	return accounts[0], nil
}

type administratorWorkspacePlan struct {
	organization *ent.Organization
	member       *ent.Membership
	adminRole    *ent.Role
	skipReason   string
}

// prepareAdministratorPlans 要求已取得账号和按 ID 排序的组织锁；再按成员、角色顺序加锁。
// 系统必要资格失败在权限目录、默认公司及任何授权写入前返回。
func prepareAdministratorPlans(ctx context.Context, client *ent.Client, account *ent.User, organizations []*ent.Organization, systemID uuid.UUID) ([]*administratorWorkspacePlan, error) {
	members, err := client.Membership.Query().Where(membership.UserIDEQ(account.ID)).Order(membership.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	// 从首次读取即取得写互斥锁，禁止共享锁升级；NO KEY UPDATE 与权限关联
	// 的 FK KEY SHARE 相容，避免并发目录同步按目录→角色顺序产生反向等待。
	roles, err := client.Role.Query().Where(role.CodeEQ("administrator")).Order(role.ByID()).ForUpdate(entsql.WithLockClause("FOR NO KEY UPDATE")).All(ctx)
	if err != nil {
		return nil, err
	}
	byOrgMembers := make(map[uuid.UUID]*ent.Membership, len(members))
	byOrgRoles := make(map[uuid.UUID]*ent.Role, len(roles))
	for _, item := range members {
		byOrgMembers[item.OrganizationID] = item
	}
	for _, item := range roles {
		byOrgRoles[item.OrganizationID] = item
	}
	plans := make([]*administratorWorkspacePlan, 0, len(organizations))
	for _, item := range organizations {
		if item.ID != systemID && item.Kind != organization.KindCompany {
			continue
		}
		plan := &administratorWorkspacePlan{organization: item, member: byOrgMembers[item.ID], adminRole: byOrgRoles[item.ID]}
		switch {
		case !item.Enabled:
			plan.skipReason = "公司已停用，请人工恢复后重跑"
		case plan.adminRole != nil && !plan.adminRole.Enabled:
			plan.skipReason = "管理员角色已停用，请人工恢复后重跑"
		case plan.adminRole != nil && plan.adminRole.DataScope == role.DataScopeSelf:
			plan.skipReason = "管理员角色数据范围已失效，请人工恢复后重跑"
		case plan.member != nil && !plan.member.Enabled:
			plan.skipReason = "管理员成员关系已停用，请人工恢复后重跑"
		}
		if item.ID == systemID && plan.skipReason != "" {
			return nil, biz.ErrAdminBootstrapSystemInactive
		}
		plans = append(plans, plan)
	}
	return plans, nil
}

// provisionAdministratorPlan 只追加缺失项；保留已分配的其他角色及角色资料。
func provisionAdministratorPlan(ctx context.Context, client *ent.Client, account *ent.User, plan *administratorWorkspacePlan, source string, actorID *uuid.UUID, summary *AdministratorProvisioningSummary) error {
	if plan.skipReason != "" {
		summary.Skipped = append(summary.Skipped, AdministratorProvisioningSkip{OrganizationID: plan.organization.ID, OrganizationCode: plan.organization.Code, Reason: plan.skipReason})
		return nil
	}
	changed := false
	permissionIDs, err := administratorPermissionIDs(ctx, client, plan.organization.Kind == organization.KindSystem)
	if err != nil {
		return err
	}
	if plan.adminRole == nil {
		if plan.organization.Kind == organization.KindCompany {
			if err := initializeCompanyAdministrator(ctx, client, plan.organization.ID); err != nil {
				return mapEntError(err, nil, biz.ErrAdminRoleCodeExists)
			}
			plan.adminRole, err = client.Role.Query().Where(role.OrganizationIDEQ(plan.organization.ID), role.CodeEQ("administrator")).Only(ctx)
		} else {
			plan.adminRole, err = client.Role.Create().SetOrganizationID(plan.organization.ID).SetCode("administrator").SetName("系统管理员").SetDataScope(role.DataScopeAll).AddPermissionIDs(permissionIDs...).Save(ctx)
		}
		if err != nil {
			return mapEntError(err, nil, biz.ErrAdminRoleCodeExists)
		}
		changed = true
	} else {
		granted, queryErr := plan.adminRole.QueryPermissions().IDs(ctx)
		if queryErr != nil {
			return queryErr
		}
		grantedSet := make(map[uuid.UUID]bool, len(granted))
		for _, id := range granted {
			grantedSet[id] = true
		}
		missing := make([]uuid.UUID, 0)
		for _, id := range permissionIDs {
			if !grantedSet[id] {
				missing = append(missing, id)
			}
		}
		if len(missing) > 0 {
			if _, err := client.Role.UpdateOneID(plan.adminRole.ID).AddPermissionIDs(missing...).Save(ctx); err != nil {
				return err
			}
			changed = true
		}
	}
	if plan.member == nil {
		plan.member, err = client.Membership.Create().SetUserID(account.ID).SetOrganizationID(plan.organization.ID).Save(ctx)
		if err != nil {
			return mapEntError(err, nil, biz.ErrAdminUserMembershipExists)
		}
		summary.MembershipsCreated++
		changed = true
	}
	bound, err := client.RoleAssignment.Query().Where(roleassignment.MembershipIDEQ(plan.member.ID), roleassignment.RoleIDEQ(plan.adminRole.ID)).Exist(ctx)
	if err != nil {
		return err
	}
	if !bound {
		if _, err := client.RoleAssignment.Create().SetMembershipID(plan.member.ID).SetRoleID(plan.adminRole.ID).Save(ctx); err != nil {
			return mapEntError(err, nil, biz.ErrAdminUserMembershipExists)
		}
		summary.BindingsCreated++
		changed = true
	}
	if !changed {
		summary.Satisfied++
		return nil
	}
	return writeAudit(ctx, client.AuditLog, &biz.AuditEvent{OrganizationID: &plan.organization.ID, UserID: actorID, Action: "admin.bootstrap.provision", ResourceType: "membership", ResourceID: plan.member.ID.String(), Result: "success", Details: map[string]string{"source": source, "target_user_id": account.ID.String(), "organization_code": plan.organization.Code, "role_id": plan.adminRole.ID.String()}})
}

// BootstrapAdministrator 必须在 WithClientTx 内调用，首次和补齐均不单独提交权限同步。
func BootstrapAdministrator(ctx context.Context, tx *ent.Tx, input BootstrapAdministratorInput) (*AdministratorProvisioningSummary, error) {
	client := tx.Client()
	hasUsers, err := client.User.Query().Exist(ctx)
	if err != nil {
		return nil, err
	}
	var account *ent.User
	if hasUsers {
		account, err = lockInitialAdministrator(ctx, client)
		if err != nil {
			return nil, err
		}
		if input.Username != "" && input.Username != account.Username {
			return nil, fmt.Errorf("配置的管理员用户名与唯一初始管理员不一致，请使用原账号或留空")
		}
	}
	// 已有账号先锁初始用户；空库通过系统组织锁串行化首次身份创建。
	organizations, err := client.Organization.Query().Order(organization.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	system, err := uniqueAdministratorSystem(organizations)
	if err != nil {
		return nil, err
	}
	if system != nil && (system.Code != input.OrganizationCode || !system.Enabled) {
		return nil, biz.ErrAdminBootstrapSystemInactive
	}
	if system == nil {
		if hasUsers {
			return nil, biz.ErrAdminBootstrapSystemInactive
		}
		system, err = client.Organization.Create().SetCode(input.OrganizationCode).SetName(input.OrganizationName).SetKind(organization.KindSystem).SetBaseCurrency("CNY").Save(ctx)
		if err != nil {
			return nil, mapEntError(err, nil, biz.ErrAdminOrganizationCodeExists)
		}
		organizations = append(organizations, system)
	}
	if !hasUsers {
		if exists, err := client.User.Query().Exist(ctx); err != nil {
			return nil, err
		} else if exists {
			return nil, fmt.Errorf("初始化期间已有账号创建，请重新执行以补齐唯一初始管理员")
		}
		if input.Username == "" || input.DisplayName == "" || input.PlainPassword == "" {
			return nil, fmt.Errorf("首次初始化需要 BOOTSTRAP_ADMIN_USERNAME、BOOTSTRAP_ADMIN_DISPLAY_NAME 和 BOOTSTRAP_ADMIN_PASSWORD")
		}
		if input.HashPassword == nil {
			return nil, biz.ErrAdminInvalidArgument
		}
		hash, err := input.HashPassword(input.PlainPassword)
		if err != nil {
			return nil, err
		}
		account, err = client.User.Create().SetUsername(input.Username).SetDisplayName(input.DisplayName).SetPasswordHash(hash).SetIsBootstrapAdmin(true).Save(ctx)
		if err != nil {
			return nil, mapEntError(err, nil, biz.ErrAdminUsernameExists)
		}
	}
	plans, err := prepareAdministratorPlans(ctx, client, account, organizations, system.ID)
	if err != nil {
		return nil, err
	}
	// 只同步权限目录，不调用会给停用角色补权限的全局角色同步。
	if err := synchronizeBootstrapPermissionCatalog(ctx, client); err != nil {
		return nil, err
	}
	if err := CreateDefaultOrderOptions(ctx, tx); err != nil {
		return nil, err
	}
	if err := CreateDefaultCountries(ctx, tx); err != nil {
		return nil, err
	}
	created, err := createDefaultBranchCompanyRecords(ctx, tx)
	if err != nil {
		return nil, err
	}
	summary := &AdministratorProvisioningSummary{Username: account.Username, AccountCreated: !hasUsers, CompaniesCreated: len(created)}
	for _, item := range created {
		plans = append(plans, &administratorWorkspacePlan{organization: item})
	}
	for _, plan := range plans {
		if err := provisionAdministratorPlan(ctx, client, account, plan, "bootstrap-admin", &account.ID, summary); err != nil {
			return nil, err
		}
	}
	return summary, nil
}

func synchronizeBootstrapPermissionCatalog(ctx context.Context, client *ent.Client) error {
	existing, err := client.Permission.Query().All(ctx)
	if err != nil {
		return err
	}
	byKey := make(map[string]*ent.Permission, len(existing))
	for _, item := range existing {
		byKey[item.Key] = item
	}
	for _, definition := range access.Manifest() {
		item := byKey[definition.Key]
		if item == nil {
			_, err = client.Permission.Create().SetKey(definition.Key).SetName(definition.Name).SetGroup(definition.Group).SetDescription(definition.Description).Save(ctx)
		} else if item.Name != definition.Name || item.Group != definition.Group || item.Description != definition.Description {
			_, err = client.Permission.UpdateOneID(item.ID).SetName(definition.Name).SetGroup(definition.Group).SetDescription(definition.Description).Save(ctx)
		}
		if err != nil {
			return fmt.Errorf("同步权限 %s: %w", definition.Key, err)
		}
	}
	return nil
}

// uniqueAdministratorSystem 拒绝多系统工作台歧义，不按配置编码建立第二套系统资格。
func uniqueAdministratorSystem(organizations []*ent.Organization) (*ent.Organization, error) {
	var system *ent.Organization
	for _, item := range organizations {
		if item.Kind != organization.KindSystem {
			continue
		}
		if system != nil {
			return nil, biz.ErrAdminBootstrapSystemInactive
		}
		system = item
	}
	return system, nil
}

// prepareCompanyInitialAdministrator 在新公司写入前复核完整的系统资格；缺失项只由 bootstrap 补齐。
func prepareCompanyInitialAdministrator(ctx context.Context, client *ent.Client) (*ent.User, error) {
	account, err := lockInitialAdministrator(ctx, client)
	if err != nil {
		return nil, err
	}
	organizations, err := client.Organization.Query().Order(organization.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return nil, err
	}
	system, err := uniqueAdministratorSystem(organizations)
	if err != nil {
		return nil, err
	}
	if system == nil || !system.Enabled {
		return nil, biz.ErrAdminBootstrapSystemInactive
	}
	plans, err := prepareAdministratorPlans(ctx, client, account, organizations, system.ID)
	if err != nil {
		return nil, err
	}
	for _, plan := range plans {
		if plan.organization.ID != system.ID {
			continue
		}
		if plan.member == nil || plan.adminRole == nil {
			return nil, biz.ErrAdminBootstrapSystemRequired
		}
		bound, err := client.RoleAssignment.Query().Where(roleassignment.MembershipIDEQ(plan.member.ID), roleassignment.RoleIDEQ(plan.adminRole.ID)).Exist(ctx)
		if err != nil {
			return nil, err
		}
		if !bound {
			return nil, biz.ErrAdminBootstrapSystemRequired
		}
	}
	return account, nil
}
