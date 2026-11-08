package data

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	memberent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	lifecycle "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	permissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/permission"
	roleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/role"
	assignmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/roleassignment"
	houseent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	masterent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func restoreFixture(t *testing.T, structure biz.SeaDocumentStructure) (*orderPostgresFixture, *biz.Order, *ent.SeaMasterBillOrderLink) {
	t.Helper()
	d, close := getIntegrationData(t)
	t.Cleanup(close)
	f := newOrderPostgresFixture(t, d)
	in := f.validInput()
	in.SeaDocumentInput.DocumentStructure = &structure
	if structure == biz.SeaDocumentStructureDirect {
		in.SeaDocumentInput.HouseBill = nil
	}
	order, err := f.newUsecase().Create(context.Background(), f.organizationID, f.actorID, in)
	if err != nil {
		t.Fatal(err)
	}
	mustCompleteTermination(t, f.newUsecase(), f, order)
	order, err = NewOrderRepo(d).Get(context.Background(), f.organizationID, order.ID)
	if err != nil {
		t.Fatal(err)
	}
	source := d.db.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(order.ID)).OnlyX(context.Background())
	return f, order, source
}

func TestSeaOrderRestorePostgres(t *testing.T) {
	for _, structure := range []biz.SeaDocumentStructure{biz.SeaDocumentStructureDirect, biz.SeaDocumentStructureHouse} {
		t.Run(string(structure), func(t *testing.T) {
			f, order, source := restoreFixture(t, structure)
			ctx := context.Background()
			masterBefore := f.data.db.SeaMasterBill.GetX(ctx, source.MasterBillID)
			transportBefore := f.data.db.SeaTransportExecution.GetX(ctx, source.TransportExecutionID)
			restored, err := f.newUsecase().TransitionTermination(ctx, f.organizationID, f.actorID, order.ID, order.Version, biz.OrderTerminationActive, nil, "恢复原运输安排")
			if err != nil {
				t.Fatal(err)
			}
			if restored.TerminationStatus != biz.OrderTerminationActive || restored.Version != order.Version+1 || restored.SeaMasterBill == nil || restored.SeaMasterBill.MasterBillID != source.MasterBillID || restored.SeaMasterBill.TransportExecutionID != source.TransportExecutionID {
				t.Fatalf("恢复状态与关联不完整: %+v", restored)
			}
			if f.data.db.SeaMasterBill.GetX(ctx, source.MasterBillID).Version != masterBefore.Version || f.data.db.SeaTransportExecution.GetX(ctx, source.TransportExecutionID).Version != transportBefore.Version {
				t.Fatal("恢复不可改写主单或航程版本")
			}
			active := f.data.db.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(order.ID), linkent.StatusEQ(linkent.StatusACTIVE)).OnlyX(ctx)
			old := f.data.db.SeaMasterBillOrderLink.GetX(ctx, source.ID)
			event := f.data.db.OrderLifecycleEvent.Query().Where(lifecycle.OrderIDEQ(order.ID), lifecycle.FromStatusEQ("TERMINATED"), lifecycle.ToStatusEQ("ACTIVE")).OnlyX(ctx)
			if event.ReferenceID == nil || *event.ReferenceID != active.ID {
				t.Fatal("恢复事件应引用新活动关系")
			}
			audit := f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID), auditent.ActionEQ("order.termination.transition")).Order(ent.Desc(auditent.FieldCreatedAt)).FirstX(ctx)
			var details map[string]string
			if err := json.Unmarshal(audit.Details, &details); err != nil {
				t.Fatal(err)
			}
			if details["ended_link_id"] != source.ID.String() || details["new_link_id"] != active.ID.String() {
				t.Fatalf("恢复审计缺少关系证据: %+v", audit.Details)
			}

			if active.ID == source.ID || active.Version != 1 || active.DocumentStructure != source.DocumentStructure || old.Status != linkent.StatusENDED || old.Version != source.Version || !old.EndedAt.Equal(*source.EndedAt) {
				t.Fatal("历史行被改写或新关系错误")
			}
			for range 2 {
				docs, err := NewSeaDocumentRepo(f.data).GetSeaOrderDocuments(ctx, f.organizationID, order.ID)
				if err != nil || docs.MasterBill == nil || docs.MasterBill.ID != source.MasterBillID {
					t.Fatalf("恢复后重新打开单证失败: %v", err)
				}
			}
			if _, _, err := NewSeaDocumentChangeRepo(f.data).ListMasterBillVersions(ctx, f.organizationID, order.ID, 1, 20); err != nil {
				t.Fatal(err)
			}
			if structure == biz.SeaDocumentStructureHouse {
				docs, err := NewSeaDocumentRepo(f.data).GetSeaOrderDocuments(ctx, f.organizationID, order.ID)
				if err != nil || docs.HouseBill == nil {
					t.Fatal("恢复丢失原分单")
				}
			}
			if _, err = f.newUsecase().TransitionTermination(ctx, f.organizationID, f.actorID, order.ID, order.Version, biz.OrderTerminationActive, nil, "旧版本重复恢复"); err == nil {
				t.Fatal("旧版本恢复应拒绝")
			}
		})
	}
}

