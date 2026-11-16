package data

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
)

// 在正文保存已预锁旧成员、等待主单锁时提交加票，确定性覆盖成员集合竞态。
func TestSeaMasterContentRejectsConcurrentMemberPostgres(t *testing.T) {
	f := newSeaDocumentChangeFixture(t)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	before := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
	audits := f.data.db.AuditLog.Query().CountX(ctx)
	result := make(chan error, 1)
	err := f.data.WithTx(ctx, func(tx *ent.Tx) error {
		member, err := tx.Order.Create().SetOrganizationID(f.orgID).SetBusinessType("SE").SetTradeDirection("export").SetCustomerID(f.partnerID).SetOrderNo("RACE-" + uuid.NewString()).SetTradeTerm("FOB").SetPaymentTerm("PREPAID").SetIdempotencyKey(uuid.NewString()).Save(ctx)
		if err != nil {
			return err
		}
		if _, err = tx.SeaMasterBill.Query().Where(masterent.IDEQ(f.mblID)).ForUpdate().Only(ctx); err != nil {
			return err
		}
		go func() {
			_, updateErr := NewSeaDocumentRepo(f.data).UpdateSeaMasterBillContent(ctx, f.orgID, f.actorID, f.orderID, before.Version, &biz.SeaBillContent{ShipperText: ptr("竞态正文")}, &biz.AuditEvent{OrganizationID: &f.orgID, UserID: &f.actorID, Result: "success"})
			result <- updateErr
		}()
		if err = waitSeaMasterContentLock(ctx, f.data, result); err != nil {
			return err
		}
		link, err := tx.SeaMasterBillOrderLink.Get(ctx, f.linkID)
		if err != nil {
			return err
		}
		_, err = tx.SeaMasterBillOrderLink.Create().SetOrganizationID(f.orgID).SetOrderID(member.ID).SetMasterBillID(f.mblID).SetTransportExecutionID(link.TransportExecutionID).SetDocumentStructure(link.DocumentStructure).Save(ctx)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	select {
	case err = <-result:
		if !errors.Is(err, biz.ErrSeaDocumentStructureConflict) {
			t.Fatalf("并发加票应拒绝正文保存: %v", err)
		}
	case <-ctx.Done():
		t.Fatal(ctx.Err())
	}
	after := f.data.db.SeaMasterBill.GetX(ctx, f.mblID)
	if after.Version != before.Version || after.ShipperText == nil || *after.ShipperText != *before.ShipperText {
		t.Fatal("竞态失败后主单正文或版本被写入")
	}
	if f.data.db.AuditLog.Query().CountX(ctx) != audits {
		t.Fatal("竞态失败后留下成功审计")
	}
}

func waitSeaMasterContentLock(ctx context.Context, data *Data, result <-chan error) error {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var waiting bool
		err := data.sqlDB.QueryRowContext(ctx, `SELECT EXISTS (
SELECT 1 FROM pg_locks l JOIN pg_stat_activity a ON a.pid=l.pid
WHERE l.relation='sea_master_bills'::regclass AND a.wait_event_type='Lock'
AND a.pid<>pg_backend_pid() AND a.query LIKE '%"sea_master_bills"%')`).Scan(&waiting)
		if err != nil {
			return err
		}
		if waiting {
			return nil
		}
		select {
		case err := <-result:
			return fmt.Errorf("持主单锁期间保存提前结束: %v", err)
		case <-ticker.C:
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
