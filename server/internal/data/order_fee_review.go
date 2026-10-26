package data

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/predicate"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	taskent "github.com/roncin/roncin-go-admin/server/internal/data/ent/backgroundtask"
	lineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebillline"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	noticeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/notificationdelivery"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	feeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	correctionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeecorrectionrequest"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeereviewevent"
	supplementent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeesupplementrequest"
	lockent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	assignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type orderFeeReviewRepo struct{ data *Data }

func NewOrderFeeReviewRepo(data *Data) biz.OrderFeeReviewRepo { return &orderFeeReviewRepo{data: data} }

func hasRealtimeFeePermission(ctx context.Context, client *ent.Client, organizationID, userID uuid.UUID, key string, lock bool) (bool, error) {
	enabled, err := client.Organization.Query().Where(organizationent.IDEQ(organizationID), organizationent.EnabledEQ(true)).Exist(ctx)
	if err != nil || !enabled {
		return false, err
	}
	if lock {
		if err := lockFeeAuthorizationParents(ctx, client, userID); err != nil {
			return false, err
		}
	}
	ancestors, err := companyGrantMembershipIDs(ctx, client, organizationID)
	if err != nil {
		return false, err
	}
	return client.RoleAssignment.Query().Where(qualifiedBusinessLockGrantPredicate(key, organizationID, ancestors, membershipent.UserIDEQ(userID), membershipent.HasUserWith(userent.EnabledEQ(true)), membershipent.HasOrganizationWith(organizationent.EnabledEQ(true)))).Exist(ctx)
}

// lockFeeAuthorizationParents 必须在订单行锁之前调用，与月度申请和账号维护
// 共用 User → Membership → Role → Order 锁序。持有父锁后重读同一授权关联，
// 防止成员、角色或权限在费用决策提交前失效。
func lockFeeAuthorizationParents(ctx context.Context, client *ent.Client, userID uuid.UUID) error {
	if _, err := client.User.Query().Where(userent.IDEQ(userID)).ForShare().Only(ctx); err != nil {
		return mapEntError(err, biz.ErrFeeReviewForbidden, nil)
	}
	if _, err := client.Membership.Query().Where(membershipent.UserIDEQ(userID)).Order(membershipent.ByID()).ForShare().All(ctx); err != nil {
		return err
	}
	_, err := client.Role.Query().Where(roleent.HasAssignmentsWith(assignmentent.HasMembershipWith(membershipent.UserIDEQ(userID)))).Order(roleent.ByID()).ForShare().All(ctx)
	return err
}

