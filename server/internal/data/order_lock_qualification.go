package data

import (
	"context"
	"sort"
	"strings"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	roleassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

// ensureSharedMBLNotLocked 检查共享 MBL 下的所有活动成员订单是否被锁定；任一被锁定则整体阻断。
func ensureSharedMBLNotLocked(ctx context.Context, tx *ent.Tx, masterBillID uuid.UUID) error {
	links, err := tx.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.MasterBillIDEQ(masterBillID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		All(ctx)
	if err != nil {
		return err
	}
	if len(links) == 0 {
		return nil
	}
	orderIDs := make([]uuid.UUID, 0, len(links))
	for _, l := range links {
		orderIDs = append(orderIDs, l.OrderID)
	}
	// 按照 UUID 升序排序
	sort.Slice(orderIDs, func(i, j int) bool {
		return orderIDs[i].String() < orderIDs[j].String()
	})

	orders, err := tx.Order.Query().
		Where(orderent.IDIn(orderIDs...)).
		Order(ent.Asc(orderent.FieldID)).
		ForUpdate().
		All(ctx)
	if err != nil {
		return err
	}

	var lockedOrderNos []string
	for _, o := range orders {
		if o.LockedAt != nil {
			lockedOrderNos = append(lockedOrderNos, o.OrderNo)
		}
	}
	if len(lockedOrderNos) > 0 {
		sort.Strings(lockedOrderNos)
		return biz.NewErrSeaMasterBillMemberOrderLocked(len(lockedOrderNos), lockedOrderNos)
	}
	return nil
}

// companyGrantMembershipIDs 返回当前启用公司及其启用部门/团队，遇到其他公司即截止。
func companyGrantMembershipIDs(ctx context.Context, client *ent.Client, companyID uuid.UUID) ([]uuid.UUID, error) {
	return companyPersonnelOrganizationIDs(ctx, client, companyID)
}

// qualifiedBusinessLockGrantPredicate 要求同公司角色、同公司启用成员资格及显式权限。
// 系统角色和其他公司的成员不得借由权限配置进入经营审批候选池。
func qualifiedBusinessLockGrantPredicate(permissionKey string, targetOrganizationID uuid.UUID, membershipIDs []uuid.UUID, membershipPredicates ...predicate.Membership) predicate.RoleAssignment {
	baseMembership := []predicate.Membership{membershipent.EnabledEQ(true), membershipent.OrganizationIDIn(membershipIDs...)}
	baseMembership = append(baseMembership, membershipPredicates...)
	rolePredicates := []predicate.Role{roleent.EnabledEQ(true), roleent.OrganizationIDEQ(targetOrganizationID), roleent.HasPermissionsWith(permissionent.KeyEQ(permissionKey))}
	if strings.HasPrefix(permissionKey, "business.order.") && strings.HasSuffix(permissionKey, ".lock") {
		base := strings.TrimSuffix(permissionKey, ".lock")
		for _, required := range []string{base + ".read", base + ".update", base + ".fee.read"} {
			rolePredicates = append(rolePredicates, roleent.HasPermissionsWith(permissionent.KeyEQ(required)))
		}
	}
	return roleassignmentent.And(
		roleassignmentent.HasMembershipWith(baseMembership...),
		roleassignmentent.HasRoleWith(rolePredicates...),
	)
}

// isUserQualifiedBusinessLockRole 验证用户是否具备目标业务类型订单锁定的有效
// lock grant。bootstrap admin 不经本函数判定，由调用方显式分流。
func isUserQualifiedBusinessLockRole(ctx context.Context, client *ent.Client, organizationID, userID uuid.UUID, businessType access.OrderBusinessType) (bool, error) {
	permissionKey := access.OrderPermission(businessType, access.OrderLock)
	if permissionKey == "" {
		return false, biz.ErrOrderBusinessUnsupported
	}
	ancestorIDs, err := companyGrantMembershipIDs(ctx, client, organizationID)
	if err != nil {
		return false, err
	}
	exists, err := client.RoleAssignment.Query().
		Where(
			qualifiedBusinessLockGrantPredicate(permissionKey, organizationID, ancestorIDs,
				membershipent.UserIDEQ(userID),
				membershipent.HasUserWith(
					userent.EnabledEQ(true),
					userent.IsBootstrapAdminEQ(false),
				),
			),
		).
		Exist(ctx)
	if err != nil {
		return false, err
	}
	return exists, nil
}

// candidateInfo 封装审批候选人解析结果。
type candidateInfo struct {
	UserID                 uuid.UUID
	DisplayName            string
	DingTalkUserIDSnapshot string
	MembershipID           uuid.UUID
	RoleID                 uuid.UUID
}

// queryQualifiedBusinessLockCandidates 查询具备目标业务类型锁定「有效 lock
// grant」的审批候选人列表，与单用户判断复用同一资格口径。bootstrap admin 不进
// 入候选池：它没有可写入候选快照的常规成员关系/角色事实，回调也不得绕过快照。
// 同一用户存在多个合格 grant 时按稳定顺序（RoleAssignment ID 升序）选择首个
// 可追溯 grant。
func queryQualifiedBusinessLockCandidates(ctx context.Context, client *ent.Client, organizationID uuid.UUID, businessType access.OrderBusinessType) ([]*candidateInfo, error) {
	permissionKey := access.OrderPermission(businessType, access.OrderLock)
	if permissionKey == "" {
		return nil, biz.ErrOrderBusinessUnsupported
	}
	ancestorIDs, err := companyGrantMembershipIDs(ctx, client, organizationID)
	if err != nil {
		return nil, err
	}
	assignments, err := client.RoleAssignment.Query().
		Where(
			qualifiedBusinessLockGrantPredicate(permissionKey, organizationID, ancestorIDs,
				membershipent.HasUserWith(
					userent.EnabledEQ(true),
					userent.IsBootstrapAdminEQ(false),
				),
			),
		).
		WithMembership(func(mq *ent.MembershipQuery) {
			mq.WithUser()
		}).
		WithRole().
		Order(roleassignmentent.ByID()).
		All(ctx)
	if err != nil {
		return nil, err
	}

	seen := make(map[uuid.UUID]struct{})
	candidates := make([]*candidateInfo, 0, len(assignments))
	for _, a := range assignments {
		if a.Edges.Membership == nil || a.Edges.Membership.Edges.User == nil || a.Edges.Role == nil {
			continue
		}
		u := a.Edges.Membership.Edges.User
		if _, ok := seen[u.ID]; ok {
			continue
		}
		seen[u.ID] = struct{}{}

		dtID := ""
		if u.DingtalkUserid != nil {
			dtID = *u.DingtalkUserid
		}
		candidates = append(candidates, &candidateInfo{
			UserID:                 u.ID,
			DisplayName:            u.DisplayName,
			DingTalkUserIDSnapshot: dtID,
			MembershipID:           a.Edges.Membership.ID,
			RoleID:                 a.Edges.Role.ID,
		})
	}

	// 稳定排序：按 UserID 升序
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].UserID.String() < candidates[j].UserID.String()
	})
	return candidates, nil
}
