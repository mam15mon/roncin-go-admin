package data

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	ordermilestoneent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordermilestone"
	orderreleasepodent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderreleasepod"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

// cleanupLifecycleGateSubresources 在共享夹具按组织清理订单前，先删除本测试
// 新增的子资源行（货物、POD、里程碑），避免共享清理删除订单时触发外键约束。
// 共享夹具（orderPostgresFixture）自身只覆盖既有测试用到的表，禁止修改。
func cleanupLifecycleGateSubresources(t *testing.T, data *Data, fixture *orderPostgresFixture) {
	t.Helper()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		steps := []struct {
			name string
			run  func() error
		}{
			{name: "货物明细", run: func() error {
				_, err := data.db.OrderCargoItem.Delete().Where(ordercargoitement.OrganizationIDEQ(fixture.organizationID)).Exec(ctx)
				return err
			}},
			{name: "放货 POD", run: func() error {
				_, err := data.db.OrderReleasePod.Delete().Where(orderreleasepodent.HasOrderWith(orderent.OrganizationIDEQ(fixture.organizationID))).Exec(ctx)
				return err
			}},
			{name: "里程碑", run: func() error {
				_, err := data.db.OrderMilestone.Delete().Where(ordermilestoneent.HasOrderWith(orderent.OrganizationIDEQ(fixture.organizationID))).Exec(ctx)
				return err
			}},
		}
		for _, step := range steps {
			if err := step.run(); err != nil {
				t.Errorf("清理生命周期门禁测试%s失败: %v", step.name, err)
				return
			}
		}
	})
}