func feeReviewReadable(ctx context.Context, client *ent.Client, p *biz.Principal, o *ent.Order) (bool, error) {
	for _, key := range []string{access.FinanceFeeRead, access.OrderPermission(access.OrderBusinessType(o.BusinessType), access.OrderFeeRead), access.OrderPermission(access.OrderBusinessType(o.BusinessType), access.OrderLock)} {
		yes, err := hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, key, false)
		if err != nil {
			return false, err
		}
		if yes {
			return true, nil
		}
	}
	return false, nil
}
func feeReviewSubmitter(ctx context.Context, client *ent.Client, p *biz.Principal, o *ent.Order, lock bool) (bool, error) {
	for _, operation := range []access.OrderOperation{access.OrderFeeCreate, access.OrderFeeUpdate} {
		key := access.OrderPermission(access.OrderBusinessType(o.BusinessType), operation)
		yes, err := hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, key, lock)
		if err != nil {
			return false, err
		}
		if yes {
			return true, nil
		}
	}
	return false, nil
}
func loadFeeReviewSnapshot(ctx context.Context, client *ent.Client, orderID uuid.UUID) (*biz.FeeReviewSnapshot, string, error) {
	items, err := client.OrderFee.Query().Where(feeent.OrderIDEQ(orderID)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Order(feeent.ByID()).All(ctx)
	if err != nil {
		return nil, "", err
	}
	fees := make([]*biz.OrderFee, 0, len(items))
	for _, item := range items {
		fee, err := orderFeeToBiz(item)
		if err != nil {
			return nil, "", err
		}
		fees = append(fees, fee)
	}
	return biz.BuildFeeReviewSnapshot(fees)
}
func hasPendingFeeSupplement(ctx context.Context, client *ent.Client, orderID uuid.UUID) (bool, error) {
	pending, err := client.OrderFeeSupplementRequest.Query().Where(supplementent.OrderIDEQ(orderID), supplementent.StatusEQ(supplementent.StatusPENDING)).Exist(ctx)
	if err != nil || pending {
		return pending, err
	}
	return client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrderIDEQ(orderID), correctionent.StatusEQ(correctionent.StatusPENDING)).Exist(ctx)
}
func currentManualOrderLock(ctx context.Context, client *ent.Client, o *ent.Order) (*ent.OrderLockRecord, error) {
	if o.LockedAt == nil {
		return nil, biz.ErrFeeSupplementNotApplicable
	}
	record, err := client.OrderLockRecord.Query().Where(lockent.OrderIDEQ(o.ID), lockent.OrganizationIDEQ(o.OrganizationID), lockent.GenerationEQ(o.LockGeneration), lockent.LockSourceEQ(lockent.LockSourceMANUAL), lockent.UnlockedAtIsNil()).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrFeeSupplementNotApplicable, nil)
	}
	if len(record.FeeSnapshot) == 0 || record.FeeContentFingerprint == nil {
		return nil, biz.ErrFeeSupplementNotApplicable
	}
	return record, nil
}
func validateOrderApprovalSnapshot(ctx context.Context, client *ent.Client, o *ent.Order) error {
	_, err := currentManualOrderLock(ctx, client, o)
	return err
}
func hasOrderReviewPermission(ctx context.Context, client *ent.Client, org, orderID, userID uuid.UUID, lock bool) (bool, error) {
	o, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(org)).Only(ctx)
	if err != nil {
		return false, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	if lock {
		if err := lockFeeAuthorizationParents(ctx, client, userID); err != nil {
			return false, err
		}
	}
	return isUserQualifiedBusinessLockRole(ctx, client, org, userID, access.OrderBusinessType(o.BusinessType))
}
func projectedFeeReviewStatus(o *ent.Order) string {
	if o.LockedAt != nil {
		return "APPROVED"
	}
	return "UNSUBMITTED"
}
func feeReviewPreviewToken(o *ent.Order, fingerprint string) string {
	return biz.FeeReviewHash([]byte(fmt.Sprintf("fee-review-v1|%s|%d|%d|%v|%s", o.ID, o.FeeContentVersion, o.FeeReviewVersion, o.CurrentFeeReviewEventID, fingerprint)))
}
func (r *orderFeeReviewRepo) Get(ctx context.Context, p *biz.Principal, id uuid.UUID) (*biz.OrderFeeReview, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return nil, err
	}
	o, err := client.Order.Get(ctx, id)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	readable, err := feeReviewReadable(ctx, client, p, o)
	if err != nil {
		return nil, err
	}
	if !readable {
		return nil, biz.ErrFeeReviewForbidden
	}
	return buildOrderFeeReview(ctx, client, p, o, true)
}
func buildOrderFeeReview(ctx context.Context, client *ent.Client, p *biz.Principal, o *ent.Order, history bool) (*biz.OrderFeeReview, error) {
	snapshot, fingerprint, err := loadFeeReviewSnapshot(ctx, client, o.ID)
	if err != nil {
		return nil, err
	}
	fullFees, loadErr := client.OrderFee.Query().Where(feeent.OrderIDEQ(o.ID)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Order(feeent.ByID()).All(ctx)
	if loadErr != nil {
		return nil, loadErr
	}
	displayFees := make([]*biz.OrderFee, 0, len(fullFees))
	for _, f := range fullFees {
		fee, convertErr := orderFeeToBiz(f)
		if convertErr != nil {
			return nil, convertErr
		}
		if biz.IsCurrentFee(fee) {
			displayFees = append(displayFees, fee)
		}
	}
	result := &biz.OrderFeeReview{OrderID: o.ID, OrderNo: o.OrderNo, Status: projectedFeeReviewStatus(o), ContentVersion: o.FeeContentVersion, ReviewVersion: o.FeeReviewVersion, ApprovedEventID: o.ApprovedFeeReviewEventID, Summary: &snapshot.Summary, Fees: displayFees, PreviewToken: feeReviewPreviewToken(o, fingerprint)}
	pending, err := hasPendingFeeSupplement(ctx, client, o.ID)
	if err != nil {
		return nil, err
	}
	result.HasPendingChange = pending
	result.HasPendingCorrection, err = client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrderIDEQ(o.ID), correctionent.StatusEQ(correctionent.StatusPENDING)).Exist(ctx)
	if err != nil {
		return nil, err
	}
	result.HasPendingSupplement = pending && !result.HasPendingCorrection
	canSubmit, err := feeReviewSubmitter(ctx, client, p, o, false)
	if err != nil {
		return nil, err
	}
	result.CanEdit = canSubmit && o.LockedAt == nil && o.TerminationStatus == orderent.TerminationStatusACTIVE
	if o.LockedAt != nil {
		result.BlockedReasons = append(result.BlockedReasons, "订单已审核锁定，费用变更需补审")
	}
	billPermission, err := hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, access.FinanceBillCreate, false)
	if err != nil {
		return nil, err
	}
	result.CanBill = billPermission
	if err := attachFeeCapabilities(ctx, client, o, displayFees); err != nil {
		return nil, err
	}

	if history {
		notices, noticeErr := client.NotificationDelivery.Query().Where(noticeent.ResourceTypeEQ("ORDER"), noticeent.ResourceIDEQ(o.ID), noticeent.TemplateIn(noticeent.TemplateFEE_REVIEW_PENDING, noticeent.TemplateFEE_REVIEW_RESULT)).WithRecipientUser().WithBackgroundTask().All(ctx)
		if noticeErr != nil {
			return nil, noticeErr
		}
		for _, notice := range notices {
			recipient := notice.Edges.RecipientUser
			if recipient.DingtalkUserid == nil || strings.TrimSpace(*recipient.DingtalkUserid) == "" {
				result.NotificationUnboundCount++
			}
			task := notice.Edges.BackgroundTask
			if task.Status == taskent.StatusFAILED || task.Status == taskent.StatusDEAD_LETTER {
				result.NotificationFailedCount++
			}
		}
		events, err := client.OrderFeeReviewEvent.Query().Where(eventent.OrderIDEQ(o.ID)).Order(eventent.ByCreatedAt(), eventent.ByID()).All(ctx)
		if err != nil {
			return nil, err
		}
		for _, e := range events {
			var saved biz.FeeReviewSnapshot
			if err := json.Unmarshal(e.Snapshot, &saved); err != nil {
				return nil, err
			}
			result.History = append(result.History, &biz.FeeReviewEvent{ID: e.ID, Action: string(e.Action), ActorID: e.ActorID, ActorName: e.ActorName, Comment: e.Comment, CreatedAt: e.CreatedAt, ContentVersion: e.ContentVersion, SubmissionID: e.SubmissionID, Summary: saved.Summary})
		}
	}
	return result, nil
}

