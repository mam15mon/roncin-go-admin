package data

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	linkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func TestSeaDocumentTerminatedHistoryPostgres(t *testing.T) {
	ctx := context.Background()
	f := newSeaDocumentChangeFixture(t)
	history := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(f.data))
	mbl := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
	order := f.data.db.Order.GetX(ctx, f.orderID)
	cmd := &biz.SeaDocumentAmendmentCommand{
		OrderID: f.orderID, DocumentType: biz.SeaDocumentTypeMasterBill, DocumentID: f.mblID,
		ExpectedOrderVersion: order.Version, ExpectedDocumentVersion: mbl.Version, ExpectedCurrentVersionID: *mbl.CurrentVersionID,
		Reason: "退关前更正发货人", IdempotencyKey: "terminated-history-" + uuid.NewString(),
		Input: &biz.SeaDocumentAmendmentInput{MasterBillContent: &biz.SeaBillContent{ShipperText: stringPtr("改单后的发货人")}}, Confirmation: f.confirmation(),
	}
	v2, err := history.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit())
	if err != nil {
		t.Fatal(err)
	}
	active, total, err := history.ListMasterBillVersions(ctx, f.orgID, f.orderID, 1, 200)
	if err != nil || total != 2 || active[0].ID != v2.ID {
		t.Fatalf("活动历史读取失败: total=%d err=%v", total, err)
	}
	fixture := &orderPostgresFixture{data: f.data, organizationID: f.orgID, actorID: f.actorID}
	current, err := NewOrderRepo(f.data).Get(ctx, f.orgID, f.orderID)
	if err != nil {
		t.Fatal(err)
	}
	mustCompleteTermination(t, fixture.newUsecase(), fixture, current)
	ended := f.data.db.SeaMasterBillOrderLink.GetX(ctx, f.linkID)
	// 旧改配主单有独立版本，不能因结束时间较新而进入当前退关主单版本列表。
	oldNo := "OLD" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", ""))
	old := f.data.db.SeaMasterBill.Create().SetOrganizationID(f.orgID).SetShippingLineID(mbl.ShippingLineID).SetMasterNo(oldNo).SetNormalizedMasterNo(oldNo).SaveX(ctx)
	oldLink := f.data.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.orgID).SetOrderID(f.orderID).SetMasterBillID(old.ID).SetTransportExecutionID(ended.TransportExecutionID).SetDocumentStructure(linkent.DocumentStructureDIRECT).SetStatus(linkent.StatusENDED).SetEndedAt(ended.EndedAt.Add(time.Hour)).SetEndedReason("订单改配").SaveX(ctx)
	if err := f.data.WithTx(ctx, func(tx *ent.Tx) error {
		_, err := createMasterVersion(ctx, tx, old, nil, f.actorID, biz.VersionSourceOrderLock, nil, nil, nil, nil)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	for page, want := range []uuid.UUID{v2.ID, f.mblVerID} {
		rows, total, err := history.ListMasterBillVersions(ctx, f.orgID, f.orderID, page+1, 1)
		if err != nil || total != 2 || len(rows) != 1 || rows[0].ID != want || rows[0].MasterBillID != f.mblID {
			t.Fatalf("终态分页误取其他关联: %+v total=%d err=%v", rows, total, err)
		}
		detail, err := history.GetDocumentVersion(ctx, f.orgID, f.orderID, want, biz.SeaDocumentTypeMasterBill)
		if err != nil || detail.Content == nil || detail.Content.ShipperText == nil {
			t.Fatalf("不可变版本详情读取失败: %+v %v", detail, err)
		}
		expected := "改单后的发货人"
		if page == 1 {
			expected = "旧主单发货人"
		}
		if *detail.Content.ShipperText != expected {
			t.Fatalf("不可变内容变化: %s", *detail.Content.ShipperText)
		}
	}
	events, _, err := history.ListDocumentEvents(ctx, f.orgID, f.orderID, 1, 200)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, e := range events {
		if e.ResultVersionID != nil && *e.ResultVersionID == v2.ID && e.EventType == biz.SeaDocumentEventTypeAmendment {
			found = true
		}
	}
	if !found {
		t.Fatal("退关后缺少改单事件")
	}
	if h, total, err := history.ListHouseBillVersions(ctx, f.orgID, f.orderID, f.hblID, 1, 200); err != nil || total != 1 || len(h) != 1 {
		t.Fatalf("分单历史回归: %v", err)
	}
	for _, page := range []int{0, -1} {
		if _, _, err := history.ListMasterBillVersions(ctx, f.orgID, f.orderID, page, 20); err == nil {
			t.Fatal("非法分页应拒绝")
		}
	}
	if _, _, err := history.ListMasterBillVersions(ctx, f.orgID, f.orderID, 1, 201); err == nil {
		t.Fatal("过大分页应拒绝")
	}
	if _, _, err := history.ListMasterBillVersions(ctx, uuid.New(), f.orderID, 1, 20); !errors.Is(err, biz.ErrOrderNotFound) {
		t.Fatalf("跨组织应拒绝: %v", err)
	}
	cmd.IdempotencyKey = uuid.NewString()
	cmd.ExpectedOrderVersion = f.data.db.Order.GetX(ctx, f.orderID).Version
	cmd.ExpectedDocumentVersion = v2.SourceEntityVersion
	cmd.ExpectedCurrentVersionID = v2.ID
	if _, err := history.ExecuteAmendment(ctx, f.orgID, f.actorID, cmd, f.audit()); !errors.Is(err, biz.ErrSeaDocumentNoActiveLink) {
		t.Fatalf("终态写门禁失效: %v", err)
	}
	f.data.db.SeaMasterBillOrderLink.UpdateOneID(f.linkID).SetEndedReason("订单改配").SaveX(ctx)
	if _, _, err := history.ListMasterBillVersions(ctx, f.orgID, f.orderID, 1, 20); !errors.Is(err, biz.ErrSeaDocumentNoActiveLink) {
		t.Fatalf("缺本次关联不得借旧改配: %v", err)
	}
	if f.data.db.SeaMasterBillOrderLink.GetX(ctx, oldLink.ID).Status != linkent.StatusENDED {
		t.Fatal("历史读取复活了关联")
	}
}