// TestOrderLifecycleGatePostgres 在真实 PostgreSQL 隔离 Schema 中验证：
//  1. 统一内容门禁对四类状态（TERMINATING/TERMINATED/CLOSED/已业务锁定）在代表性
//     子资源写入口上的稳定阻断，以及正常态放行；
//  2. 生命周期命令不被内容门禁误杀：锁定后的放单订单可结案、可反结案；
//  3. 退关完成原子结束活动 SE Link，取消退关保留 Link，恢复不复活历史 Link，
//     旧版本请求稳定冲突且零部分提交；
//  4. 自拼汇总只统计 Link ACTIVE 且 Order ACTIVE 的 LCL 成员，HBL 来自当前
//     SeaHouseBill 而不是旧 ShippingDocuments。
func TestOrderLifecycleGatePostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	t.Run("内容门禁四类状态阻断代表性子资源写入", func(t *testing.T) {
		for _, tc := range []struct {
			stateName  string
			wantReason string
		}{
			{stateName: "TERMINATING", wantReason: "ORDER_TERMINATION_IN_PROGRESS"},
			{stateName: "TERMINATED", wantReason: "ORDER_TERMINATED"},
			{stateName: "CLOSED", wantReason: "ORDER_CLOSED"},
			{stateName: "已业务锁定", wantReason: "ORDER_BUSINESS_LOCKED"},
		} {
			t.Run(tc.stateName, func(t *testing.T) {
				fixture := newOrderPostgresFixture(t, data)
				cleanupLifecycleGateSubresources(t, data, fixture)
				ctx := context.Background()
				usecase := fixture.newUsecase()
				created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.validInput())
				if err != nil {
					t.Fatalf("创建订单失败: %v", err)
				}
				version := created.Version
				switch tc.stateName {
				case "TERMINATING":
					version = mustStartTermination(t, usecase, fixture, created)
				case "TERMINATED":
					version = mustCompleteTermination(t, usecase, fixture, created)
				case "CLOSED":
					// 经 DOCUMENT_RELEASED 正常结案，保持终止维度 ACTIVE，验证
					// 门禁返回 ORDER_CLOSED 而不是终止类错误。
					version = mustAdvanceFlowToReleased(t, usecase, fixture, created)
					version = mustCloseOrder(t, usecase, fixture, created.ID, version)
				case "已业务锁定":
					version = mustLockOrderDirectly(t, data, fixture, created.ID)
				}

				// 代表性子资源写入：订单草稿、放货 POD（原重复门禁改造点）、
				// 货物明细、里程碑。
				_, updateErr := usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, version, fixture.validUpdateInput())
				requireOrderGateReason(t, "订单草稿更新", updateErr, tc.wantReason)

				releaseNo := "POD-" + fixture.suffix
				_, podErr := NewOrderReleasePodRepo(data).Add(ctx, fixture.organizationID, created.ID,
					&biz.OrderReleasePod{ID: uuid.Must(uuid.NewV7()), ReleaseNo: &releaseNo},
					testGateAuditEvent(fixture, "order.release_pod.add"))
				requireOrderGateReason(t, "放货 POD 新增", podErr, tc.wantReason)

				_, cargoErr := NewOrderCargoItemRepo(data).Add(ctx, fixture.organizationID, created.ID,
					&biz.OrderCargoItem{CargoName: "测试货物", PackageCount: 10, GrossWeightKg: 100, VolumeCbm: 1, Version: 1},
					testGateAuditEvent(fixture, "order.cargo_item.add"))
				requireOrderGateReason(t, "货物明细新增", cargoErr, tc.wantReason)

				occurredAt := time.Now().UTC()
				_, milestoneErr := NewOrderMilestoneRepo(data).Set(ctx, fixture.organizationID, created.ID,
					"ATD", version, &occurredAt, nil, false, fixture.actorID,
					testGateAuditEvent(fixture, "order.milestone.set"))
				requireOrderGateReason(t, "里程碑设置", milestoneErr, tc.wantReason)

				// 阻断后零写入。
				afterBlock, err := data.db.Order.Query().
					Where(orderent.IDEQ(created.ID)).
					WithReleasePods().
					WithCargoItems().
					Only(ctx)
				if err != nil {
					t.Fatalf("读取阻断后订单失败: %v", err)
				}
				if len(afterBlock.Edges.ReleasePods) != 0 || len(afterBlock.Edges.CargoItems) != 0 {
					t.Fatalf("阻断后仍产生子资源写入: pods=%d cargo=%d", len(afterBlock.Edges.ReleasePods), len(afterBlock.Edges.CargoItems))
				}
			})
		}
	})

	t.Run("正常 ACTIVE+OPEN+未锁定状态子资源写入成功", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		cleanupLifecycleGateSubresources(t, data, fixture)
		ctx := context.Background()
		usecase := fixture.newUsecase()
		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.validInput())
		if err != nil {
			t.Fatalf("创建订单失败: %v", err)
		}
		if _, err := usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version, fixture.validUpdateInput()); err != nil {
			t.Fatalf("正常态订单草稿更新应成功: %v", err)
		}
		releaseNo := "POD-OK-" + fixture.suffix
		if _, err := NewOrderReleasePodRepo(data).Add(ctx, fixture.organizationID, created.ID,
			&biz.OrderReleasePod{ID: uuid.Must(uuid.NewV7()), ReleaseNo: &releaseNo},
			testGateAuditEvent(fixture, "order.release_pod.add")); err != nil {
			t.Fatalf("正常态放货 POD 新增应成功: %v", err)
		}
		if _, err := NewOrderCargoItemRepo(data).Add(ctx, fixture.organizationID, created.ID,
			&biz.OrderCargoItem{CargoName: "测试货物", PackageCount: 1, GrossWeightKg: 1, VolumeCbm: 1, Version: 1},
			testGateAuditEvent(fixture, "order.cargo_item.add")); err != nil {
			t.Fatalf("正常态货物明细新增应成功: %v", err)
		}
	})

	t.Run("锁定后的放单订单可结案且反结案不受历史业务锁阻断", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		cleanupLifecycleGateSubresources(t, data, fixture)
		ctx := context.Background()
		usecase := fixture.newUsecase()
		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.validInput())
		if err != nil {
			t.Fatalf("创建订单失败: %v", err)
		}
		version := mustAdvanceFlowToReleased(t, usecase, fixture, created)
		version = mustLockOrderDirectly(t, data, fixture, created.ID)

		closed, err := usecase.TransitionClosure(ctx, fixture.organizationID, fixture.actorID, created.ID, version, biz.OrderClosureClosed, "放单完成结案")
		if err != nil {
			t.Fatalf("锁定的 DOCUMENT_RELEASED 订单结案应成功: %v", err)
		}
		if closed.ClosureStatus != biz.OrderClosureClosed {
			t.Fatalf("结案后 closure_status = %s，期望 CLOSED", closed.ClosureStatus)
		}
		stored, err := data.db.Order.Get(ctx, created.ID)
		if err != nil {
			t.Fatalf("读取结案后订单失败: %v", err)
		}
		if stored.LockedAt == nil || stored.LockGeneration != 1 {
			t.Fatalf("结案不应清除业务锁状态: %#v", stored)
		}

		reopened, err := usecase.TransitionClosure(ctx, fixture.organizationID, fixture.actorID, created.ID, closed.Version, biz.OrderClosureOpen, "结案错误反结案")
		if err != nil {
			t.Fatalf("带历史业务锁的反结案应成功: %v", err)
		}
		if reopened.ClosureStatus != biz.OrderClosureOpen {
			t.Fatalf("反结案后 closure_status = %s，期望 OPEN", reopened.ClosureStatus)
		}
	})

	t.Run("退关完成原子结束活动 Link 且恢复不复活", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		cleanupLifecycleGateSubresources(t, data, fixture)
		ctx := context.Background()
		usecase := fixture.newUsecase()
		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.validInput())
		if err != nil {
			t.Fatalf("创建订单失败: %v", err)
		}
		linkBefore, err := data.db.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrderIDEQ(created.ID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Only(ctx)
		if err != nil {
			t.Fatalf("读取初始活动 Link 失败: %v", err)
		}

		terminating := mustStartTermination(t, usecase, fixture, created)
		// ACTIVE → TERMINATING 不结束 Link。
		activeCount, err := data.db.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrderIDEQ(created.ID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Count(ctx)
		if err != nil || activeCount != 1 {
			t.Fatalf("进入终止流程后活动 Link 数 = %d，期望 1（error=%v）", activeCount, err)
		}

		// 旧版本请求稳定冲突，且不产生任何部分提交。
		if _, err := usecase.TransitionTermination(ctx, fixture.organizationID, fixture.actorID, created.ID, terminating-1, biz.OrderTerminationTerminated, terminationTypePtr(biz.OrderTerminationCustomsReturn), "旧版本完成退关"); !isOrderStatusConflict(err) {
			t.Fatalf("旧版本完成退关应返回 ORDER_STATUS_CONFLICT，实际: %v", err)
		}
		stuck, err := data.db.Order.Get(ctx, created.ID)
		if err != nil || stuck.TerminationStatus != orderent.TerminationStatusTERMINATING || stuck.Version != terminating {
			t.Fatalf("旧版本冲突后存在部分提交: %#v error=%v", stuck, err)
		}
		if count, err := data.db.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrderIDEQ(created.ID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Count(ctx); err != nil || count != 1 {
			t.Fatalf("旧版本冲突后活动 Link 数 = %d，期望 1（error=%v）", count, err)
		}

		terminated := mustCompleteTerminationFrom(t, usecase, fixture, created.ID, terminating)

		endedLinks, err := data.db.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrderIDEQ(created.ID),
		).All(ctx)
		if err != nil || len(endedLinks) != 1 {
			t.Fatalf("退关完成后 Link 数 = %d，期望 1（error=%v）", len(endedLinks), err)
		}
		ended := endedLinks[0]
		if ended.ID != linkBefore.ID {
			t.Fatalf("结束的 Link 与原活动 Link 不一致")
		}
		if ended.Status != seamasterbillorderlinkent.StatusENDED {
			t.Fatalf("Link status = %s，期望 ENDED", ended.Status)
		}
		if ended.EndedAt == nil || ended.EndedReason == nil || *ended.EndedReason != "订单退关" {
			t.Fatalf("Link 缺少结束事实: ended_at=%v ended_reason=%v", ended.EndedAt, ended.EndedReason)
		}
		if ended.Version != linkBefore.Version+1 {
			t.Fatalf("Link version = %d，期望 %d", ended.Version, linkBefore.Version+1)
		}
		if activeCount, err := data.db.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrderIDEQ(created.ID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Count(ctx); err != nil || activeCount != 0 {
			t.Fatalf("退关完成后活动 Link 数 = %d，期望 0（error=%v）", activeCount, err)
		}
		// 终止完成写入生命周期事件与审计。
		if count, err := data.db.OrderLifecycleEvent.Query().Where(
			orderlifecycleeventent.OrderIDEQ(created.ID),
			orderlifecycleeventent.DimensionEQ(orderlifecycleeventent.DimensionTERMINATION),
		).Count(ctx); err != nil || count != 2 {
			t.Fatalf("终止维度生命周期事件数 = %d，期望 2（error=%v）", count, err)
		}
		if count, err := data.db.AuditLog.Query().Where(
			auditlogent.OrganizationIDEQ(fixture.organizationID),
			auditlogent.ActionEQ("order.termination.transition"),
		).Count(ctx); err != nil || count != 2 {
			t.Fatalf("终止维度审计数 = %d，期望 2（error=%v）", count, err)
		}

		// TERMINATED → ACTIVE 恢复订单但不复活历史 Link。
		recovered, err := usecase.TransitionTermination(ctx, fixture.organizationID, fixture.actorID, created.ID, terminated, biz.OrderTerminationActive, nil, "恢复订单")
		if err != nil {
			t.Fatalf("终止恢复应成功: %v", err)
		}
		if recovered.TerminationStatus != biz.OrderTerminationActive {
			t.Fatalf("恢复后 termination_status = %s，期望 ACTIVE", recovered.TerminationStatus)
		}
		storedLinks, err := data.db.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrderIDEQ(created.ID),
		).All(ctx)
		if err != nil || len(storedLinks) != 1 || storedLinks[0].Status != seamasterbillorderlinkent.StatusENDED {
			t.Fatalf("恢复后历史 Link 应保持 ENDED 不可变: %#v error=%v", storedLinks, err)
		}
	})

	t.Run("退关取消保留活动 Link", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		cleanupLifecycleGateSubresources(t, data, fixture)
		ctx := context.Background()
		usecase := fixture.newUsecase()
		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, fixture.validInput())
		if err != nil {
			t.Fatalf("创建订单失败: %v", err)
		}
		terminating := mustStartTermination(t, usecase, fixture, created)
		cancelled, err := usecase.TransitionTermination(ctx, fixture.organizationID, fixture.actorID, created.ID, terminating, biz.OrderTerminationActive, nil, "取消退关")
		if err != nil {
			t.Fatalf("取消退关应成功: %v", err)
		}
		if cancelled.TerminationStatus != biz.OrderTerminationActive {
			t.Fatalf("取消退关后 termination_status = %s，期望 ACTIVE", cancelled.TerminationStatus)
		}
		link, err := data.db.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrderIDEQ(created.ID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Only(ctx)
		if err != nil {
			t.Fatalf("取消退关后应保留原活动 Link: %v", err)
		}
		if link.EndedAt != nil || link.EndedReason != nil {
			t.Fatalf("取消退关不应写入结束事实: %#v", link)
		}
	})

	t.Run("自拼汇总过滤退关成员并使用当前 HBL", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		cleanupLifecycleGateSubresources(t, data, fixture)
		ctx := context.Background()
		usecase := fixture.newUsecase()
		lcl := biz.OrderShipmentLCL

		first := fixture.validInput()
		first.ShipmentType = &lcl
		first.SeaMasterBillInput.MasterNo = "CONS" + strings.ToUpper(fixture.suffix)
		packages1, gross1, volume1 := 10, 100.0, 1.0
		first.TotalPackages = &packages1
		first.TotalGrossWeightKg = &gross1
		first.TotalVolumeCbm = &volume1
		order1, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, first)
		if err != nil {
			t.Fatalf("创建 LCL 订单 1 失败: %v", err)
		}

		// 第 2 票确认关联同一 MBL。
		link1, err := data.db.SeaMasterBillOrderLink.Query().Where(
			seamasterbillorderlinkent.OrganizationIDEQ(fixture.organizationID),
			seamasterbillorderlinkent.OrderIDEQ(order1.ID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).Only(ctx)
		if err != nil {
			t.Fatalf("读取订单 1 活动 Link 失败: %v", err)
		}
		te1, err := data.db.SeaTransportExecution.Get(ctx, link1.TransportExecutionID)
		if err != nil {
			t.Fatalf("读取订单 1 运输执行失败: %v", err)
		}
		second := fixture.validInput()
		second.ShipmentType = &lcl
		second.SeaMasterBillInput.MasterNo = first.SeaMasterBillInput.MasterNo
		second.SeaMasterBillInput.CandidateID = &order1.SeaMasterBill.MasterBillID
		second.SeaMasterBillInput.ExpectedCandidateVersion = &order1.SeaMasterBill.Version
		second.SeaMasterBillInput.CandidateTEID = &te1.ID
		second.SeaMasterBillInput.ExpectedCandidateTEVersion = &te1.Version
		packages2, gross2, volume2 := 5, 50.0, 0.5
		second.TotalPackages = &packages2
		second.TotalGrossWeightKg = &gross2
		second.TotalVolumeCbm = &volume2
		order2, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, second)
		if err != nil {
			t.Fatalf("创建共享 MBL 的 LCL 订单 2 失败: %v", err)
		}

		// 旧 ShippingDocument 模型中的同号单据不应进入自拼汇总。
		if _, err := data.db.OrderShippingDocument.Create().
			SetOrderID(order1.ID).
			SetHouseNo("LEGACY-" + fixture.suffix).
			SetStatus(ordershippingdocumentent.StatusCONFIRMED).
			Save(ctx); err != nil {
			t.Fatalf("创建旧模型分单记录失败: %v", err)
		}
		hbl1 := mustOrderActiveHouseNo(t, data, order1.ID)
		hbl2 := mustOrderActiveHouseNo(t, data, order2.ID)

		summaries, err := usecase.ListConsolidationSummaries(ctx, fixture.organizationID, order1.ID)
		if err != nil {
			t.Fatalf("查询自拼汇总失败: %v", err)
		}
		if len(summaries) != 1 {
			t.Fatalf("自拼汇总数 = %d，期望 1", len(summaries))
		}
		before := summaries[0]
		if len(before.Members) != 2 {
			t.Fatalf("退关前成员数 = %d，期望 2", len(before.Members))
		}
		if before.Entrusted.Packages != packages1+packages2 || before.Entrusted.GrossWeightKg != gross1+gross2 || before.Entrusted.VolumeCbm != volume1+volume2 {
			t.Fatalf("退关前委托件重尺异常: %#v", before.Entrusted)
		}
		houseNosByOrder := map[uuid.UUID][]string{}
		for _, member := range before.Members {
			houseNosByOrder[member.OrderID] = member.HouseNos
		}
		assertStringSliceEqual(t, houseNosByOrder[order1.ID], []string{hbl1})
		assertStringSliceEqual(t, houseNosByOrder[order2.ID], []string{hbl2})

		// 订单 2 完整退关后不再计入成员、件重尺与 HBL。
		terminating := mustStartTermination(t, usecase, fixture, order2)
		if _, err := usecase.TransitionTermination(ctx, fixture.organizationID, fixture.actorID, order2.ID, terminating, biz.OrderTerminationTerminated, terminationTypePtr(biz.OrderTerminationCustomerCancel), "订单2退关"); err != nil {
			t.Fatalf("订单 2 完成退关失败: %v", err)
		}
		after, err := usecase.ListConsolidationSummaries(ctx, fixture.organizationID, order1.ID)
		if err != nil {
			t.Fatalf("退关后查询自拼汇总失败: %v", err)
		}
		if len(after) != 1 {
			t.Fatalf("退关后自拼汇总数 = %d，期望 1", len(after))
		}
		if len(after[0].Members) != 1 || after[0].Members[0].OrderID != order1.ID {
			t.Fatalf("退关后成员 = %#v，期望仅订单 1", after[0].Members)
		}
		if after[0].Entrusted.Packages != packages1 || after[0].Entrusted.GrossWeightKg != gross1 || after[0].Entrusted.VolumeCbm != volume1 {
			t.Fatalf("退关后委托件重尺异常: %#v", after[0].Entrusted)
		}
		assertStringSliceEqual(t, after[0].Members[0].HouseNos, []string{hbl1})

		// 作废 HBL 后不再返回 house number。
		if _, err := data.db.SeaHouseBill.Update().Where(
			seahousebillent.OrganizationIDEQ(fixture.organizationID),
			seahousebillent.OrderIDEQ(order1.ID),
		).SetStatus(seahousebillent.StatusVOIDED).Save(ctx); err != nil {
			t.Fatalf("作废订单 1 的 HBL 失败: %v", err)
		}
		voided, err := usecase.ListConsolidationSummaries(ctx, fixture.organizationID, order1.ID)
		if err != nil {
			t.Fatalf("HBL 作废后查询自拼汇总失败: %v", err)
		}
		if len(voided) != 1 || len(voided[0].Members) != 1 || len(voided[0].Members[0].HouseNos) != 0 {
			t.Fatalf("作废 HBL 后 house numbers = %#v，期望为空", voided[0].Members[0].HouseNos)
		}
	})
}

