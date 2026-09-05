package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	backgroundtaskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	dingtalkapprovaldispatchent "github.com/roncin/roncin-go-admin/server/internal/data/ent/dingtalkapprovaldispatch"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	orderunlockrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderunlockrequest"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	roleassignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type orderLockRepo struct {
	data     *Data
	security *conf.Security
}

// NewOrderLockRepo 创建订单锁数据仓储实现。
func NewOrderLockRepo(data *Data, security *conf.Security) biz.OrderLockRepo {
	return &orderLockRepo{data: data, security: security}
}

// orderAccessBusinessType 把订单持久化类型收敛为权限目录类型，拒绝任何非法值。
func orderAccessBusinessType(businessType orderent.BusinessType) (access.OrderBusinessType, error) {
	parsed := access.OrderBusinessType(businessType)
	if !parsed.Valid() {
		return "", biz.ErrOrderBusinessUnsupported
	}
	return parsed, nil
}

// ensureOrderBusinessEditable 统一订单锁门禁：检查订单是否处于未锁定状态。若已锁定，返回结构化 ORDER_BUSINESS_LOCKED 错误。
func ensureOrderBusinessEditable(ctx context.Context, tx *ent.Tx, existing *ent.Order) error {
	if existing == nil {
		return nil
	}
	if _, err := orderAccessBusinessType(existing.BusinessType); err != nil {
		return err
	}
	if existing.LockedAt == nil {
		return nil
	}
	var lockedByName string
	if existing.LockedBy != nil {
		u, err := tx.User.Get(ctx, *existing.LockedBy)
		if err == nil && u != nil {
			lockedByName = u.DisplayName
		}
	}
	return biz.NewErrOrderBusinessLocked(existing.ID, existing.OrderNo, existing.LockGeneration, *existing.LockedAt, lockedByName)
}