func bumpFeeReviewVersion(ctx context.Context, client *ent.Client, orderID uuid.UUID) error {
	return client.Order.UpdateOneID(orderID).AddFeeReviewVersion(1).Exec(ctx)
}

func lockBillFeeOrders(ctx context.Context, client *ent.Client, organizationIDs, feeIDs []uuid.UUID, validate bool) error {
	fees, err := client.OrderFee.Query().Where(feeent.IDIn(feeIDs...), feeent.HasOrderWith(orderent.OrganizationIDIn(organizationIDs...))).All(ctx)
	if err != nil {
		return err
	}
	if len(fees) != len(feeIDs) {
		return biz.ErrFinanceBillFeeInvalid
	}
	seen := map[uuid.UUID]bool{}
	ids := []uuid.UUID{}
	for _, fee := range fees {
		if !seen[fee.OrderID] {
			seen[fee.OrderID] = true
			ids = append(ids, fee.OrderID)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	orders, err := client.Order.Query().Where(orderent.IDIn(ids...), orderent.OrganizationIDIn(organizationIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	if len(orders) != len(ids) {
		return biz.ErrFinanceBillFeeInvalid
	}
	latest, err := client.OrderFee.Query().Where(feeent.IDIn(feeIDs...)).All(ctx)
	if err != nil {
		return err
	}
	if len(latest) != len(fees) {
		return biz.ErrFeeReviewConflict
	}
	old := map[uuid.UUID]uuid.UUID{}
	for _, fee := range fees {
		old[fee.ID] = fee.OrderID
	}
	for _, fee := range latest {
		if old[fee.ID] != fee.OrderID {
			return biz.ErrFeeReviewConflict
		}
	}
	if validate {
		pending, err := client.OrderFeeCorrectionRequest.Query().Where(correctionent.SourceFeeIDIn(feeIDs...), correctionent.StatusEQ(correctionent.StatusPENDING)).Exist(ctx)
		if err != nil {
			return err
		}
		if pending {
			return biz.ErrFeeCorrectionPending
		}
	}

	return nil
}

// LoadOrderFeeReviewSnapshot 供正式种子在同一事务中固化明确的演示审核事实。
func LoadOrderFeeReviewSnapshot(ctx context.Context, client *ent.Client, orderID uuid.UUID) (*biz.FeeReviewSnapshot, string, error) {
	return loadFeeReviewSnapshot(ctx, client, orderID)
}
func orderHasPendingSupplementPredicate() predicate.Order {
	return orderent.Or(orderent.HasFeeSupplementRequestsWith(supplementent.StatusEQ(supplementent.StatusPENDING)), orderent.HasFeeCorrectionRequestsWith(correctionent.StatusEQ(correctionent.StatusPENDING)))
}

// attachOrderFeeReviewProjections 仅批量读取当前页费用与事件；无费用读权限时不投影金额。
func attachOrderFeeReviewProjections(ctx context.Context, client *ent.Client, items []*biz.Order) error {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil
	}
	ids := make([]uuid.UUID, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ID)
	}
	if len(ids) == 0 {
		return nil
	}
	orders, err := client.Order.Query().Where(orderent.IDIn(ids...)).All(ctx)
	if err != nil {
		return err
	}
	orderMap := map[uuid.UUID]*ent.Order{}
	eventsIDs := []uuid.UUID{}
	readable := map[uuid.UUID]bool{}
	permissionCache := map[string]bool{}
	for _, o := range orders {
		orderMap[o.ID] = o
		if o.CurrentFeeReviewEventID != nil {
			eventsIDs = append(eventsIDs, *o.CurrentFeeReviewEventID)
		}
		for _, key := range []string{access.FinanceFeeRead, access.OrderPermission(access.OrderBusinessType(o.BusinessType), access.OrderLock), access.OrderPermission(access.OrderBusinessType(o.BusinessType), access.OrderFeeRead)} {
			cacheKey := o.OrganizationID.String() + ":" + key
			yes, cached := permissionCache[cacheKey]
			if !cached {
				yes, err = hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, key, false)
				if err != nil {
					return err
				}
				permissionCache[cacheKey] = yes
			}
			if yes {
				readable[o.ID] = true
			}
		}
	}
	permitted := []uuid.UUID{}
	for _, o := range orders {
		if readable[o.ID] {
			permitted = append(permitted, o.ID)
		}
	}
	fees, err := client.OrderFee.Query().Where(feeent.OrderIDIn(permitted...)).WithSettlementParty().WithFinanceBillLines(effectiveBillLineFilter).Order(feeent.ByID()).All(ctx)
	if err != nil {
		return err
	}
	grouped := map[uuid.UUID][]*biz.OrderFee{}
	for _, fee := range fees {
		converted, err := orderFeeToBiz(fee)
		if err != nil {
			return err
		}
		grouped[fee.OrderID] = append(grouped[fee.OrderID], converted)
	}
	pending, err := client.OrderFeeSupplementRequest.Query().Where(supplementent.OrderIDIn(ids...), supplementent.StatusEQ(supplementent.StatusPENDING)).All(ctx)
	if err != nil {
		return err
	}
	pendingSet := map[uuid.UUID]bool{}
	for _, request := range pending {
		pendingSet[request.OrderID] = true
	}
	corrections, err := client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrderIDIn(ids...), correctionent.StatusEQ(correctionent.StatusPENDING)).All(ctx)
	if err != nil {
		return err
	}
	for _, request := range corrections {
		pendingSet[request.OrderID] = true
	}
	events, err := client.OrderFeeReviewEvent.Query().Where(eventent.IDIn(eventsIDs...)).All(ctx)
	if err != nil {
		return err
	}
	eventMap := map[uuid.UUID]*ent.OrderFeeReviewEvent{}
	for _, e := range events {
		eventMap[e.ID] = e
	}
	for _, item := range items {
		o := orderMap[item.ID]
		if o == nil {
			continue
		}
		r := item.FeeReview
		if r == nil {
			continue
		}
		r.HasPendingChange = pendingSet[o.ID]
		for _, c := range corrections {
			if c.OrderID == o.ID {
				r.HasPendingCorrection = true
				break
			}
		}
		r.HasPendingSupplement = r.HasPendingChange && !r.HasPendingCorrection
		r.Status = projectedFeeReviewStatus(o)
		r.CanApprove = false
		if readable[o.ID] {
			snapshot, fingerprint, err := biz.BuildFeeReviewSnapshot(grouped[o.ID])
			if err != nil {
				return err
			}
			r.Summary = &snapshot.Summary
			r.PreviewToken = feeReviewPreviewToken(o, fingerprint)
			if o.CurrentFeeReviewEventID != nil {
				e := eventMap[*o.CurrentFeeReviewEventID]
				if e != nil && e.Action == eventent.ActionSUBMIT {
					r.SubmissionID = &e.ID
					r.Summary.SubmittedByName = e.ActorName
					r.Summary.SubmittedAt = &e.CreatedAt
				}
			}
		}
	}
	return nil
}