// mustAdvanceFlowToReleased 把订单主流程推进到 DOCUMENT_RELEASED 并返回最新版本。
func mustAdvanceFlowToReleased(t *testing.T, usecase *biz.OrderUsecase, fixture *orderPostgresFixture, order *biz.Order) uint64 {
	t.Helper()
	ctx := context.Background()
	version := order.Version
	for _, target := range []biz.OrderFlowStatus{
		biz.OrderFlowBooked,
		biz.OrderFlowSpaceAllocated,
		biz.OrderFlowDocumentCutoff,
		biz.OrderFlowCustomsDeclarationArranged,
		biz.OrderFlowDocumentReleased,
	} {
		updated, err := usecase.TransitionStatus(ctx, fixture.organizationID, fixture.actorID, order.ID, version, target, "主流程推进")
		if err != nil {
			t.Fatalf("主流程推进到 %s 失败: %v", target, err)
		}
		version = updated.Version
	}
	return version
}

// mustStartTermination 发起退关（ACTIVE → TERMINATING）并返回最新版本。
func mustStartTermination(t *testing.T, usecase *biz.OrderUsecase, fixture *orderPostgresFixture, order *biz.Order) uint64 {
	t.Helper()
	updated, err := usecase.TransitionTermination(context.Background(), fixture.organizationID, fixture.actorID, order.ID, order.Version, biz.OrderTerminationTerminating, terminationTypePtr(biz.OrderTerminationCustomerCancel), "测试发起退关")
	if err != nil {
		t.Fatalf("发起退关失败: %v", err)
	}
	return updated.Version
}

