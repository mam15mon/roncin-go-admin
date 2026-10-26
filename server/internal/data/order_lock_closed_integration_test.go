package data

import (
	"encoding/json"
	"testing"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	lockent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

// 完结表示操作流程结束，人工审核锁定独立固化资料与费用证据。
func TestClosedOrderManualReviewPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	newCorrectionTestUsecase(t, f)
	repo := NewOrderLockRepo(f.data, nil)
	for _, businessType := range []orderent.BusinessType{orderent.BusinessTypeSI, orderent.BusinessTypeSE} {
		t.Run(string(businessType), func(t *testing.T) {
			f.setT(t)
			for _, op := range []access.OrderOperation{access.OrderRead, access.OrderUpdate, access.OrderFeeRead, access.OrderLock} {
				key := access.OrderPermission(access.OrderBusinessType(businessType), op)
				permission, err := f.data.db.Permission.Query().Where(permissionent.KeyEQ(key)).Only(f.ctx)
				if ent.IsNotFound(err) {
					permission, err = f.data.db.Permission.Create().SetKey(key).SetName(key).SetGroup("审核验证").Save(f.ctx)
				}
				if err != nil {
					t.Fatal(err)
				}
				f.data.db.Role.UpdateOneID(f.lockRoleID).AddPermissions(permission).SaveX(f.ctx)
			}
			o := f.createOrder("CLOSED-REVIEW-" + uuid.NewString())
			fee := addOrdinaryTestFee(t, f, o, 100)
			if businessType == orderent.BusinessTypeSE {
				o = f.data.db.Order.UpdateOneID(o.ID).SetBusinessType(businessType).SaveX(f.ctx)
				carrier := f.data.db.ShippingLine.Create().SetScacCode(newTestSCAC()).SetNameZh("完结审核测试船司").SetNameEn("Closed review carrier").SetCountryCode("CN").SaveX(f.ctx)
				execution := f.data.db.SeaTransportExecution.Create().SetOrganizationID(f.organizationID).SetShippingLineID(carrier.ID).SaveX(f.ctx)
				masterNo := "CLOSED-" + uuid.NewString()
				master := f.data.db.SeaMasterBill.Create().SetOrganizationID(f.organizationID).SetMasterNo(masterNo).SetNormalizedMasterNo(masterNo).SetShippingLineID(carrier.ID).SaveX(f.ctx)
				f.data.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(o.ID).SetMasterBillID(master.ID).SetTransportExecutionID(execution.ID).SetDocumentStructure(linkent.DocumentStructureDIRECT).SaveX(f.ctx)
			}
			// DRAFT 账单即占用费用，完结及审核均不能释放这条限制。
			bill := createOccupiedTestFeeBill(t, f, o, fee)
			if _, err := NewOrderFeeRepo(f.data).Update(f.ctx, f.organizationID, o.ID, fee.ID, fee, nil, nil); err != biz.ErrOrderFeeBillOccupied {
				t.Fatalf("建账后未审核费用仍不可直接修改: %v", err)
			}
			o = f.data.db.Order.GetX(f.ctx, o.ID)
			closed, err := transitionClosureForTest(f.ctx, f, o, biz.OrderClosureClosed)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := NewFinanceBillRepo(f.data).Cancel(f.ctx, []uuid.UUID{f.organizationID}, bill.ID, f.approverID, bill.Version, "按权限回退至未建账", &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.approverID, Action: "finance.bill.cancel", Result: "success"}); err != nil {
				t.Fatal(err)
			}
			fee, err = NewOrderFeeRepo(f.data).Get(f.ctx, f.organizationID, o.ID, fee.ID)
			if err != nil {
				t.Fatal(err)
			}
			note := "完结后、未审核且未建账仍可核对修改"
			fee.Note = &note
			fee, err = NewOrderFeeRepo(f.data).Update(f.ctx, f.organizationID, o.ID, fee.ID, fee, nil, &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.approverID, Action: "order.fee.update", Result: "success"})
			if err != nil || fee.Note == nil || *fee.Note != note {
				t.Fatalf("已完结未锁费用回退未建账后必须可修改: %+v %v", fee, err)
			}
			createOccupiedTestFeeBill(t, f, o, fee)
			state, err := repo.GetOrderLockState(f.ctx, f.organizationID, o.ID, f.approverPrincipal())
			if err != nil || !state.CanLock || state.IsLocked || len(state.LockBlockedReasons) != 0 {
				t.Fatalf("已完结未审核订单必须可审核: state=%+v err=%v", state, err)
			}
			review := lockTestReviewInput(t, f.data, f.ctx, o.ID)
			if _, err := repo.LockOrder(f.ctx, f.approverPrincipal(), o.ID, closed.Version-1, uuid.NewString(), review, nil); err != biz.ErrOrderStatusConflict {
				t.Fatalf("已完结审核仍须匹配订单版本: %v", err)
			}
			staleReview := review
			staleReview.ExpectedFeeContentVersion++
			if _, err := repo.LockOrder(f.ctx, f.approverPrincipal(), o.ID, closed.Version, uuid.NewString(), staleReview, nil); err != biz.ErrFeeReviewConflict {
				t.Fatalf("已完结审核仍须匹配费用内容版本: %v", err)
			}
			auditBefore := f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID), auditent.ActionEQ("order.lock")).CountX(f.ctx)
			key := uuid.NewString()
			audit := &biz.AuditEvent{OrganizationID: &f.organizationID, UserID: &f.approverID, Action: "order.lock", Result: "success"}
			result, err := repo.LockOrder(f.ctx, f.approverPrincipal(), o.ID, closed.Version, key, review, audit)
			if err != nil {
				t.Fatalf("已完结已建账订单应可审核锁定: %v", err)
			}
			after := f.data.db.Order.GetX(f.ctx, o.ID)
			if after.ClosureStatus != orderent.ClosureStatusCLOSED || after.LockedAt == nil || after.LockedBy == nil || *after.LockedBy != f.approverID || after.Version != closed.Version+1 || after.LockGeneration != 1 {
				t.Fatalf("审核应保留完结状态并递增订单版本及锁代次: %+v", after)
			}
			record := f.data.db.OrderLockRecord.Query().Where(lockent.OrderIDEQ(o.ID)).OnlyX(f.ctx)
			if !json.Valid(record.FeeSnapshot) || record.FeeContentVersion == nil || *record.FeeContentVersion != review.ExpectedFeeContentVersion || record.FeeContentFingerprint == nil || *record.FeeContentFingerprint != review.FeeContentFingerprint {
				t.Fatalf("人工审核必须保存费用不可变证据: %+v", record)
			}
			var snapshot biz.FeeReviewSnapshot
			if err := json.Unmarshal(record.FeeSnapshot, &snapshot); err != nil || len(snapshot.Fees) != 1 || snapshot.Fees[0].ID != fee.ID || !snapshot.Fees[0].TotalAmount.Equal(fee.TotalAmount) {
				t.Fatalf("已建账费用也必须完整纳入人工审核快照: %+v %v", snapshot, err)
			}
			if (record.MasterBillVersionID != nil) != (businessType == orderent.BusinessTypeSE) || (record.TransportExecutionVersionID != nil) != (businessType == orderent.BusinessTypeSE) {
				t.Fatalf("SE应保存单证版本，非SE不得保存海运引用: %+v", record)
			}
			if count := f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID), auditent.ActionEQ("order.lock")).CountX(f.ctx); count != auditBefore+1 {
				t.Fatalf("每次审核应只增加一条审计记录: %d", count)
			}
			replay, err := repo.LockOrder(f.ctx, f.approverPrincipal(), o.ID, closed.Version, key, review, audit)
			if err != nil || replay.LockRecord.ID != result.LockRecord.ID || f.data.db.OrderLockRecord.Query().Where(lockent.OrderIDEQ(o.ID)).CountX(f.ctx) != 1 {
				t.Fatalf("已完结审核幂等重放不得新增记录: %+v %v", replay, err)
			}
			if _, err := NewOrderFeeRepo(f.data).Update(f.ctx, f.organizationID, o.ID, fee.ID, fee, nil, nil); errors.FromError(err).Reason != "ORDER_ALREADY_LOCKED" {
				t.Fatalf("审核锁定后普通修改费用必须被锁阻断: %v", err)
			}
		})
	}
}