func lockExistingBillOrders(ctx context.Context, client *ent.Client, organizationIDs, billIDs []uuid.UUID) error {
	lines, err := client.FinanceBillLine.Query().Where(lineent.BillIDIn(billIDs...), lineent.ActiveEQ(true)).All(ctx)
	if err != nil {
		return err
	}
	ids := []uuid.UUID{}
	seen := map[uuid.UUID]bool{}
	for _, line := range lines {
		if !seen[line.OrderID] {
			ids = append(ids, line.OrderID)
			seen[line.OrderID] = true
		}
	}
	if len(ids) == 0 {
		return nil
	}
	orders, err := client.Order.Query().Where(orderent.IDIn(ids...), orderent.OrganizationIDIn(organizationIDs...)).Order(orderent.ByID()).ForUpdate().All(ctx)
	if err != nil {
		return err
	}
	if len(orders) != len(ids) {
		return biz.ErrFinanceBillNotFound
	}
	return nil
}

func qualifiedOrderReviewCandidates(ctx context.Context, client *ent.Client, org, orderID uuid.UUID) ([]biz.OrderFeeSupplementApprover, error) {
	o, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(org)).Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	ancestors, err := companyGrantMembershipIDs(ctx, client, org)
	if err != nil {
		return nil, err
	}
	grants, err := client.RoleAssignment.Query().Where(qualifiedBusinessLockGrantPredicate(access.OrderPermission(access.OrderBusinessType(o.BusinessType), access.OrderLock), org, ancestors, membershipent.HasUserWith(userent.EnabledEQ(true), userent.IsBootstrapAdminEQ(false)), membershipent.HasOrganizationWith(organizationent.EnabledEQ(true)))).WithMembership(func(q *ent.MembershipQuery) { q.WithUser() }).Order(assignmentent.ByID()).All(ctx)
	if err != nil {
		return nil, err
	}
	out := []biz.OrderFeeSupplementApprover{}
	seen := map[uuid.UUID]bool{}
	for _, g := range grants {
		u := g.Edges.Membership.Edges.User
		if seen[u.ID] {
			continue
		}
		seen[u.ID] = true
		out = append(out, biz.OrderFeeSupplementApprover{UserID: u.ID, DisplayName: u.DisplayName, DingTalkUserID: optionalStringValue(u.DingtalkUserid)})
	}
	return out, nil
}