// mustCompleteTermination 完成退关（ACTIVE → TERMINATING → TERMINATED）并返回最新版本。
func mustCompleteTermination(t *testing.T, usecase *biz.OrderUsecase, fixture *orderPostgresFixture, order *biz.Order) uint64 {
	t.Helper()
	terminating := mustStartTermination(t, usecase, fixture, order)
	return mustCompleteTerminationFrom(t, usecase, fixture, order.ID, terminating)
}

// mustCompleteTerminationFrom 从 TERMINATING 完成退关并返回最新版本。
func mustCompleteTerminationFrom(t *testing.T, usecase *biz.OrderUsecase, fixture *orderPostgresFixture, orderID uuid.UUID, terminatingVersion uint64) uint64 {
	t.Helper()
	updated, err := usecase.TransitionTermination(context.Background(), fixture.organizationID, fixture.actorID, orderID, terminatingVersion, biz.OrderTerminationTerminated, terminationTypePtr(biz.OrderTerminationCustomsReturn), "测试完成退关")
	if err != nil {
		t.Fatalf("完成退关失败: %v", err)
	}
	return updated.Version
}

// mustCloseOrder 结案并返回最新版本。
func mustCloseOrder(t *testing.T, usecase *biz.OrderUsecase, fixture *orderPostgresFixture, orderID uuid.UUID, version uint64) uint64 {
	t.Helper()
	updated, err := usecase.TransitionClosure(context.Background(), fixture.organizationID, fixture.actorID, orderID, version, biz.OrderClosureClosed, "测试结案")
	if err != nil {
		t.Fatalf("结案失败: %v", err)
	}
	return updated.Version
}

