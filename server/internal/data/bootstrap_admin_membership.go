package data

import (
	"context"
	"database/sql"
	"fmt"

	"entgo.io/ent/dialect"
	entsql "entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	roleassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// bootstrap 管理员默认组织覆盖策略：每个公司工作台各持有一份 administrator 角色，
// bootstrap 管理员在其中持有启用成员关系并挂接该角色。往来单位创建等路径要求
// 操作者在目标公司子树内持有真实成员关系，仅靠 IsBootstrapAdmin 的权限旁路不满足，
// 因此建公司与迁移收敛两条路径都统一执行本补齐逻辑。
const bootstrapAdministratorRoleCode = "administrator"

// BootstrapAdminMembershipSyncSummary 汇报一次补齐的增量，供迁移命令输出。
type BootstrapAdminMembershipSyncSummary struct {
	Companies              int
	CreatedRoles           int
	CreatedMemberships     int
	ReenabledMemberships   int
	CreatedRoleAssignments int
}

func (s *BootstrapAdminMembershipSyncSummary) add(other *BootstrapAdminMembershipSyncSummary) {
	s.Companies += other.Companies
	s.CreatedRoles += other.CreatedRoles
	s.CreatedMemberships += other.CreatedMemberships
	s.ReenabledMemberships += other.ReenabledMemberships
	s.CreatedRoleAssignments += other.CreatedRoleAssignments
}

// ensureBootstrapAdminCompanyMembership 幂等补齐单个公司内的 administrator 角色、
// bootstrap 管理员成员关系与角色分配。client 由调用方提供；需要事务时传入
// tx.Client()，并发冲突由 (organization_id, code)、(user_id, organization_id)、
// (membership_id, role_id) 三个唯一索引兜底后转为读取既有行。
func ensureBootstrapAdminCompanyMembership(ctx context.Context, client *ent.Client, organizationID uuid.UUID) (*BootstrapAdminMembershipSyncSummary, error) {
	summary := &BootstrapAdminMembershipSyncSummary{Companies: 1}
	permissions, err := client.Permission.Query().All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询权限目录: %w", err)
	}
	role, err := ensureBootstrapAdministratorRole(ctx, client, organizationID, permissions, summary)
	if err != nil {
		return nil, err
	}
	administrators, err := client.User.Query().
		Where(userent.IsBootstrapAdmin(true), userent.EnabledEQ(true)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询 bootstrap 管理员: %w", err)
	}
	for _, administrator := range administrators {
		if err := ensureBootstrapAdminMembershipAndAssignment(ctx, client, administrator.ID, organizationID, role.ID, summary); err != nil {
			return nil, err
		}
	}
	return summary, nil
}

func ensureBootstrapAdministratorRole(ctx context.Context, client *ent.Client, organizationID uuid.UUID, permissions []*ent.Permission, summary *BootstrapAdminMembershipSyncSummary) (*ent.Role, error) {
	existing, err := client.Role.Query().
		Where(roleent.OrganizationIDEQ(organizationID), roleent.CodeEQ(bootstrapAdministratorRoleCode)).
		WithPermissions().
		Only(ctx)
	if err == nil {
		held := make(map[uuid.UUID]struct{}, len(existing.Edges.Permissions))
		for _, permission := range existing.Edges.Permissions {
			held[permission.ID] = struct{}{}
		}
		missing := make([]*ent.Permission, 0, len(permissions))
		for _, permission := range permissions {
			if _, ok := held[permission.ID]; !ok {
				missing = append(missing, permission)
			}
		}
		if len(missing) > 0 {
			if _, updateErr := existing.Update().AddPermissions(missing...).Save(ctx); updateErr != nil {
				return nil, fmt.Errorf("补挂系统管理员角色权限: %w", updateErr)
			}
		}
		return existing, nil
	}
	if !ent.IsNotFound(err) {
		return nil, fmt.Errorf("查询系统管理员角色: %w", err)
	}
	created, createErr := client.Role.Create().
		SetOrganizationID(organizationID).
		SetCode(bootstrapAdministratorRoleCode).
		SetName("系统管理员").
		SetDataScope(roleent.DataScopeAll).
		SetEnabled(true).
		AddPermissions(permissions...).
		Save(ctx)
	if createErr == nil {
		summary.CreatedRoles++
		return created, nil
	}
	if !ent.IsConstraintError(createErr) {
		return nil, fmt.Errorf("创建系统管理员角色: %w", createErr)
	}
	// 并发建角色撞唯一索引：收敛为读取既有行继续补成员关系。
	raced, queryErr := client.Role.Query().
		Where(roleent.OrganizationIDEQ(organizationID), roleent.CodeEQ(bootstrapAdministratorRoleCode)).
		Only(ctx)
	if queryErr != nil {
		return nil, fmt.Errorf("并发创建系统管理员角色后读取失败: %w", queryErr)
	}
	return raced, nil
}