func TestSeaOrderRestoreBlockedPostgres(t *testing.T) {
	cases := []string{"主单作废", "分单作废", "航程变化", "本票审核锁", "共享成员审核锁", "缺少本次退关关系", "本次退关关系歧义", "直单主单被占用", "审计故障"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			structure := biz.SeaDocumentStructureHouse
			if name == "直单主单被占用" {
				structure = biz.SeaDocumentStructureDirect
			}
			f, order, source := restoreFixture(t, structure)
			ctx := context.Background()
			beforeEvents := f.data.db.OrderLifecycleEvent.Query().Where(lifecycle.OrderIDEQ(order.ID)).CountX(ctx)
			switch name {
			case "主单作废":
				f.data.db.SeaMasterBill.UpdateOneID(source.MasterBillID).SetStatus(masterent.StatusVOIDED).SaveX(ctx)
			case "分单作废":
				f.data.db.SeaHouseBill.Update().Where(houseent.OrderIDEQ(order.ID)).SetStatus(houseent.StatusVOIDED).SaveX(ctx)
			case "航程变化":
				f.data.db.SeaTransportExecution.UpdateOneID(source.TransportExecutionID).SetVesselName("其他船舶").SaveX(ctx)
			case "本票审核锁":
				mustLockOrderDirectly(t, f.data, f, order.ID)
			case "共享成员审核锁":
				other := f.data.db.Order.Create().SetIdempotencyKey(uuid.NewString()).SetOrganizationID(f.organizationID).SetCustomerID(f.partnerID).SetOrderNo("OTHER" + f.suffix).SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).SetTradeTerm(orderent.TradeTermFOB).SetPaymentTerm(orderent.PaymentTermPREPAID).SaveX(ctx)
				mustLockOrderDirectly(t, f.data, f, other.ID)
				f.data.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(other.ID).SetMasterBillID(source.MasterBillID).SetTransportExecutionID(source.TransportExecutionID).SetDocumentStructure(linkent.DocumentStructureHOUSE).SaveX(ctx)
			case "直单主单被占用":
				other := f.data.db.Order.Create().SetIdempotencyKey(uuid.NewString()).SetOrganizationID(f.organizationID).SetCustomerID(f.partnerID).SetOrderNo("MEMBER" + f.suffix).SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).SetTradeTerm(orderent.TradeTermFOB).SetPaymentTerm(orderent.PaymentTermPREPAID).SaveX(ctx)
				f.data.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(other.ID).SetMasterBillID(source.MasterBillID).SetTransportExecutionID(source.TransportExecutionID).SetDocumentStructure(linkent.DocumentStructureHOUSE).SaveX(ctx)
			case "缺少本次退关关系":
				source.Update().SetEndedReason("旧改配").SaveX(ctx)
			case "本次退关关系歧义":
				f.data.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(order.ID).SetMasterBillID(source.MasterBillID).SetTransportExecutionID(source.TransportExecutionID).SetDocumentStructure(source.DocumentStructure).SetStatus(linkent.StatusENDED).SetEndedAt(*source.EndedAt).SetEndedReason(orderTerminationLinkEndedReason).SaveX(ctx)
			case "审计故障":
				f.data.db.AuditLog.Use(func(next ent.Mutator) ent.Mutator {
					return ent.MutateFunc(func(ctx context.Context, m ent.Mutation) (ent.Value, error) {
						if m.Op().Is(ent.OpCreate) {
							return nil, fmt.Errorf("注入审计失败")
						}
						return next.Mutate(ctx, m)
					})
				})
			}
			baseline := f.data.db.Order.GetX(ctx, order.ID)
			if _, err := f.newUsecase().TransitionTermination(ctx, f.organizationID, f.actorID, order.ID, baseline.Version, biz.OrderTerminationActive, nil, "不得部分恢复"); err == nil {
				t.Fatal("不满足恢复条件应拒绝")
			}
			current := f.data.db.Order.GetX(ctx, order.ID)
			if current.TerminationStatus != orderent.TerminationStatusTERMINATED || current.Version != baseline.Version || f.data.db.OrderLifecycleEvent.Query().Where(lifecycle.OrderIDEQ(order.ID)).CountX(ctx) != beforeEvents {
				t.Fatal("失败产生部分订单状态/事件")
			}
			if count := f.data.db.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(order.ID), linkent.StatusEQ(linkent.StatusACTIVE)).CountX(ctx); count != 0 {
				t.Fatal("失败留下新活动关系")
			}
		})
	}
}