// mustLockOrderDirectly 以数据库事实直接构造业务锁定状态，返回递增后的版本。
// 本测试验证生命周期与内容门禁的判定，不重复覆盖锁单资格链路。
func mustLockOrderDirectly(t *testing.T, data *Data, fixture *orderPostgresFixture, orderID uuid.UUID) uint64 {
	t.Helper()
	stored, err := data.db.Order.Get(context.Background(), orderID)
	if err != nil {
		t.Fatalf("读取待锁定订单失败: %v", err)
	}
	updated, err := data.db.Order.UpdateOne(stored).
		SetLockedAt(time.Now().UTC()).
		SetLockedBy(fixture.actorID).
		SetLockGeneration(1).
		SetVersion(stored.Version + 1).
		Save(context.Background())
	if err != nil {
		t.Fatalf("直接构造业务锁定状态失败: %v", err)
	}
	return updated.Version
}

// mustOrderActiveHouseNo 返回订单当前唯一有效 HBL 号。
func mustOrderActiveHouseNo(t *testing.T, data *Data, orderID uuid.UUID) string {
	t.Helper()
	hb, err := data.db.SeaHouseBill.Query().
		Where(seahousebillent.OrderIDEQ(orderID)).
		Only(context.Background())
	if err != nil {
		t.Fatalf("读取订单 %s 的 HBL 失败: %v", orderID, err)
	}
	return hb.HouseNo
}

