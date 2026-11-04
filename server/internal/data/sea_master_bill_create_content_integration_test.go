package data

import (
	"context"
	"reflect"
	"testing"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderlockhousebillsnapshotent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockhousebillsnapshot"
	orderlockrecordent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillversion"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	seatransportexecutionversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecutionversion"
)

func TestSeaMasterBillCreatePreservesSharedContentPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	fixture := newOrderPostgresFixture(t, data)
	// 本用例生成真实审核快照，先清理其引用，再由订单夹具清理业务行。
	t.Cleanup(func() {
		steps := []func() (int, error){
			func() (int, error) {
				return data.db.OrderLockHouseBillSnapshot.Delete().Where(orderlockhousebillsnapshotent.OrganizationIDEQ(fixture.organizationID)).Exec(ctx)
			},
			func() (int, error) {
				return data.db.OrderLockRecord.Delete().Where(orderlockrecordent.OrganizationIDEQ(fixture.organizationID)).Exec(ctx)
			},
			func() (int, error) {
				return data.db.SeaHouseBill.Update().Where(seahousebillent.OrganizationIDEQ(fixture.organizationID)).ClearCurrentVersionID().Save(ctx)
			},
			func() (int, error) {
				return data.db.SeaMasterBill.Update().Where(seamasterbillent.OrganizationIDEQ(fixture.organizationID)).ClearCurrentVersionID().Save(ctx)
			},
			func() (int, error) {
				return data.db.SeaTransportExecution.Update().Where(seatransportexecutionent.OrganizationIDEQ(fixture.organizationID)).ClearCurrentVersionID().Save(ctx)
			},
			func() (int, error) {
				return data.db.SeaHouseBillVersion.Delete().Where(seahousebillversionent.OrganizationIDEQ(fixture.organizationID)).Exec(ctx)
			},
			func() (int, error) {
				return data.db.SeaMasterBillVersion.Delete().Where(seamasterbillversionent.OrganizationIDEQ(fixture.organizationID)).Exec(ctx)
			},
			func() (int, error) {
				return data.db.SeaTransportExecutionVersion.Delete().Where(seatransportexecutionversionent.OrganizationIDEQ(fixture.organizationID)).Exec(ctx)
			},
		}
		for _, step := range steps {
			if _, err := step(); err != nil {
				t.Errorf("清理审核快照失败: %v", err)
				return
			}
		}
	})
	uc := fixture.newUsecase()
	input := fixture.validInput()
	shipper, consignee, description := "原始发货人", "原始收货人", "原始货物描述"
	packages, weight, volume := int32(12), 340.5, 6.25
	input.SeaDocumentInput.MasterBillContent = &biz.SeaBillContent{
		ShipperText: &shipper, ConsigneeText: &consignee, GoodsDescriptionText: &description,
		PackageCount: &packages, GrossWeightKg: &weight, VolumeCbm: &volume,
	}
	first, err := uc.Create(ctx, fixture.organizationID, fixture.actorID, input)
	if err != nil {
		t.Fatal(err)
	}
	masterID := first.SeaMasterBill.MasterBillID
	master := data.db.SeaMasterBill.GetX(ctx, masterID)
	if master.ShipperText == nil || *master.ShipperText != shipper || master.PackageCount == nil || *master.PackageCount != int(packages) {
		t.Fatal("独立新建母单没有保存正文")
	}
	caller := &biz.Principal{UserID: fixture.actorID, IsBootstrapAdmin: true, Organization: biz.Organization{ID: fixture.organizationID}}
	if _, err := NewOrderLockRepo(data, nil).LockOrder(ctx, caller, first.ID, first.Version, uuid.NewString(), lockTestReviewInput(t, data, ctx, first.ID), testGateAuditEvent(fixture, "order.lock")); err != nil {
		t.Fatalf("审核锁定原票失败: %v", err)
	}
	before := data.db.SeaMasterBill.GetX(ctx, masterID)
	lockedOrder := data.db.Order.GetX(ctx, first.ID)
	version := data.db.SeaMasterBillVersion.GetX(ctx, *before.CurrentVersionID)

	if _, err := createOrderConfirmCandidate(ctx, fixture, uc, before.MasterNo, masterID, before.Version, nil); err != nil {
		t.Fatalf("不改正文的加票应成功: %v", err)
	}
	after := data.db.SeaMasterBill.GetX(ctx, masterID)
	if computeMBLContentHash(after) != computeMBLContentHash(before) || after.Version != before.Version || !after.UpdatedAt.Equal(before.UpdatedAt) || !reflect.DeepEqual(after.CurrentVersionID, before.CurrentVersionID) {
		t.Fatal("加票改动了共享母单正文或版本")
	}
	if current := data.db.Order.GetX(ctx, first.ID); current.Version != lockedOrder.Version || !reflect.DeepEqual(current.LockedAt, lockedOrder.LockedAt) {
		t.Fatal("加票改动了原票审核状态")
	}
	if current := data.db.SeaMasterBillVersion.GetX(ctx, version.ID); current.ContentHash != version.ContentHash || !reflect.DeepEqual(current.ShipperText, version.ShipperText) {
		t.Fatal("加票改动了审核时的母单快照")
	}

	for _, label := range []string{"空白正文", "有值正文"} {
		t.Run(label, func(t *testing.T) {
			document := fixture.validInput().SeaDocumentInput
			document.MasterBillContent = &biz.SeaBillContent{}
			if label == "有值正文" {
				changed := "不应覆盖原母单"
				document.MasterBillContent.ShipperText = &changed
			}
			_, err := createOrderConfirmCandidate(ctx, fixture, uc, before.MasterNo, masterID, before.Version, document)
			if err == nil || kratoserrors.FromError(err).Reason != "SEA_DOCUMENT_INVALID_ARGUMENT" {
				t.Fatalf("夹带正文应明确拒绝，实际: %v", err)
			}
			fixture.requireCommittedOrders(2, 2)
			current := data.db.SeaMasterBill.GetX(ctx, masterID)
			if computeMBLContentHash(current) != computeMBLContentHash(before) || current.Version != before.Version {
				t.Fatal("拒绝加票后仍发生了母单写入")
			}
		})
	}

	t.Run("已作废候选拒绝加票", func(t *testing.T) {
		data.db.SeaMasterBill.UpdateOneID(masterID).SetStatus(seamasterbillent.StatusVOIDED).ExecX(ctx)
		_, err := createOrderConfirmCandidate(ctx, fixture, uc, before.MasterNo, masterID, before.Version, nil)
		if err == nil || kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentVoided.Reason {
			t.Fatalf("作废候选应拒绝，实际: %v", err)
		}
		fixture.requireCommittedOrders(2, 2)
	})
}