func TestSeaOrderRestoreConcurrentPostgres(t *testing.T) {
	f, order, _ := restoreFixture(t, biz.SeaDocumentStructureDirect)
	var wg sync.WaitGroup
	results := make(chan error, 2)
	start := make(chan struct{})
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, err := f.newUsecase().TransitionTermination(context.Background(), f.organizationID, f.actorID, order.ID, order.Version, biz.OrderTerminationActive, nil, "并发恢复")
			results <- err
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatalf("并发恢复成功%d次", success)
	}
	if count := f.data.db.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(order.ID), linkent.StatusEQ(linkent.StatusACTIVE)).CountX(context.Background()); count != 1 {
		t.Fatalf("活动关系%d条", count)
	}
}

func grantRestoreRepair(t *testing.T, f *orderPostgresFixture) {
	t.Helper()
	ctx := context.Background()
	role := f.data.db.Role.Create().SetOrganizationID(f.organizationID).SetCode("repair-" + f.suffix).SetName("显式补链测试授权").SaveX(ctx)
	for _, key := range []string{"business.order.se.read", "business.order.se.transition"} {
		p, err := f.data.db.Permission.Query().Where(permissionent.KeyEQ(key)).Only(ctx)
		if ent.IsNotFound(err) {
			p, err = f.data.db.Permission.Create().SetKey(key).SetName(key).SetGroup("order").Save(ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
		f.data.db.Role.UpdateOne(role).AddPermissions(p).SaveX(ctx)
	}
	m := f.data.db.Membership.Query().Where(memberent.UserIDEQ(f.actorID), memberent.OrganizationIDEQ(f.organizationID)).OnlyX(ctx)
	f.data.db.RoleAssignment.Create().SetMembershipID(m.ID).SetRoleID(role.ID).SaveX(ctx)
	t.Cleanup(func() {
		f.data.db.RoleAssignment.Delete().Where(assignmentent.RoleIDEQ(role.ID)).ExecX(ctx)
		f.data.db.Role.Delete().Where(roleent.IDEQ(role.ID)).ExecX(ctx)
	})
}

func TestSeaOrderRestoreRepairPostgres(t *testing.T) {
	f, order, source := restoreFixture(t, biz.SeaDocumentStructureHouse)
	ctx := context.Background()
	// 构造旧流程真实产生的状态：清退关状态但未创建活动关系，事件保留原始事实。
	at := time.Now().UTC()
	f.data.db.Order.UpdateOneID(order.ID).SetTerminationStatus(orderent.TerminationStatusACTIVE).ClearTerminationType().ClearTerminationReason().ClearTerminatedAt().ClearTerminatedBy().SetVersion(order.Version + 1).SaveX(ctx)
	f.data.db.OrderLifecycleEvent.Create().SetOrderID(order.ID).SetDimension(lifecycle.DimensionTERMINATION).SetFromStatus("TERMINATED").SetToStatus("ACTIVE").SetAction("transition").SetOperatorID(f.actorID).SetChangedAt(at).SaveX(ctx)
	input := SeaOrderRestoreRepairInput{OrganizationID: f.organizationID, OrderID: order.ID, ExpectedVersion: order.Version + 1, EndedLinkID: source.ID, ActorID: f.actorID, Reason: "修复已授权验收订单"}
	if _, err := f.data.RepairSeaOrderRestoreLink(ctx, input, false); err == nil {
		t.Fatal("无真实授权不得预检")
	}
	grantRestoreRepair(t, f)
	grant := f.data.db.Role.Query().Where(roleent.OrganizationIDEQ(f.organizationID), roleent.CodeEQ("repair-"+f.suffix)).OnlyX(ctx)
	grant.Update().SetEnabled(false).SaveX(ctx)
	if _, err := f.data.RepairSeaOrderRestoreLink(ctx, input, false); err == nil {
		t.Fatal("角色撤权后必须即时拒绝")
	}
	grant.Update().SetEnabled(true).SaveX(ctx)
	duplicate := f.data.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(order.ID).SetMasterBillID(source.MasterBillID).SetTransportExecutionID(source.TransportExecutionID).SetDocumentStructure(source.DocumentStructure).SetStatus(linkent.StatusENDED).SetEndedReason(orderTerminationLinkEndedReason).SetEndedAt(*source.EndedAt).SaveX(ctx)
	if _, err := f.data.RepairSeaOrderRestoreLink(ctx, input, false); err == nil {
		t.Fatal("相同时间两条退关关系不得任选")
	}
	f.data.db.SeaMasterBillOrderLink.DeleteOne(duplicate).ExecX(ctx)
	later := f.data.db.OrderLifecycleEvent.Create().SetOrderID(order.ID).SetDimension(lifecycle.DimensionTERMINATION).SetFromStatus("ACTIVE").SetToStatus("TERMINATING").SetAction("transition").SetChangedAt(at.Add(time.Second)).SaveX(ctx)
	if _, err := f.data.RepairSeaOrderRestoreLink(ctx, input, false); err == nil {
		t.Fatal("最新状态链已变化不得补链")
	}
	f.data.db.OrderLifecycleEvent.DeleteOne(later).ExecX(ctx)
	beforeAudit := f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID)).CountX(ctx)
	preview, err := f.data.RepairSeaOrderRestoreLink(ctx, input, false)
	if err != nil || preview.Applied || preview.NewLinkID != nil {
		t.Fatalf("预检失败: %+v %v", preview, err)
	}
	if f.data.db.Order.GetX(ctx, order.ID).Version != input.ExpectedVersion || f.data.db.AuditLog.Query().Where(auditent.OrganizationIDEQ(f.organizationID)).CountX(ctx) != beforeAudit || f.data.db.SeaMasterBillOrderLink.Query().Where(linkent.OrderIDEQ(order.ID)).CountX(ctx) != 1 {
		t.Fatal("默认预检产生业务写入")
	}
	bad := input
	bad.ExpectedVersion++
	if _, err := f.data.RepairSeaOrderRestoreLink(ctx, bad, true); err == nil {
		t.Fatal("期望版本不符必须拒绝")
	}
	source.Update().SetEndedAt(source.EndedAt.Add(-time.Second)).SaveX(ctx)
	if _, err := f.data.RepairSeaOrderRestoreLink(ctx, input, true); err == nil {
		t.Fatal("事件和关系不匹配必须拒绝")
	}
	source.Update().SetEndedAt(*source.EndedAt).SaveX(ctx)
	applied, err := f.data.RepairSeaOrderRestoreLink(ctx, input, true)
	if err != nil || !applied.Applied || applied.NewLinkID == nil {
		t.Fatalf("补链失败: %+v %v", applied, err)
	}
	if f.data.db.Order.GetX(ctx, order.ID).Version != input.ExpectedVersion+1 {
		t.Fatal("补链版本未递增")
	}
	if n := f.data.db.OrderLifecycleEvent.Query().Where(lifecycle.OrderIDEQ(order.ID), lifecycle.DimensionEQ(lifecycle.DimensionTERMINATION)).CountX(ctx); n != 3 {
		t.Fatal("补链不得伪造状态转移")
	}
	if _, err := NewSeaDocumentRepo(f.data).GetSeaOrderDocuments(ctx, f.organizationID, order.ID); err != nil {
		t.Fatal(err)
	}
	if n := f.data.db.AuditLog.Query().Where(auditent.ActionEQ("order.sea_restore_link.repair"), auditent.ResourceIDEQ(order.ID.String())).CountX(ctx); n != 1 {
		t.Fatal("补链审计不完整")
	}
	if _, err := f.data.RepairSeaOrderRestoreLink(ctx, input, true); err == nil {
		t.Fatal("补链不可重复创建")
	}
}

