package data

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seasharedcontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainer"
)

// 覆盖从另一张未锁订单进入共享箱时的跨票门禁，以及解除门禁后的正常路径。
func TestSeaSharedContainerMemberBusinessGatePostgres(t *testing.T) {
	f := newSharedContainerFixture(t)
	ctx := context.Background()
	container := f.createContainer(t, "GATE")
	var err error
	container, err = f.uc.SaveDraft(ctx, f.orgID, f.userID, f.order1.ID, container.ID, container.Version, f.fullAllocationInputs())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := f.data.db.Order.UpdateOneID(f.order1.ID).SetLockedAt(time.Now()).SetLockedBy(f.userID).SetLockGeneration(1).Save(ctx); err != nil {
		t.Fatal(err)
	}
	changed := *container
	changed.ContainerNo += "CHANGED"
	seal := "NEW-SEAL"
	changed.SealNo = &seal
	if _, err := f.uc.Update(ctx, f.orgID, f.userID, f.order2.ID, container.ID, container.Version, &changed); kratoserrors.Reason(err) != "ORDER_BUSINESS_LOCKED" {
		t.Fatalf("从未锁成员修改共享箱必须因另一成员的审核锁而拒绝: %v", err)
	}
	stored, err := f.data.db.SeaSharedContainer.Get(ctx, container.ID)
	if err != nil || stored.ContainerNo != container.ContainerNo || stored.Version != container.Version || stored.SealNo != nil {
		t.Fatalf("被拒绝的箱号/封号修改不应留下写入: %+v, %v", stored, err)
	}
	if _, err := f.data.db.Order.UpdateOneID(f.order1.ID).ClearLockedAt().ClearLockedBy().Save(ctx); err != nil {
		t.Fatal(err)
	}
	container, err = f.uc.Confirm(ctx, f.orgID, f.userID, f.order2.ID, container.ID, container.Version, nil)
	if err != nil {
		t.Fatalf("全员可编辑时应允许确认共享箱: %v", err)
	}
	if _, err := f.data.db.Order.UpdateOneID(f.order1.ID).SetLockedAt(time.Now()).SetLockedBy(f.userID).Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Withdraw(ctx, f.orgID, f.userID, f.order2.ID, container.ID, container.Version); kratoserrors.Reason(err) != "ORDER_BUSINESS_LOCKED" {
		t.Fatalf("从未锁成员撤销共享箱确认必须被拦截: %v", err)
	}
	stored, err = f.data.db.SeaSharedContainer.Get(ctx, container.ID)
	if err != nil || stored.Status != seasharedcontainerent.StatusCONFIRMED || stored.Version != container.Version || stored.ConfirmedAt == nil {
		t.Fatalf("被拒绝的撤销不能改变确认事实: %+v, %v", stored, err)
	}
	if _, err := f.data.db.Order.UpdateOneID(f.order1.ID).ClearLockedAt().ClearLockedBy().SetTerminationStatus(orderent.TerminationStatusTERMINATING).SetTerminationType(orderent.TerminationTypeCARRIER_CANCEL).SetTerminationReason("船司取消").Save(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := f.uc.Withdraw(ctx, f.orgID, f.userID, f.order2.ID, container.ID, container.Version); kratoserrors.Reason(err) != "ORDER_TERMINATION_IN_PROGRESS" {
		t.Fatalf("成员终止中也不允许撤销共享箱确认: %v", err)
	}
	if _, err := f.data.db.Order.UpdateOneID(f.order1.ID).SetTerminationStatus(orderent.TerminationStatusACTIVE).ClearTerminationType().ClearTerminationReason().Save(ctx); err != nil {
		t.Fatal(err)
	}
	container, err = f.uc.Withdraw(ctx, f.orgID, f.userID, f.order2.ID, container.ID, container.Version)
	if err != nil {
		t.Fatalf("解除成员门禁后应允许撤销: %v", err)
	}
	changed.Version = container.Version
	if _, err := f.uc.Update(ctx, f.orgID, f.userID, f.order2.ID, container.ID, container.Version-1, &changed); kratoserrors.Reason(err) != "SEA_SHARED_CONTAINER_CONFLICT" {
		t.Fatalf("成员可编辑也不能绕过箱版本校验: %v", err)
	}
	updated, err := f.uc.Update(ctx, f.orgID, f.userID, f.order2.ID, container.ID, container.Version, &changed)
	if err != nil || updated.ContainerNo != changed.ContainerNo || updated.SealNo == nil || *updated.SealNo != seal {
		t.Fatalf("解除成员门禁后应允许修改箱号与封号: %+v, %v", updated, err)
	}
}

func TestSeaReassignmentSharedAllocationAndSameMasterPostgres(t *testing.T) {
	env := newSplitTestEnv(t)
	ctx := context.Background()
	f := createTestSplitFixture(t, env, "CONSISTENCY", splitFixtureOptions{withSharedBox: true})
	master, err := env.data.db.SeaMasterBill.Get(ctx, env.mblID)
	if err != nil {
		t.Fatal(err)
	}
	masterNo := strings.ToUpper(master.MasterNo)
	if _, err := master.Update().SetMasterNo(masterNo).SetNormalizedMasterNo(masterNo).Save(ctx); err != nil {
		t.Fatal(err)
	}
	input := &biz.SeaOrderReassignmentInput{
		OrderID: f.order.ID, ExpectedOrderVersion: f.order.Version, ExpectedLinkVersion: f.link.Version,
		IdempotencyKey: "same-master-" + uuid.NewString(), RequestFingerprint: "same-master-intent",
		Reason: "船公司确认换航次", ResponsibilityType: biz.ResponsibilityTypeCarrier,
		Confirmation: &biz.SeaExternalConfirmation{ConfirmedByParty: "船公司", ConfirmedAt: time.Now().UTC(), ConfirmationNote: "确认保留母单号换航次"},
		Target:       &biz.SeaOrderReassignmentTargetInput{TargetType: biz.SplitTargetTypeNew, MasterNo: strings.ToLower(masterNo), ShippingLineID: &env.carrierID, VesselName: "NEW VESSEL", VoyageNo: "002W"},
	}
	repo := NewSeaOrderChangeRepo(env.data)
	for _, status := range []seasharedcontainerent.Status{seasharedcontainerent.StatusDRAFT, seasharedcontainerent.StatusCONFIRMED} {
		if _, err := env.data.db.SeaSharedContainer.UpdateOneID(f.sharedContainerID).SetStatus(status).Save(ctx); err != nil {
			t.Fatal(err)
		}
		actions, err := env.uc.GetChangeActions(ctx, env.orgID, f.order.ID)
		if err != nil || actions.CanReassign || !strings.Contains(strings.Join(actions.ReassignBlockedReasons, " "), "共享箱分配") {
			t.Fatalf("%s 共享箱分配应关闭改配动作: %+v, %v", status, actions, err)
		}
		preview, err := env.uc.PreviewReassignment(ctx, env.orgID, input)
		if err != nil || preview.IsValid || !strings.Contains(strings.Join(preview.Errors, " "), "移除本票分配") {
			t.Fatalf("%s 共享箱分配应使预览不可执行: %+v, %v", status, preview, err)
		}
		if _, err := repo.ExecuteReassignment(ctx, env.orgID, env.userID, input, &biz.AuditEvent{}); kratoserrors.Reason(err) != "SEA_ORDER_REASSIGNMENT_BLOCKED" {
			t.Fatalf("%s 分配必须在执行锁内独立拒绝: %v", status, err)
		}
		order, _ := env.data.db.Order.Get(ctx, f.order.ID)
		link, _ := env.data.db.SeaMasterBillOrderLink.Get(ctx, f.link.ID)
		house, _ := env.data.db.SeaHouseBill.Get(ctx, f.hbl.ID)
		if order.Version != f.order.Version || link.Status != seamasterbillorderlinkent.StatusACTIVE || house.Version != f.hbl.Version {
			t.Fatalf("拒绝改配不能修改订单/关联/HBL: %+v %+v %+v", order, link, house)
		}
		eventCount, err := env.data.db.SeaOrderReassignmentEvent.Query().Count(ctx)
		if err != nil || eventCount != 0 {
			t.Fatalf("拒绝改配不得留下事件: %d, %v", eventCount, err)
		}
	}
	if _, err := env.data.db.Order.UpdateOneID(f.order.ID).SetLockedAt(time.Now()).SetLockedBy(env.userID).Save(ctx); err != nil {
		t.Fatal(err)
	}
	preview, err := env.uc.PreviewReassignment(ctx, env.orgID, input)
	if err != nil || kratoserrors.Reason(preview.GateBlockedError) != "ORDER_BUSINESS_LOCKED" {
		t.Fatalf("已有审核锁的优先级应高于共享箱提示: %+v, %v", preview, err)
	}
	if _, err := env.data.db.Order.UpdateOneID(f.order.ID).ClearLockedAt().ClearLockedBy().Save(ctx); err != nil {
		t.Fatal(err)
	}
	if err := env.data.db.SeaSharedContainerAllocation.DeleteOneID(f.sharedAllocationID).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	preview, err = env.uc.PreviewReassignment(ctx, env.orgID, input)
	if err != nil || !preview.IsValid || preview.TargetMasterBill.MasterBillID != env.mblID {
		t.Fatalf("移除分配后同母单新航次预览应通过: %+v, %v", preview, err)
	}
	// 模拟预览后新增分配：执行不能仅相信预览结果。
	allocation, err := env.data.db.SeaSharedContainerAllocation.Create().SetOrganizationID(env.orgID).SetSharedContainerID(f.sharedContainerID).SetOrderID(f.order.ID).SetHouseBillID(f.hbl.ID).SetCargoItemID(f.cargoItem.ID).SetPackageCount(1).SetGrossWeightKg("1").SetVolumeCbm("1").SetVersion(1).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ExecuteReassignment(ctx, env.orgID, env.userID, input, &biz.AuditEvent{}); kratoserrors.Reason(err) != "SEA_ORDER_REASSIGNMENT_BLOCKED" {
		t.Fatalf("预览后新增分配应在执行锁内阻断: %v", err)
	}
	if err := env.data.db.SeaSharedContainerAllocation.DeleteOne(allocation).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	event, err := env.uc.ExecuteReassignment(ctx, env.orgID, env.userID, input)
	if err != nil {
		t.Fatalf("同母单新航次实际改配应成功: %v", err)
	}
	if event.PreviousMasterBillID != env.mblID || event.TargetMasterBillID != env.mblID {
		t.Fatalf("同母单改配不能创建另一张母单: %+v", event)
	}
	oldLink, _ := env.data.db.SeaMasterBillOrderLink.Get(ctx, f.link.ID)
	newLink, err := env.data.db.SeaMasterBillOrderLink.Get(ctx, event.TargetLinkID)
	if err != nil || oldLink.Status != seamasterbillorderlinkent.StatusENDED || newLink.TransportExecutionID == env.teID || newLink.MasterBillID != env.mblID {
		t.Fatalf("应结束旧关联并建立同母单新航次关联: old=%+v new=%+v %v", oldLink, newLink, err)
	}
	masterCount, err := env.data.db.SeaMasterBill.Query().Count(ctx)
	if err != nil || masterCount != 1 {
		t.Fatalf("同母单改配后母单数量不应增加: %d, %v", masterCount, err)
	}
	replay, err := env.uc.ExecuteReassignment(ctx, env.orgID, env.userID, input)
	if err != nil || replay.ID != event.ID {
		t.Fatalf("改配应支持相同意图幂等重放: %+v, %v", replay, err)
	}
	executionCount, err := env.data.db.SeaTransportExecution.Query().Count(ctx)
	if err != nil || executionCount != 2 {
		t.Fatalf("幂等重放不得重复生成新航次: %d, %v", executionCount, err)
	}
	eventCount, err := env.data.db.SeaOrderReassignmentEvent.Query().Count(ctx)
	if err != nil || eventCount != 1 {
		t.Fatalf("幂等重放不得重复生成改配事件: %d, %v", eventCount, err)
	}
}

// 将成员新增固定在 Update 的预读之后，验证不会在持有箱锁后追加新订单锁。
func TestSeaSharedContainerMemberChangeDuringUpdatePostgres(t *testing.T) {
	f := newSharedContainerFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	container := f.createContainer(t, "MEMBER")
	type mutationProbeKey struct{}
	readDone, proceed := make(chan struct{}), make(chan struct{})
	var once sync.Once
	f.data.db.SeaSharedContainerAllocation.Intercept(ent.InterceptFunc(func(next ent.Querier) ent.Querier {
		return ent.QuerierFunc(func(queryCtx context.Context, query ent.Query) (ent.Value, error) {
			value, err := next.Query(queryCtx, query)
			if queryCtx.Value(mutationProbeKey{}) == true {
				once.Do(func() {
					close(readDone)
					select {
					case <-proceed:
					case <-queryCtx.Done():
					}
				})
			}
			return value, err
		})
	}))
	changed := *container
	changed.ContainerNo += "CHANGED"
	result := make(chan error, 1)
	go func() {
		_, err := f.uc.Update(context.WithValue(ctx, mutationProbeKey{}, true), f.orgID, f.userID, f.order1.ID, container.ID, container.Version, &changed)
		result <- err
	}()
	select {
	case <-readDone:
	case <-ctx.Done():
		t.Fatal("等待共享箱成员预读超时")
	}
	if _, err := f.uc.SaveDraft(ctx, f.orgID, f.userID, f.order1.ID, container.ID, container.Version, f.fullAllocationInputs()); err != nil {
		close(proceed)
		t.Fatalf("预读之后新增成员分配失败: %v", err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Count(ctx)
	if err != nil {
		close(proceed)
		t.Fatal(err)
	}
	close(proceed)
	select {
	case err := <-result:
		if kratoserrors.Reason(err) != "SEA_SHARED_CONTAINER_CONFLICT" {
			t.Fatalf("成员集合变化必须返回冲突，不能后置补锁: %v", err)
		}
	case <-ctx.Done():
		t.Fatal("共享箱成员变化后更新未结束，可能存在锁序死锁")
	}
	stored, err := f.data.db.SeaSharedContainer.Get(ctx, container.ID)
	if err != nil || stored.ContainerNo != container.ContainerNo || stored.Version != container.Version+1 {
		t.Fatalf("仅分配保存可生效，资料更新应整体回滚: %+v, %v", stored, err)
	}
	finalAuditCount, err := f.data.db.AuditLog.Query().Count(ctx)
	if err != nil || finalAuditCount != auditCount {
		t.Fatalf("被拒绝资料更新不得追加审计: before=%d after=%d err=%v", auditCount, finalAuditCount, err)
	}
}
