package data

import (
	"context"
	"slices"
	"sort"

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

// organizationLockGrantAncestorIDs 沿 parent 链返回目标组织的全部祖先组织 ID
// （不含目标组织自身）。带环保护；组织行缺失时视为链终止，由调用方的成员关系
// 条件自然 fail-closed。
func organizationLockGrantAncestorIDs(ctx context.Context, client *ent.Client, organizationID uuid.UUID) ([]uuid.UUID, error) {
	ancestors := make([]uuid.UUID, 0, 4)
	seen := map[uuid.UUID]struct{}{organizationID: {}}
	current := organizationID
	for {
		org, err := client.Organization.Get(ctx, current)
		if err != nil {
			if ent.IsNotFound(err) {
				return ancestors, nil
			}
			return nil, err
		}
		if org.ParentID == nil {
			return ancestors, nil
		}
		parent := *org.ParentID
		if _, visited := seen[parent]; visited {
			return ancestors, nil
		}
		seen[parent] = struct{}{}
		ancestors = append(ancestors, parent)
		current = parent
	}
}

// qualifiedBusinessLockGrantPredicate 构造「有效 lock grant」的统一资格谓词，
// 单用户判断（isUserQualifiedBusinessLockRole）与候选查询
// （queryQualifiedBusinessLockCandidates）必须复用同一口径：
//   - 用户 enabled、非 bootstrap（bootstrap 由调用方显式分流）；
//   - membership enabled；
//   - 角色 enabled 且真实持有目标业务类型 lock 权限（不按角色代码排除
//     administrator 等任何角色）；
//   - 角色数据范围覆盖目标订单组织：ORGANIZATION 要求 membership 组织即目标
//     组织；ORGANIZATION_TREE 要求 membership 组织是目标组织或其祖先；
//     ALL 覆盖任意组织。
//
// membershipPredicates 由调用方追加用户过滤等条件。
func qualifiedBusinessLockGrantPredicate(permissionKey string, targetOrganizationID uuid.UUID, ancestorIDs []uuid.UUID, membershipPredicates ...predicate.Membership) predicate.RoleAssignment {
	baseMembership := make([]predicate.Membership, 0, len(membershipPredicates)+2)
	baseMembership = append(baseMembership, membershipent.EnabledEQ(true))
	baseMembership = append(baseMembership, membershipPredicates...)
	roleQualifies := func(scope roleent.DataScope) []predicate.Role {
		return []predicate.Role{
			roleent.EnabledEQ(true),
			roleent.HasPermissionsWith(permissionent.KeyEQ(permissionKey)),
			roleent.DataScopeEQ(scope),
		}
	}
	treeOrganizationIDs := make([]uuid.UUID, 0, len(ancestorIDs)+1)
	treeOrganizationIDs = append(treeOrganizationIDs, targetOrganizationID)
	treeOrganizationIDs = append(treeOrganizationIDs, ancestorIDs...)
	return roleassignmentent.Or(
		roleassignmentent.And(
			roleassignmentent.HasMembershipWith(baseMembership...),
			roleassignmentent.HasRoleWith(roleQualifies(roleent.DataScopeAll)...),
		),
		roleassignmentent.And(
			roleassignmentent.HasMembershipWith(append(slices.Clone(baseMembership), membershipent.OrganizationIDEQ(targetOrganizationID))...),
			roleassignmentent.HasRoleWith(roleQualifies(roleent.DataScopeOrganization)...),
		),
		roleassignmentent.And(
			roleassignmentent.HasMembershipWith(append(slices.Clone(baseMembership), membershipent.OrganizationIDIn(treeOrganizationIDs...))...),
			roleassignmentent.HasRoleWith(roleQualifies(roleent.DataScopeOrganizationTree)...),
		),
	)
}

// isUserQualifiedBusinessLockRole 验证用户是否具备目标业务类型订单锁定的有效
// lock grant。bootstrap admin 不经本函数判定，由调用方显式分流。
func isUserQualifiedBusinessLockRole(ctx context.Context, client *ent.Client, organizationID, userID uuid.UUID, businessType access.OrderBusinessType) (bool, error) {
	permissionKey := access.OrderPermission(businessType, access.OrderLock)
	if permissionKey == "" {
		return false, biz.ErrOrderBusinessUnsupported
	}
	ancestorIDs, err := organizationLockGrantAncestorIDs(ctx, client, organizationID)
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
	ancestorIDs, err := organizationLockGrantAncestorIDs(ctx, client, organizationID)
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