func ensureBootstrapAdminMembershipAndAssignment(ctx context.Context, client *ent.Client, userID, organizationID, roleID uuid.UUID, summary *BootstrapAdminMembershipSyncSummary) error {
	membership, err := client.Membership.Query().
		Where(membershipent.UserIDEQ(userID), membershipent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	switch {
	case err == nil:
		if !membership.Enabled {
			if _, updateErr := membership.Update().SetEnabled(true).Save(ctx); updateErr != nil {
				return fmt.Errorf("重新启用管理员成员关系: %w", updateErr)
			}
			summary.ReenabledMemberships++
		}
	case ent.IsNotFound(err):
		created, createErr := client.Membership.Create().
			SetUserID(userID).
			SetOrganizationID(organizationID).
			SetEnabled(true).
			Save(ctx)
		if createErr != nil {
			if !ent.IsConstraintError(createErr) {
				return fmt.Errorf("创建管理员成员关系: %w", createErr)
			}
			// 并发建成员关系撞唯一索引：收敛为读取既有行继续挂角色。
			membership, err = client.Membership.Query().
				Where(membershipent.UserIDEQ(userID), membershipent.OrganizationIDEQ(organizationID)).
				Only(ctx)
			if err != nil {
				return fmt.Errorf("并发创建管理员成员关系后读取失败: %w", err)
			}
		} else {
			membership = created
			summary.CreatedMemberships++
		}
	default:
		return fmt.Errorf("查询管理员成员关系: %w", err)
	}
	assigned, err := client.RoleAssignment.Query().
		Where(roleassignmentent.MembershipIDEQ(membership.ID), roleassignmentent.RoleIDEQ(roleID)).
		Exist(ctx)
	if err != nil {
		return fmt.Errorf("查询管理员角色分配: %w", err)
	}
	if assigned {
		return nil
	}
	if _, createErr := client.RoleAssignment.Create().
		SetMembershipID(membership.ID).
		SetRoleID(roleID).
		Save(ctx); createErr != nil && !ent.IsConstraintError(createErr) {
		return fmt.Errorf("创建管理员角色分配: %w", createErr)
	} else if createErr == nil {
		summary.CreatedRoleAssignments++
	}
	return nil
}

// SyncBootstrapAdminCompanyMemberships 对当前全部启用公司执行幂等补齐，供
// cmd/migrate 在迁移完成后收敛存量环境，保证「bootstrap 管理员存在于每个公司」
// 的默认覆盖状态。每个公司独立提交，单公司失败不影响其他公司。
func SyncBootstrapAdminCompanyMemberships(ctx context.Context, db *sql.DB) (*BootstrapAdminMembershipSyncSummary, error) {
	client := ent.NewClient(ent.Driver(entsql.OpenDB(dialect.Postgres, db)))
	defer client.Close()
	organizations, err := client.Organization.Query().
		Where(organizationent.KindEQ(organizationent.KindCompany), organizationent.EnabledEQ(true)).
		All(ctx)
	if err != nil {
		return nil, fmt.Errorf("查询公司组织: %w", err)
	}
	total := &BootstrapAdminMembershipSyncSummary{}
	for _, organization := range organizations {
		var summary *BootstrapAdminMembershipSyncSummary
		err := runTransaction(func() (*ent.Tx, error) {
			return client.Tx(ctx)
		}, func(tx *ent.Tx) error {
			var ensureErr error
			summary, ensureErr = ensureBootstrapAdminCompanyMembership(ctx, tx.Client(), organization.ID)
			return ensureErr
		})
		if err != nil {
			return nil, fmt.Errorf("补齐公司 %s 的管理员覆盖: %w", organization.Code, err)
		}
		total.add(summary)
	}
	return total, nil
}