func TestSeaOrderRestoreClosedUnlockedMemberPostgres(t *testing.T) {
	f, order, source := restoreFixture(t, biz.SeaDocumentStructureHouse)
	ctx := context.Background()
	other := f.data.db.Order.Create().SetIdempotencyKey(uuid.NewString()).SetOrganizationID(f.organizationID).SetCustomerID(f.partnerID).SetOrderNo("CLOSED" + f.suffix).SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).SetTradeTerm(orderent.TradeTermFOB).SetPaymentTerm(orderent.PaymentTermPREPAID).SetFlowStatus(orderent.FlowStatusDOCUMENT_RELEASED).SetClosureStatus(orderent.ClosureStatusCLOSED).SetClosureReason("正常完结").SetClosedAt(time.Now()).SetClosedBy(f.actorID).SaveX(ctx)
	f.data.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(other.ID).SetMasterBillID(source.MasterBillID).SetTransportExecutionID(source.TransportExecutionID).SetDocumentStructure(linkent.DocumentStructureHOUSE).SaveX(ctx)
	if _, err := f.newUsecase().TransitionTermination(ctx, f.organizationID, f.actorID, order.ID, order.Version, biz.OrderTerminationActive, nil, "同批已完结但未锁可恢复"); err != nil {
		t.Fatal(err)
	}
}

func TestSeaOrderRestoreActionProjection(t *testing.T) {
	now := time.Now()
	for _, state := range []biz.OrderTerminationStatus{biz.OrderTerminationTerminated, biz.OrderTerminationTerminating} {
		order := &biz.Order{BusinessType: biz.OrderBusinessSE, TerminationStatus: state, ClosureStatus: biz.OrderClosureOpen, LockedAt: &now}
		canRestore := false
		for _, action := range orderAllowedActions(order) {
			if action == biz.OrderActionCancelTermination {
				canRestore = true
			}
		}
		if canRestore != (state == biz.OrderTerminationTerminating) {
			t.Fatalf("锁定状态动作错误: %s %v", state, canRestore)
		}
	}
}