// attachFeeCapabilities 按费用粒度投影操作能力；补审只占用它引用的原费用。
func attachFeeCapabilities(ctx context.Context, client *ent.Client, o *ent.Order, fees []*biz.OrderFee) error {
	p, err := biz.RequirePrincipal(ctx)
	if err != nil {
		return nil
	}
	editable, err := hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, access.OrderPermission(access.OrderBusinessType(o.BusinessType), access.OrderFeeUpdate), false)
	if err != nil {
		return err
	}
	canDelete, err := hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, access.OrderPermission(access.OrderBusinessType(o.BusinessType), access.OrderFeeDelete), false)
	if err != nil {
		return err
	}
	billable, err := hasRealtimeFeePermission(ctx, client, o.OrganizationID, p.UserID, access.FinanceBillCreate, false)
	if err != nil {
		return err
	}
	requests, err := client.OrderFeeCorrectionRequest.Query().Where(correctionent.OrderIDEQ(o.ID), correctionent.StatusEQ(correctionent.StatusPENDING)).All(ctx)
	if err != nil {
		return err
	}
	pending := map[uuid.UUID]bool{}
	for _, r := range requests {
		if r.SourceFeeID != nil {
			pending[*r.SourceFeeID] = true
		}
	}
	for _, f := range fees {
		f.HasPendingChange = pending[f.ID]
		f.CanCorrect = (editable || canDelete) && o.LockedAt != nil && o.TerminationStatus == orderent.TerminationStatusACTIVE && !f.HasActiveBill && !f.HasPendingChange && biz.IsCurrentFee(f)
		f.CanDelete = canDelete && o.LockedAt == nil && o.TerminationStatus == orderent.TerminationStatusACTIVE && !f.HasActiveBill && biz.IsCurrentFee(f)
		f.CanEdit = editable && o.LockedAt == nil && o.TerminationStatus == orderent.TerminationStatusACTIVE && !f.HasActiveBill && biz.IsCurrentFee(f)
		f.CanBill = billable && biz.RequiresAccounting(f) && !f.HasActiveBill && !f.HasPendingChange
		if o.LockedAt != nil {
			f.BlockedReasons = append(f.BlockedReasons, "订单已审核锁定，费用变更需补审")
		}
		if f.HasActiveBill {
			f.BlockedReasons = append(f.BlockedReasons, "费用已建立账单，请先逐步回退至账单未建立")
		}
		if f.HasPendingChange {
			f.BlockedReasons = append(f.BlockedReasons, "原费用存在待补审申请，暂不能建账")
		}
	}
	return nil
}

func pendingCorrectionSourceFees(ctx context.Context, client *ent.Client, ids []uuid.UUID) (map[uuid.UUID]bool, error) {
	result := map[uuid.UUID]bool{}
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := client.OrderFeeCorrectionRequest.Query().Where(correctionent.SourceFeeIDIn(ids...), correctionent.StatusEQ(correctionent.StatusPENDING)).All(ctx)
	if err != nil {
		return nil, err
	}
	for _, r := range rows {
		if r.SourceFeeID != nil {
			result[*r.SourceFeeID] = true
		}
	}
	return result, nil
}
