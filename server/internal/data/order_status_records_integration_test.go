package data

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	eventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
)

func TestOrderStatusRecordsPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newOrderPostgresFixture(t, data)
	ctx := context.Background()
	uc := f.newUsecase()
	created, err := uc.Create(ctx, f.organizationID, f.actorID, f.validInput())
	if err != nil {
		t.Fatal(err)
	}
	updated, err := uc.TransitionStatus(ctx, f.organizationID, f.actorID, created.ID, created.Version, biz.OrderFlowBooked, "订舱确认\n客户已知悉")
	if err != nil {
		t.Fatal(err)
	}
	// 停用员工的历史仍然展示姓名；同一节点重入只展示最近一次真实记录。
	if _, err := data.db.User.UpdateOneID(f.actorID).SetDisplayName("状态测试操作员").SetEnabled(false).Save(ctx); err != nil {
		t.Fatal(err)
	}
	at := time.Now().UTC().Add(time.Minute).Truncate(time.Second)
	for index, reason := range []string{"旧退关原因", "最新退关原因"} {
		_, err := data.db.OrderLifecycleEvent.Create().SetOrderID(created.ID).
			SetDimension(eventent.DimensionTERMINATION).SetToStatus("TERMINATING").
			SetAction("transition").SetReason(reason).SetOperatorID(f.actorID).
			SetChangedAt(at.Add(time.Duration(index) * time.Second)).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
	}
	_, err = data.db.OrderLifecycleEvent.Create().SetOrderID(created.ID).
		SetDimension(eventent.DimensionCLOSURE).SetToStatus("CLOSED").SetAction("transition").
		SetChangedAt(at).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	repo := NewOrderRepo(data)
	scopes := []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{f.organizationID}}}
	t.Run("详情返回每个节点最近一次的操作记录", func(t *testing.T) {
		order, err := repo.Get(ctx, f.organizationID, created.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(order.StatusRecords) != 4 {
			t.Fatalf("应有草稿、订舱、最近退关和结案四条记录，实际: %#v", order.StatusRecords)
		}
		for _, record := range order.StatusRecords {
			switch record.Dimension + ":" + record.Status {
			case "FLOW:BOOKED":
				if record.OperatorName != "状态测试操作员" || record.Reason != "订舱确认\n客户已知悉" || record.ChangedAt.IsZero() {
					t.Fatalf("订舱操作信息丢失: %#v", record)
				}
			case "TERMINATION:TERMINATING":
				if record.Reason != "最新退关原因" || !record.ChangedAt.Equal(at.Add(time.Second)) {
					t.Fatalf("未选中最新事件: %#v", record)
				}
			case "CLOSURE:CLOSED":
				if record.Reason != "" || record.OperatorName != "" {
					t.Fatalf("空操作信息不得伪造: %#v", record)
				}
			}
		}
	})
	anchor, err := repo.FindAuthorized(ctx, created.ID, scopes)
	if err != nil || len(anchor.StatusRecords) != 0 {
		t.Fatalf("鉴权锚点不应加载状态详情: %v, %#v", err, anchor)
	}
	if len(updated.StatusRecords) != 2 {
		t.Fatalf("流转响应应立即包含草稿与订舱记录: %#v", updated.StatusRecords)
	}
	if _, err := repo.Get(ctx, uuid.New(), created.ID); err == nil {
		t.Fatal("不允许跨组织读取状态记录")
	}
	if _, err := repo.FindAuthorized(ctx, created.ID, []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{uuid.New()}}}); err == nil {
		t.Fatal("不允许越过授权范围读取状态记录")
	}
}