func terminationTypePtr(t biz.OrderTerminationType) *biz.OrderTerminationType {
	return &t
}

func testGateAuditEvent(fixture *orderPostgresFixture, action string) *biz.AuditEvent {
	organizationID := fixture.organizationID
	actorID := fixture.actorID
	return &biz.AuditEvent{OrganizationID: &organizationID, UserID: &actorID, Action: action, Result: "success", Details: map[string]string{}}
}

func requireOrderGateReason(t *testing.T, operation string, err error, wantReason string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s 应被阻断，实际成功", operation)
	}
	kErr := errors.FromError(err)
	if kErr == nil || kErr.Reason != wantReason {
		t.Fatalf("%s 错误 = %v，期望 reason %s", operation, err, wantReason)
	}
	if kErr.Code != 409 {
		t.Fatalf("%s HTTP code = %d，期望 409", operation, kErr.Code)
	}
}

func isOrderStatusConflict(err error) bool {
	if err == nil {
		return false
	}
	kErr := errors.FromError(err)
	return kErr != nil && kErr.Reason == "ORDER_STATUS_CONFLICT"
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("切片长度 = %d (%v)，期望 %d (%v)", len(got), got, len(want), want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("第 %d 个元素 = %s，期望 %s（got=%v want=%v）", i, got[i], want[i], got, want)
		}
	}
}