// lockOrderAndEnsureBusinessEditable 在事务中以 FOR UPDATE 读取订单并校验是否可业务编辑。
func lockOrderAndEnsureBusinessEditable(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID) (*ent.Order, error) {
	existing, err := tx.Order.Query().
		Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
		ForUpdate().
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	if err := ensureOrderBusinessEditable(ctx, tx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

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

// isUserQualifiedBusinessLockRole 验证用户是否具备目标业务类型订单锁定的有效业务角色资格。
func isUserQualifiedBusinessLockRole(ctx context.Context, client *ent.Client, organizationID, userID uuid.UUID, businessType access.OrderBusinessType) (bool, error) {
	permissionKey := access.OrderPermission(businessType, access.OrderLock)
	if permissionKey == "" {
		return false, biz.ErrOrderBusinessUnsupported
	}
	u, err := client.User.Query().
		Where(
			userent.IDEQ(userID),
			userent.EnabledEQ(true),
			userent.IsBootstrapAdminEQ(false),
		).
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}

	exists, err := client.RoleAssignment.Query().
		Where(
			roleassignmentent.HasMembershipWith(
				membershipent.UserIDEQ(u.ID),
				membershipent.OrganizationIDEQ(organizationID),
				membershipent.EnabledEQ(true),
			),
			roleassignmentent.HasRoleWith(
				roleent.OrganizationIDEQ(organizationID),
				roleent.EnabledEQ(true),
				roleent.CodeNEQ("administrator"),
				roleent.HasPermissionsWith(
					permissionent.KeyEQ(permissionKey),
				),
				roleent.DataScopeIn(roleent.DataScopeAll, roleent.DataScopeOrganizationTree, roleent.DataScopeOrganization),
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

// queryQualifiedBusinessLockCandidates 查询组织内所有具备目标业务类型锁定权限的有效业务角色成员候选列表。
func queryQualifiedBusinessLockCandidates(ctx context.Context, tx *ent.Tx, organizationID uuid.UUID, businessType access.OrderBusinessType) ([]*candidateInfo, error) {
	permissionKey := access.OrderPermission(businessType, access.OrderLock)
	if permissionKey == "" {
		return nil, biz.ErrOrderBusinessUnsupported
	}
	assignments, err := tx.RoleAssignment.Query().
		Where(
			roleassignmentent.HasMembershipWith(
				membershipent.OrganizationIDEQ(organizationID),
				membershipent.EnabledEQ(true),
				membershipent.HasUserWith(
					userent.EnabledEQ(true),
					userent.IsBootstrapAdminEQ(false),
				),
			),
			roleassignmentent.HasRoleWith(
				roleent.OrganizationIDEQ(organizationID),
				roleent.EnabledEQ(true),
				roleent.CodeNEQ("administrator"),
				roleent.HasPermissionsWith(
					permissionent.KeyEQ(permissionKey),
				),
				roleent.DataScopeIn(roleent.DataScopeAll, roleent.DataScopeOrganizationTree, roleent.DataScopeOrganization),
			),
		).
		WithMembership(func(mq *ent.MembershipQuery) {
			mq.WithUser()
		}).
		WithRole().
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

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func derefInt(i *int) int {
	if i == nil {
		return 0
	}
	return *i
}

func derefFloat(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func derefUUID(u *uuid.UUID) string {
	if u == nil {
		return ""
	}
	return u.String()
}

func computeMBLContentHash(mbl *ent.SeaMasterBill, exec *ent.SeaTransportExecution) string {
	h := sha256.New()
	fmt.Fprintf(h, "no:%s|norm:%s|issuer:%s|trans:%s|", mbl.MasterNo, mbl.NormalizedMasterNo, mbl.IssuerPartnerID, mbl.TransportExecutionID)
	if exec != nil {
		var etdStr, etaStr string
		if exec.Etd != nil {
			etdStr = exec.Etd.Format(time.RFC3339)
		}
		if exec.Eta != nil {
			etaStr = exec.Eta.Format(time.RFC3339)
		}
		fmt.Fprintf(h, "vessel:%s|voyage:%s|etd:%s|eta:%s|carrier:%s|origin:%s|discharge:%s|transit:%s|",
			exec.VesselName, exec.VoyageNo, etdStr, etaStr,
			derefUUID(exec.CarrierID), derefUUID(exec.OriginLocationID), derefUUID(exec.DischargeLocationID), derefUUID(exec.TransitLocationID))
	}
	fmt.Fprintf(h, "shipper:%s|consignee:%s|notify:%s|notify2:%s|marks:%s|goods:%s|",
		derefStr(mbl.ShipperText), derefStr(mbl.ConsigneeText), derefStr(mbl.NotifyPartyText),
		derefStr(mbl.SecondNotifyPartyText), derefStr(mbl.MarksText), derefStr(mbl.GoodsDescriptionText))
	fmt.Fprintf(h, "pkgs:%d|pkgunit:%s|gw:%.4f|vol:%.4f|freight:%s|trans_terms:%s|bill_form:%s|release:%s|clauses:%s|",
		derefInt(mbl.PackageCount), derefStr(mbl.PackageUnit), derefFloat(mbl.GrossWeightKg), derefFloat(mbl.VolumeCbm),
		derefStr(mbl.FreightTerms), derefStr(mbl.TransportTerms), derefStr(mbl.BillForm), derefStr(mbl.ReleaseType), derefStr(mbl.Clauses))
	return hex.EncodeToString(h.Sum(nil))
}

func computeHBLContentHash(hbl *ent.SeaHouseBill) string {
	h := sha256.New()
	fmt.Fprintf(h, "no:%s|norm:%s|source:%s|issuer_org:%s|issuer_partner:%s|note:%s|",
		hbl.HouseNo, hbl.NormalizedHouseNo, hbl.IssuerSource, derefUUID(hbl.IssuerOrganizationID), derefUUID(hbl.IssuerPartnerID), derefStr(hbl.Note))
	fmt.Fprintf(h, "shipper:%s|consignee:%s|notify:%s|notify2:%s|marks:%s|goods:%s|",
		derefStr(hbl.ShipperText), derefStr(hbl.ConsigneeText), derefStr(hbl.NotifyPartyText),
		derefStr(hbl.SecondNotifyPartyText), derefStr(hbl.MarksText), derefStr(hbl.GoodsDescriptionText))
	fmt.Fprintf(h, "pkgs:%d|pkgunit:%s|gw:%.4f|vol:%.4f|freight:%s|trans_terms:%s|bill_form:%s|release:%s|clauses:%s|",
		derefInt(hbl.PackageCount), derefStr(hbl.PackageUnit), derefFloat(hbl.GrossWeightKg), derefFloat(hbl.VolumeCbm),
		derefStr(hbl.FreightTerms), derefStr(hbl.TransportTerms), derefStr(hbl.BillForm), derefStr(hbl.ReleaseType), derefStr(hbl.Clauses))
	return hex.EncodeToString(h.Sum(nil))
}

func computeLockOrderFingerprint(organizationID, orderID uuid.UUID, expectedVersion uint64, callerID uuid.UUID) string {
	h := sha256.New()
	fmt.Fprintf(h, "org:%s|order:%s|ver:%d|caller:%s", organizationID, orderID, expectedVersion, callerID)
	return hex.EncodeToString(h.Sum(nil))
}

func computeRequestFingerprint(organizationID, orderID uuid.UUID, expectedVersion uint64, callerID uuid.UUID, reason *string) string {
	h := sha256.New()
	r := ""
	if reason != nil {
		r = *reason
	}
	fmt.Fprintf(h, "org:%s|order:%s|ver:%d|caller:%s|reason:%s", organizationID, orderID, expectedVersion, callerID, r)
	return hex.EncodeToString(h.Sum(nil))
}

func findOrderLockRecordByIdempotencyKey(ctx context.Context, client *ent.Client, organizationID uuid.UUID, idempotencyKey string) (*ent.OrderLockRecord, error) {
	record, err := client.OrderLockRecord.Query().
		Where(
			orderlockrecordent.OrganizationIDEQ(organizationID),
			orderlockrecordent.IdempotencyKeyEQ(idempotencyKey),
		).
		First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	return record, err
}

func lockRecordMatchesRequest(record *ent.OrderLockRecord, orderID, callerID uuid.UUID, fingerprint string) bool {
	return record != nil && record.OrderID == orderID && record.LockedBy == callerID && record.RequestFingerprint == fingerprint
}

func findOrderUnlockRequestByIdempotencyKey(ctx context.Context, client *ent.Client, organizationID uuid.UUID, idempotencyKey string) (*ent.OrderUnlockRequest, error) {
	request, err := client.OrderUnlockRequest.Query().
		Where(
			orderunlockrequestent.OrganizationIDEQ(organizationID),
			orderunlockrequestent.IdempotencyKeyEQ(idempotencyKey),
		).
		First(ctx)
	if ent.IsNotFound(err) {
		return nil, nil
	}
	return request, err
}

func unlockRequestMatchesRequest(request *ent.OrderUnlockRequest, orderID, callerID uuid.UUID, fingerprint string) bool {
	return request != nil && request.OrderID == orderID && request.RequestedBy == callerID && request.RequestFingerprint == fingerprint
}

func safeWriteAudit(ctx context.Context, client *ent.AuditLogClient, audit *biz.AuditEvent, organizationID, userID uuid.UUID) error {
	if audit == nil {
		return nil
	}
	if audit.Result == "" {
		audit.Result = "success"
	}
	if audit.OrganizationID == nil {
		audit.OrganizationID = &organizationID
	}
	if audit.UserID == nil {
		audit.UserID = &userID
	}
	return writeAudit(ctx, client, audit)
}

func (r *orderLockRepo) GetOrderLockState(ctx context.Context, organizationID, orderID uuid.UUID, caller *biz.Principal) (*biz.OrderLockState, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	order, err := client.Order.Query().
		Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
		Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	businessType, err := orderAccessBusinessType(order.BusinessType)
	if err != nil {
		return nil, err
	}
	updatePermission := access.OrderPermission(businessType, access.OrderUpdate)

	isLocked := order.LockedAt != nil
	var lockedByName *string
	if order.LockedBy != nil {
		u, err := client.User.Get(ctx, *order.LockedBy)
		if err == nil && u != nil {
			lockedByName = &u.DisplayName
		}
	}

	state := &biz.OrderLockState{
		OrderID:        order.ID,
		OrderNo:        order.OrderNo,
		BusinessType:   biz.OrderBusinessType(businessType),
		IsLocked:       isLocked,
		LockGeneration: order.LockGeneration,
		LockedAt:       order.LockedAt,
		LockedBy:       order.LockedBy,
		LockedByName:   lockedByName,
		OrderVersion:   order.Version,
	}

	// 读取当前锁定记录（若存在）
	if order.LockGeneration > 0 {
		rec, err := client.OrderLockRecord.Query().
			Where(
				orderlockrecordent.OrderIDEQ(order.ID),
				orderlockrecordent.GenerationEQ(order.LockGeneration),
			).
			WithHouseBillSnapshots().
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return nil, err
		}
		if rec != nil {
			if access.OrderBusinessType(rec.BusinessType) != businessType {
				return nil, biz.ErrOrderStatusConflict
			}
			state.CurrentLockRecord = r.mapLockRecord(ctx, client, rec)
		}
	}

	// 读取当前活动解锁请求
	activeReq, err := client.OrderUnlockRequest.Query().
		Where(
			orderunlockrequestent.OrderIDEQ(order.ID),
			orderunlockrequestent.LockGenerationEQ(order.LockGeneration),
			orderunlockrequestent.StatusIn(
				biz.UnlockStatusPendingDispatch,
				biz.UnlockStatusPendingApproval,
				biz.UnlockStatusApprovedPendingApply,
				biz.UnlockStatusDispatchUnknown,
			),
		).
		WithApproverCandidates().
		First(ctx)
	if err != nil && !ent.IsNotFound(err) {
		return nil, err
	}
	if activeReq != nil {
		if access.OrderBusinessType(activeReq.BusinessType) != businessType {
			return nil, biz.ErrOrderStatusConflict
		}
		state.ActiveUnlockRequest = r.mapUnlockRequest(ctx, client, activeReq)
	}

	// 判断调用人角色资格
	isQualifiedBusinessRole := false
	if caller != nil && !caller.IsBootstrapAdmin {
		isQualifiedBusinessRole, err = isUserQualifiedBusinessLockRole(ctx, client, organizationID, caller.UserID, businessType)
		if err != nil {
			return nil, err
		}
	}

	// 只有 SE 锁定会形成海运单证版本，因此只对 SE 要求活动提单关系。
	hasActiveLink := true
	if businessType == access.OrderBusinessSE {
		hasActiveLink, err = client.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(organizationID),
				seamasterbillorderlinkent.OrderIDEQ(orderID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Exist(ctx)
		if err != nil {
			return nil, err
		}
	}

	// 计算可执行动作与阻断原因
	if !isLocked {
		if order.TerminationStatus != orderent.TerminationStatusACTIVE {
			state.LockBlockedReasons = append(state.LockBlockedReasons, "订单已进入终止流程")
		}
		if order.ClosureStatus != orderent.ClosureStatusOPEN {
			state.LockBlockedReasons = append(state.LockBlockedReasons, "订单已结案")
		}
		if businessType == access.OrderBusinessSE && !hasActiveLink {
			state.LockBlockedReasons = append(state.LockBlockedReasons, "订单未关联有效海运提单")
		}
		if !isQualifiedBusinessRole {
			state.LockBlockedReasons = append(state.LockBlockedReasons, "当前用户未分配对应业务类型的订单锁定角色")
		}
		state.CanLock = len(state.LockBlockedReasons) == 0
	} else {
		state.LockBlockedReasons = append(state.LockBlockedReasons, "订单已被锁定")

		if caller != nil {
			if caller.IsBootstrapAdmin {
				state.CanAdminEmergencyUnlock = true
			} else if isQualifiedBusinessRole {
				state.CanRoleDirectUnlock = true
			} else if caller.HasPermissionInScope(updatePermission, biz.DataScopeOrganization) && caller.CanAccessOrderOrganization(order.OrganizationID, true) {
				if state.ActiveUnlockRequest == nil {
					state.CanRequestUnlock = true
				} else {
					state.UnlockBlockedReasons = append(state.UnlockBlockedReasons, "当前已有生效中或审批中的解锁请求")
				}
			} else {
				state.UnlockBlockedReasons = append(state.UnlockBlockedReasons, "无订单编辑权限")
			}
		}
	}

	return state, nil
}

func (r *orderLockRepo) LockOrder(ctx context.Context, caller *biz.Principal, orderID uuid.UUID, expectedOrderVersion uint64, idempotencyKey string, audit *biz.AuditEvent) (*biz.OrderLockResult, error) {
	if caller == nil {
		return nil, biz.ErrOrderLockRoleRequired
	}
	organizationID := caller.Organization.ID
	fingerprint := computeLockOrderFingerprint(organizationID, orderID, expectedOrderVersion, caller.UserID)

	var resultRecord *ent.OrderLockRecord
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 1. 锁序 1: Order FOR UPDATE，并以数据库中的权威业务类型解析权限。
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		businessType, parseErr := orderAccessBusinessType(order.BusinessType)
		if parseErr != nil {
			return parseErr
		}
		qualified, qualificationErr := isUserQualifiedBusinessLockRole(ctx, tx.Client(), organizationID, caller.UserID, businessType)
		if qualificationErr != nil {
			return qualificationErr
		}
		if !qualified {
			return biz.ErrOrderLockRoleRequired
		}

		// 取得 Order 锁后重查幂等记录，确保两个并发同键请求中等待者能看到先提交的事实，
		// 同时避免先锁幂等事实再锁 Order 形成反向锁序。
		existingRec, recErr := findOrderLockRecordByIdempotencyKey(ctx, tx.Client(), organizationID, idempotencyKey)
		if recErr != nil {
			return recErr
		}
		if existingRec != nil {
			if lockRecordMatchesRequest(existingRec, orderID, caller.UserID, fingerprint) {
				resultRecord = existingRec
				return nil
			}
			return biz.ErrOrderStatusConflict
		}
		if order.TerminationStatus != orderent.TerminationStatusACTIVE || order.ClosureStatus != orderent.ClosureStatusOPEN {
			return biz.ErrOrderStatusConflict
		}
		if order.LockedAt != nil {
			return biz.ErrOrderAlreadyLocked
		}
		if order.Version != expectedOrderVersion {
			return biz.ErrOrderStatusConflict
		}

		var seaSnapshot *seaOrderLockSnapshot
		if businessType == access.OrderBusinessSE {
			seaSnapshot, queryErr = createSeaOrderLockSnapshot(ctx, tx, organizationID, orderID, caller.UserID)
			if queryErr != nil {
				return queryErr
			}
		}
		// 8. 更新订单锁状态并推进版本
		newLockGen := order.LockGeneration + 1
		now := time.Now().UTC()
		orderVersionAtLock := order.Version + 1

		if _, err := tx.Order.UpdateOne(order).
			SetLockedAt(now).
			SetLockedBy(caller.UserID).
			SetLockGeneration(newLockGen).
			SetVersion(orderVersionAtLock).
			Save(ctx); err != nil {
			return err
		}

		// 9. 创建 OrderLockRecord
		recordCreate := tx.OrderLockRecord.Create().
			SetOrganizationID(organizationID).
			SetOrderID(order.ID).
			SetOrderNo(order.OrderNo).
			SetBusinessType(orderlockrecordent.BusinessType(businessType)).
			SetGeneration(newLockGen).
			SetLockedBy(caller.UserID).
			SetLockedAt(now).
			SetOrderVersionAtLock(orderVersionAtLock).
			SetIdempotencyKey(idempotencyKey).
			SetRequestFingerprint(fingerprint)
		if seaSnapshot != nil {
			recordCreate.SetMasterBillID(seaSnapshot.MasterBillID).SetMasterBillVersionID(seaSnapshot.MasterBillVersionID)
		}

		rec, err := recordCreate.Save(ctx)
		if err != nil {
			return err
		}
		resultRecord = rec

		// 10. 创建 OrderLockHouseBillSnapshot
		var houseBillSnapshots []seaOrderLockHouseBillSnapshot
		if seaSnapshot != nil {
			houseBillSnapshots = seaSnapshot.HouseBills
		}
		for _, snap := range houseBillSnapshots {
			if _, err := tx.OrderLockHouseBillSnapshot.Create().
				SetOrganizationID(organizationID).
				SetLockRecordID(rec.ID).
				SetHouseBillID(snap.HouseBillID).
				SetHouseBillVersionID(snap.HouseBillVersionID).
				SetHouseNoSnapshot(snap.HouseNoSnapshot).
				Save(ctx); err != nil {
				return err
			}
		}

		// 11. 写入审计日志
		if audit != nil {
			if audit.Action == "" {
				audit.Action = "order.lock"
			}
			audit.Details = map[string]string{
				"business_type":         string(businessType),
				"order_id":              order.ID.String(),
				"order_no":              order.OrderNo,
				"lock_generation":       fmt.Sprintf("%d", newLockGen),
				"order_version_at_lock": fmt.Sprintf("%d", orderVersionAtLock),
			}
			if seaSnapshot != nil {
				audit.Details["master_bill_id"] = seaSnapshot.MasterBillID.String()
				audit.Details["master_bill_version_id"] = seaSnapshot.MasterBillVersionID.String()
				audit.Details["house_bills_count"] = fmt.Sprintf("%d", len(seaSnapshot.HouseBills))
			}
			if err := safeWriteAudit(ctx, tx.AuditLog, audit, organizationID, caller.UserID); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil && ent.IsConstraintError(err) {
		// 不同 Order 可并发使用同一组织级幂等键；唯一索引竞争失败后在事务外读取胜者，
		// 同指纹返回原事实，异指纹映射为稳定 409，而不是泄漏驱动错误。
		client, clientErr := r.data.client(ctx)
		if clientErr != nil {
			return nil, clientErr
		}
		existingRec, lookupErr := findOrderLockRecordByIdempotencyKey(ctx, client, organizationID, idempotencyKey)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if existingRec != nil {
			if !lockRecordMatchesRequest(existingRec, orderID, caller.UserID, fingerprint) {
				return nil, biz.ErrOrderStatusConflict
			}
			resultRecord = existingRec
			err = nil
		}
	}
	if err != nil {
		return nil, err
	}

	state, err := r.GetOrderLockState(ctx, organizationID, orderID, caller)
	if err != nil {
		return nil, err
	}

	lockRecord, err := r.mapLockRecordByID(ctx, resultRecord.ID)
	if err != nil {
		return nil, err
	}
	return &biz.OrderLockResult{
		State:      state,
		LockRecord: lockRecord,
	}, nil
}

func (r *orderLockRepo) RequestOrderUnlock(ctx context.Context, caller *biz.Principal, orderID uuid.UUID, expectedOrderVersion uint64, idempotencyKey string, reason *string, audit *biz.AuditEvent) (*biz.OrderUnlockResult, error) {
	if caller == nil {
		return nil, biz.ErrOrderInvalidArgument
	}
	organizationID := caller.Organization.ID
	fingerprint := computeRequestFingerprint(organizationID, orderID, expectedOrderVersion, caller.UserID, reason)

	var resultRequest *ent.OrderUnlockRequest
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		// 1. 锁序 1: Order FOR UPDATE
		order, queryErr := tx.Order.Query().
			Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).
			ForUpdate().
			Only(ctx)
		if queryErr != nil {
			return mapEntError(queryErr, biz.ErrOrderNotFound, nil)
		}
		businessType, parseErr := orderAccessBusinessType(order.BusinessType)
		if parseErr != nil {
			return parseErr
		}
		if order.LockedAt == nil && order.LockGeneration == 0 {
			return biz.ErrOrderNotLocked
		}

		// 2. 锁定当代 OrderLockRecord。
		lockRec, err := tx.OrderLockRecord.Query().
			Where(
				orderlockrecordent.OrderIDEQ(order.ID),
				orderlockrecordent.GenerationEQ(order.LockGeneration),
			).
			ForUpdate().
			Only(ctx)
		if err != nil {
			return err
		}
		if access.OrderBusinessType(lockRec.BusinessType) != businessType {
			return biz.ErrOrderStatusConflict
		}

		// 3. 查询当前当代已有活动请求。
		activeReq, err := tx.OrderUnlockRequest.Query().
			Where(
				orderunlockrequestent.OrderIDEQ(order.ID),
				orderunlockrequestent.LockGenerationEQ(order.LockGeneration),
				orderunlockrequestent.StatusIn(
					biz.UnlockStatusPendingDispatch,
					biz.UnlockStatusPendingApproval,
					biz.UnlockStatusApprovedPendingApply,
					biz.UnlockStatusDispatchUnknown,
				),
			).
			ForUpdate().
			First(ctx)
		if err != nil && !ent.IsNotFound(err) {
			return err
		}
		if activeReq != nil && access.OrderBusinessType(activeReq.BusinessType) != businessType {
			return biz.ErrOrderStatusConflict
		}

		// 幂等检查位于锁状态和预期版本终态校验之前；取得固定事实锁后重查可覆盖并发直解。
		existingReq, requestErr := findOrderUnlockRequestByIdempotencyKey(ctx, tx.Client(), organizationID, idempotencyKey)
		if requestErr != nil {
			return requestErr
		}
		if existingReq != nil {
			if unlockRequestMatchesRequest(existingReq, orderID, caller.UserID, fingerprint) && access.OrderBusinessType(existingReq.BusinessType) == businessType {
				resultRequest = existingReq
				return nil
			}
			return biz.ErrOrderStatusConflict
		}
		if order.LockedAt == nil || lockRec.UnlockedAt != nil {
			return biz.ErrOrderNotLocked
		}
		if order.Version != expectedOrderVersion {
			return biz.ErrOrderStatusConflict
		}

		now := time.Now().UTC()

		// 5. 分流分支
		// 分支 A: Bootstrap Admin 紧急直接解锁
		if caller.IsBootstrapAdmin {
			var newReqID = uuid.Must(uuid.NewV7())
			newOrderVersion := order.Version + 1
			if _, err := tx.Order.UpdateOne(order).
				ClearLockedAt().
				ClearLockedBy().
				SetVersion(newOrderVersion).
				Save(ctx); err != nil {
				return err
			}

			// 创建 APPROVED 的请求记录
			reqCreate := tx.OrderUnlockRequest.Create().
				SetID(newReqID).
				SetOrganizationID(organizationID).
				SetOrderID(order.ID).
				SetOrderNo(order.OrderNo).
				SetBusinessType(orderunlockrequestent.BusinessType(businessType)).
				SetLockRecordID(lockRec.ID).
				SetLockGeneration(order.LockGeneration).
				SetRequestedBy(caller.UserID).
				SetRequestedAt(now).
				SetNillableReason(reason).
				SetExpectedOrderVersion(expectedOrderVersion).
				SetIdempotencyKey(idempotencyKey).
				SetRequestFingerprint(fingerprint).
				SetRoute(orderunlockrequestent.RouteADMIN_EMERGENCY).
				SetStatus(orderunlockrequestent.StatusAPPROVED).
				SetDecidedBy(caller.UserID).
				SetDecidedAt(now).
				SetDecisionSource("BOOTSTRAP_ADMIN").
				SetUnlockedAt(now).
				SetResultOrderVersion(newOrderVersion)

			savedReq, err := reqCreate.Save(ctx)
			if err != nil {
				return err
			}
			resultRequest = savedReq
			// superseded_by_request_id 是即时外键，必须先保存取代它的新请求。
			if activeReq != nil {
				if _, err := tx.OrderUnlockRequest.UpdateOne(activeReq).
					SetStatus(orderunlockrequestent.StatusSTALE).
					SetSupersededByRequestID(savedReq.ID).
					Save(ctx); err != nil {
					return err
				}
			}

			// 关闭 LockRecord
			recordUpdate := tx.OrderLockRecord.UpdateOne(lockRec).
				SetUnlockedBy(caller.UserID).
				SetUnlockedAt(now).
				SetOrderVersionAtUnlock(newOrderVersion).
				SetUnlockRequestID(savedReq.ID).
				SetUnlockMode(orderlockrecordent.UnlockModeADMIN_EMERGENCY).
				SetNillableUnlockReason(reason)
			if _, err := recordUpdate.Save(ctx); err != nil {
				return err
			}

			// 审计
			if audit != nil {
				if audit.Action == "" {
					audit.Action = "order.unlock.request"
				}
				auditDetails := map[string]string{
					"route":                biz.UnlockRouteAdminEmergency,
					"business_type":        string(businessType),
					"admin_emergency":      "true",
					"order_id":             order.ID.String(),
					"order_no":             order.OrderNo,
					"lock_generation":      fmt.Sprintf("%d", order.LockGeneration),
					"result_order_version": fmt.Sprintf("%d", newOrderVersion),
					"requested_by":         caller.UserID.String(),
					"decided_by":           caller.UserID.String(),
				}
				if reason != nil {
					auditDetails["reason"] = *reason
				}
				audit.Details = auditDetails
				if err := safeWriteAudit(ctx, tx.AuditLog, audit, organizationID, caller.UserID); err != nil {
					return err
				}
			}
			return nil
		}

		// 分支 B: 业务锁定/解锁角色成员直接解锁
		qualifiedRoleMember, err := isUserQualifiedBusinessLockRole(ctx, tx.Client(), organizationID, caller.UserID, businessType)
		if err != nil {
			return err
		}
		if qualifiedRoleMember {
			var newReqID = uuid.Must(uuid.NewV7())
			newOrderVersion := order.Version + 1
			if _, err := tx.Order.UpdateOne(order).
				ClearLockedAt().
				ClearLockedBy().
				SetVersion(newOrderVersion).
				Save(ctx); err != nil {
				return err
			}

			reqCreate := tx.OrderUnlockRequest.Create().
				SetID(newReqID).
				SetOrganizationID(organizationID).
				SetOrderID(order.ID).
				SetOrderNo(order.OrderNo).
				SetBusinessType(orderunlockrequestent.BusinessType(businessType)).
				SetLockRecordID(lockRec.ID).
				SetLockGeneration(order.LockGeneration).
				SetRequestedBy(caller.UserID).
				SetRequestedAt(now).
				SetNillableReason(reason).
				SetExpectedOrderVersion(expectedOrderVersion).
				SetIdempotencyKey(idempotencyKey).
				SetRequestFingerprint(fingerprint).
				SetRoute(orderunlockrequestent.RouteROLE_DIRECT).
				SetStatus(orderunlockrequestent.StatusAPPROVED).
				SetDecidedBy(caller.UserID).
				SetDecidedAt(now).
				SetDecisionSource("ROLE_DIRECT").
				SetUnlockedAt(now).
				SetResultOrderVersion(newOrderVersion)

			savedReq, err := reqCreate.Save(ctx)
			if err != nil {
				return err
			}
			resultRequest = savedReq
			// superseded_by_request_id 是即时外键，必须先保存取代它的新请求，再更新旧申请；
			// 否则角色直解与审批本地生效竞争时会稳定触发外键约束并回滚直解。
			if activeReq != nil {
				if _, err := tx.OrderUnlockRequest.UpdateOne(activeReq).
					SetStatus(orderunlockrequestent.StatusSTALE).
					SetSupersededByRequestID(savedReq.ID).
					Save(ctx); err != nil {
					return err
				}
			}

			recordUpdate := tx.OrderLockRecord.UpdateOne(lockRec).
				SetUnlockedBy(caller.UserID).
				SetUnlockedAt(now).
				SetOrderVersionAtUnlock(newOrderVersion).
				SetUnlockRequestID(savedReq.ID).
				SetUnlockMode(orderlockrecordent.UnlockModeROLE_DIRECT).
				SetNillableUnlockReason(reason)
			if _, err := recordUpdate.Save(ctx); err != nil {
				return err
			}

			if audit != nil {
				if audit.Action == "" {
					audit.Action = "order.unlock.request"
				}
				auditDetails := map[string]string{
					"route":                biz.UnlockRouteRoleDirect,
					"business_type":        string(businessType),
					"order_id":             order.ID.String(),
					"order_no":             order.OrderNo,
					"lock_generation":      fmt.Sprintf("%d", order.LockGeneration),
					"result_order_version": fmt.Sprintf("%d", newOrderVersion),
					"requested_by":         caller.UserID.String(),
					"decided_by":           caller.UserID.String(),
				}
				if reason != nil {
					auditDetails["reason"] = *reason
				}
				audit.Details = auditDetails
				if err := safeWriteAudit(ctx, tx.AuditLog, audit, organizationID, caller.UserID); err != nil {
					return err
				}
			}
			return nil
		}

		// 分支 C: 普通订单编辑人发起钉钉审批
		updatePermission := access.OrderPermission(businessType, access.OrderUpdate)
		if !caller.HasPermissionInScope(updatePermission, biz.DataScopeOrganization) || !caller.CanAccessOrderOrganization(order.OrganizationID, true) {
			return biz.ErrOrderLockRoleRequired
		}

		// 同一锁定代次若已有活动审批，普通编辑人重提返回既有活动请求
		if activeReq != nil {
			resultRequest = activeReq
			return nil
		}

		// 解析全部有效业务角色成员和钉钉绑定
		candidates, err := queryQualifiedBusinessLockCandidates(ctx, tx, organizationID, businessType)
		if err != nil {
			return err
		}

		// 检查申请人钉钉 UserID
		callerUser, err := tx.User.Get(ctx, caller.UserID)
		if err != nil {
			return err
		}
		callerDtID := ""
		if callerUser.DingtalkUserid != nil {
			callerDtID = strings.TrimSpace(*callerUser.DingtalkUserid)
		}
		approvalProcessCode := ""
		approvalCorpID := ""
		approvalEventToken := ""
		approvalEventAESKey := ""
		approvalEnabled := false
		if r.security != nil && r.security.Dingtalk != nil {
			approvalEnabled = r.security.Dingtalk.Enabled
			approvalProcessCode = strings.TrimSpace(r.security.Dingtalk.ApprovalProcessCode)
			approvalCorpID = strings.TrimSpace(r.security.Dingtalk.CorpId)
			approvalEventToken = strings.TrimSpace(r.security.Dingtalk.EventToken)
			approvalEventAESKey = strings.TrimSpace(r.security.Dingtalk.EventAesKey)
		}

		reqStatus := orderunlockrequestent.StatusPENDING_DISPATCH
		var failureCode *string
		var failureMessage *string

		if len(candidates) == 0 {
			reqStatus = orderunlockrequestent.StatusCONFIGURATION_FAILED
			fCode := "ORDER_UNLOCK_APPROVER_NOT_CONFIGURED"
			fMsg := "未配置具备对应业务类型订单锁定权限的业务角色成员"
			failureCode = &fCode
			failureMessage = &fMsg
		} else if callerDtID == "" {
			reqStatus = orderunlockrequestent.StatusCONFIGURATION_FAILED
			fCode := "ORDER_UNLOCK_DINGTALK_NOT_CONFIGURED"
			fMsg := "申请人未绑定钉钉账号"
			failureCode = &fCode
			failureMessage = &fMsg
		} else if !approvalEnabled || approvalProcessCode == "" || approvalCorpID == "" || approvalEventToken == "" || approvalEventAESKey == "" {
			reqStatus = orderunlockrequestent.StatusCONFIGURATION_FAILED
			fCode := "ORDER_UNLOCK_DINGTALK_NOT_CONFIGURED"
			fMsg := "未启用钉钉审批，或审批模板与事件回调配置不完整"
			failureCode = &fCode
			failureMessage = &fMsg
		} else {
			for _, c := range candidates {
				if c.DingTalkUserIDSnapshot == "" {
					reqStatus = orderunlockrequestent.StatusCONFIGURATION_FAILED
					fCode := "ORDER_UNLOCK_DINGTALK_NOT_CONFIGURED"
					fMsg := fmt.Sprintf("审批候选人 %s 未绑定钉钉账号", c.DisplayName)
					failureCode = &fCode
					failureMessage = &fMsg
					break
				}
			}
		}

		reqCreate := tx.OrderUnlockRequest.Create().
			SetOrganizationID(organizationID).
			SetOrderID(order.ID).
			SetOrderNo(order.OrderNo).
			SetBusinessType(orderunlockrequestent.BusinessType(businessType)).
			SetLockRecordID(lockRec.ID).
			SetLockGeneration(order.LockGeneration).
			SetRequestedBy(caller.UserID).
			SetRequestedAt(now).
			SetNillableReason(reason).
			SetExpectedOrderVersion(expectedOrderVersion).
			SetIdempotencyKey(idempotencyKey).
			SetRequestFingerprint(fingerprint).
			SetRoute(orderunlockrequestent.RouteDINGTALK_APPROVAL).
			SetStatus(reqStatus).
			SetNillableFailureCode(failureCode).
			SetNillableFailureMessage(failureMessage)

		savedReq, err := reqCreate.Save(ctx)
		if err != nil {
			return err
		}
		resultRequest = savedReq

		// 保存候选人快照
		for _, c := range candidates {
			if _, err := tx.OrderUnlockApproverCandidate.Create().
				SetRequestID(savedReq.ID).
				SetUserID(c.UserID).
				SetMembershipID(c.MembershipID).
				SetRoleID(c.RoleID).
				SetDisplayNameSnapshot(c.DisplayName).
				SetDingtalkUseridSnapshot(c.DingTalkUserIDSnapshot).
				Save(ctx); err != nil {
				return err
			}
		}

		// 若配置正常，创建后台派发任务与 Outbox
		if reqStatus == orderunlockrequestent.StatusPENDING_DISPATCH {
			candidateDtIDs := make([]string, 0, len(candidates))
			for _, c := range candidates {
				candidateDtIDs = append(candidateDtIDs, c.DingTalkUserIDSnapshot)
			}

			bgTask, bgErr := tx.BackgroundTask.Create().
				SetOrganizationID(organizationID).
				SetKind(backgroundtaskent.KindDINGTALK_APPROVAL_CREATE).
				SetIdempotencyKey(fmt.Sprintf("dt-approval-%s", savedReq.ID.String())).
				SetStatus(backgroundtaskent.StatusPENDING).
				Save(ctx)
			if bgErr != nil {
				return bgErr
			}

			if _, err := tx.DingTalkApprovalDispatch.Create().
				SetOrganizationID(organizationID).
				SetBackgroundTaskID(bgTask.ID).
				SetUnlockRequestID(savedReq.ID).
				SetProcessCodeSnapshot(approvalProcessCode).
				SetApplicantDingtalkUserid(callerDtID).
				SetCandidateDingtalkUserids(candidateDtIDs).
				SetRequestPayloadHash(fingerprint).
				SetDispatchStatus(dingtalkapprovaldispatchent.DispatchStatusPENDING).
				Save(ctx); err != nil {
				return err
			}
		}

		if audit != nil {
			if audit.Action == "" {
				audit.Action = "order.unlock.request"
			}
			auditDetails := map[string]string{
				"route":            biz.UnlockRouteDingTalkApproval,
				"business_type":    string(businessType),
				"status":           string(reqStatus),
				"order_id":         order.ID.String(),
				"order_no":         order.OrderNo,
				"lock_generation":  fmt.Sprintf("%d", order.LockGeneration),
				"requested_by":     caller.UserID.String(),
				"candidates_count": fmt.Sprintf("%d", len(candidates)),
			}
			if reason != nil {
				auditDetails["reason"] = *reason
			}
			audit.Details = auditDetails
			if err := safeWriteAudit(ctx, tx.AuditLog, audit, organizationID, caller.UserID); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil && ent.IsConstraintError(err) {
		client, clientErr := r.data.client(ctx)
		if clientErr != nil {
			return nil, clientErr
		}
		existingReq, lookupErr := findOrderUnlockRequestByIdempotencyKey(ctx, client, organizationID, idempotencyKey)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if existingReq != nil {
			if !unlockRequestMatchesRequest(existingReq, orderID, caller.UserID, fingerprint) {
				return nil, biz.ErrOrderStatusConflict
			}
			resultRequest = existingReq
			err = nil
		}
	}
	if err != nil {
		return nil, err
	}

	state, err := r.GetOrderLockState(ctx, organizationID, orderID, caller)
	if err != nil {
		return nil, err
	}

	request, err := r.mapUnlockRequestByID(ctx, resultRequest.ID)
	if err != nil {
		return nil, err
	}
	return &biz.OrderUnlockResult{
		State:   state,
		Request: request,
	}, nil
}

func (r *orderLockRepo) ListOrderUnlockRequests(ctx context.Context, organizationID, orderID uuid.UUID, page, pageSize int) ([]*biz.OrderUnlockRequest, int, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, 0, err
	}

	query := client.OrderUnlockRequest.Query().
		Where(
			orderunlockrequestent.OrganizationIDEQ(organizationID),
			orderunlockrequestent.OrderIDEQ(orderID),
		)

	total, err := query.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	items, err := query.
		Order(ent.Desc(orderunlockrequestent.FieldCreatedAt), ent.Desc(orderunlockrequestent.FieldID)).
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		WithApproverCandidates().
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	result := make([]*biz.OrderUnlockRequest, 0, len(items))
	for _, item := range items {
		result = append(result, r.mapUnlockRequest(ctx, client, item))
	}
	return result, total, nil
}

func (r *orderLockRepo) GetOrderUnlockRequest(ctx context.Context, organizationID, orderID, requestID uuid.UUID) (*biz.OrderUnlockRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}

	req, err := client.OrderUnlockRequest.Query().
		Where(
			orderunlockrequestent.IDEQ(requestID),
			orderunlockrequestent.OrganizationIDEQ(organizationID),
			orderunlockrequestent.OrderIDEQ(orderID),
		).
		WithApproverCandidates().
		Only(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			return nil, biz.ErrOrderUnlockRequestNotFound
		}
		return nil, err
	}
	return r.mapUnlockRequest(ctx, client, req), nil
}

func (r *orderLockRepo) mapLockRecordByID(ctx context.Context, recordID uuid.UUID) (*biz.OrderLockRecord, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	rec, err := client.OrderLockRecord.Query().
		Where(orderlockrecordent.IDEQ(recordID)).
		WithHouseBillSnapshots().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.mapLockRecord(ctx, client, rec), nil
}

func (r *orderLockRepo) mapLockRecord(ctx context.Context, client *ent.Client, rec *ent.OrderLockRecord) *biz.OrderLockRecord {
	var lockedByName string
	if u, err := client.User.Get(ctx, rec.LockedBy); err == nil && u != nil {
		lockedByName = u.DisplayName
	}
	var unlockedByName *string
	if rec.UnlockedBy != nil {
		if u, err := client.User.Get(ctx, *rec.UnlockedBy); err == nil && u != nil {
			unlockedByName = &u.DisplayName
		}
	}
	var unlockMode *string
	if rec.UnlockMode != nil {
		modeStr := string(*rec.UnlockMode)
		unlockMode = &modeStr
	}

	res := &biz.OrderLockRecord{
		ID:                   rec.ID,
		OrganizationID:       rec.OrganizationID,
		OrderID:              rec.OrderID,
		OrderNo:              rec.OrderNo,
		BusinessType:         biz.OrderBusinessType(rec.BusinessType),
		Generation:           rec.Generation,
		LockedBy:             rec.LockedBy,
		LockedByName:         lockedByName,
		LockedAt:             rec.LockedAt,
		OrderVersionAtLock:   rec.OrderVersionAtLock,
		MasterBillID:         rec.MasterBillID,
		MasterBillVersionID:  rec.MasterBillVersionID,
		UnlockedBy:           rec.UnlockedBy,
		UnlockedByName:       unlockedByName,
		UnlockedAt:           rec.UnlockedAt,
		OrderVersionAtUnlock: rec.OrderVersionAtUnlock,
		UnlockRequestID:      rec.UnlockRequestID,
		UnlockReason:         rec.UnlockReason,
		UnlockMode:           unlockMode,
	}

	for _, s := range rec.Edges.HouseBillSnapshots {
		res.HouseBillSnapshots = append(res.HouseBillSnapshots, &biz.OrderLockHouseBillSnapshot{
			ID:                 s.ID,
			OrganizationID:     s.OrganizationID,
			LockRecordID:       s.LockRecordID,
			HouseBillID:        s.HouseBillID,
			HouseBillVersionID: s.HouseBillVersionID,
			HouseNoSnapshot:    s.HouseNoSnapshot,
			CreatedAt:          s.CreatedAt,
		})
	}
	return res
}

func (r *orderLockRepo) mapUnlockRequestByID(ctx context.Context, requestID uuid.UUID) (*biz.OrderUnlockRequest, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	req, err := client.OrderUnlockRequest.Query().
		Where(orderunlockrequestent.IDEQ(requestID)).
		WithApproverCandidates().
		Only(ctx)
	if err != nil {
		return nil, err
	}
	return r.mapUnlockRequest(ctx, client, req), nil
}

func (r *orderLockRepo) mapUnlockRequest(ctx context.Context, client *ent.Client, req *ent.OrderUnlockRequest) *biz.OrderUnlockRequest {
	var requestedByName string
	if u, err := client.User.Get(ctx, req.RequestedBy); err == nil && u != nil {
		requestedByName = u.DisplayName
	}
	var decidedByName *string
	if req.DecidedBy != nil {
		if u, err := client.User.Get(ctx, *req.DecidedBy); err == nil && u != nil {
			decidedByName = &u.DisplayName
		}
	}

	res := &biz.OrderUnlockRequest{
		ID:                        req.ID,
		OrganizationID:            req.OrganizationID,
		OrderID:                   req.OrderID,
		OrderNo:                   req.OrderNo,
		BusinessType:              biz.OrderBusinessType(req.BusinessType),
		LockRecordID:              req.LockRecordID,
		LockGeneration:            req.LockGeneration,
		RequestedBy:               req.RequestedBy,
		RequestedByName:           requestedByName,
		RequestedAt:               req.RequestedAt,
		Reason:                    req.Reason,
		ExpectedOrderVersion:      req.ExpectedOrderVersion,
		IdempotencyKey:            req.IdempotencyKey,
		RequestFingerprint:        req.RequestFingerprint,
		Route:                     string(req.Route),
		Status:                    string(req.Status),
		DingTalkProcessInstanceID: req.DingtalkProcessInstanceID,
		DingTalkProcessCode:       req.DingtalkProcessCode,
		DecidedBy:                 req.DecidedBy,
		DecidedByName:             decidedByName,
		DecidedAt:                 req.DecidedAt,
		DecisionSource:            req.DecisionSource,
		FailureCode:               req.FailureCode,
		FailureMessage:            req.FailureMessage,
		SupersededByRequestID:     req.SupersededByRequestID,
		UnlockedAt:                req.UnlockedAt,
		ResultOrderVersion:        req.ResultOrderVersion,
	}

	for _, c := range req.Edges.ApproverCandidates {
		res.ApproverCandidates = append(res.ApproverCandidates, &biz.OrderUnlockApproverCandidate{
			ID:                     c.ID,
			RequestID:              c.RequestID,
			UserID:                 c.UserID,
			MembershipID:           c.MembershipID,
			RoleID:                 c.RoleID,
			DisplayNameSnapshot:    c.DisplayNameSnapshot,
			DingTalkUserIDSnapshot: c.DingtalkUseridSnapshot,
			CreatedAt:              c.CreatedAt,
		})
	}
	return res
}
