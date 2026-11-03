package data

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/objectstoragedeletion"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachment"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent/orderattachmentasset"
)

func TestOrderAttachmentRepositoryPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	client, err := d.client(ctx)
	if err != nil {
		t.Fatal(err)
	}
	org, err := client.Organization.Create().SetCode("ATTACHMENT-TEST").SetName("附件测试公司").SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	actor, err := client.User.Create().SetDisplayName("附件测试员").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	partner, err := client.Partner.Create().SetOrganizationID(org.ID).SetCode("CUSTOMER").SetLegalName("测试客户").SetNormalizedName("测试客户").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	makeOrder := func(number string) *ent.Order {
		row, err := client.Order.Create().SetOrganizationID(org.ID).SetCustomerID(partner.ID).SetOrderNo(number).SetIdempotencyKey(uuid.NewString()).SetBusinessType(order.BusinessTypeSE).SetTradeDirection(order.TradeDirectionExport).SetPaymentTerm(order.PaymentTermPREPAID).SetFlowStatus(order.FlowStatusDRAFT).SetVersion(1).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		return row
	}
	first, second := makeOrder("ATTACHMENT-1"), makeOrder("ATTACHMENT-2")
	repo := NewOrderAttachmentRepo(d)
	input := &biz.OrderAttachment{DocType: "BL", IdempotencyKey: uuid.NewString(), FileName: "提单.pdf", MIMEType: "application/pdf", FileSize: 2048, ObjectKey: "orders/" + org.ID.String() + "/" + first.ID.String() + "/" + uuid.NewString(), Checksum: strings.Repeat("a", 64)}
	audit := func() *biz.AuditEvent {
		return &biz.AuditEvent{OrganizationID: &org.ID, UserID: &actor.ID, Action: "order.attachment.register", Result: "success", Details: map[string]string{}}
	}
	created, err := repo.Create(ctx, org.ID, actor.ID, first.ID, input, audit())
	if err != nil {
		t.Fatal(err)
	}
	retried, err := repo.Create(ctx, org.ID, actor.ID, first.ID, input, audit())
	if err != nil || retried.ID != created.ID {
		t.Fatal("重试未复用既有引用", err)
	}
	inputConflict := *input
	inputConflict.FileName = "不同.pdf"
	if _, err := repo.Create(ctx, org.ID, actor.ID, first.ID, &inputConflict, audit()); err != biz.ErrOrderAttachmentExists {
		t.Fatal("同幂等键冲突未拒绝", err)
	}
	if count, err := client.OrderAttachment.Query().Where(orderattachment.OrderIDEQ(first.ID)).Count(ctx); err != nil || count != 1 {
		t.Fatal("创建了重复引用", err)
	}
	if _, err := repo.Get(ctx, uuid.New(), first.ID, created.ID); err != biz.ErrOrderAttachmentNotFound {
		t.Fatal("跨组织读取未拒绝", err)
	}
	if _, err := repo.Get(ctx, org.ID, second.ID, created.ID); err != biz.ErrOrderAttachmentNotFound {
		t.Fatal("跨订单读取未拒绝", err)
	}
	if err := repo.CheckEditable(ctx, org.ID, first.ID); err != nil {
		t.Fatal(err)
	}
	if err := client.Order.UpdateOneID(first.ID).SetLockedAt(time.Now()).SetLockedBy(actor.ID).SetLockSource(order.LockSourceMANUAL).SetLockGeneration(1).Exec(ctx); err != nil {
		t.Fatal(err)
	}
	if err := repo.CheckEditable(ctx, org.ID, first.ID); err == nil {
		t.Fatal("准备上传未感知业务锁")
	}
	newInput := *input
	newInput.IdempotencyKey = uuid.NewString()
	newInput.ObjectKey = "orders/" + org.ID.String() + "/" + first.ID.String() + "/" + uuid.NewString()
	if _, err := repo.Create(ctx, org.ID, actor.ID, first.ID, &newInput, audit()); err == nil {
		t.Fatal("准备后锁定订单仍允许登记")
	}
	if err := repo.RemoveReference(ctx, org.ID, actor.ID, first.ID, created.ID, audit()); err == nil {
		t.Fatal("锁定订单仍允许解除附件")
	}
	if err := client.Order.UpdateOneID(first.ID).ClearLockedAt().ClearLockedBy().ClearLockSource().Exec(ctx); err != nil {
		t.Fatal(err)
	}
	shared, err := client.OrderAttachment.Create().SetOrderID(second.ID).SetAssetID(created.AssetID).SetDocType("BL").SetIdempotencyKey(uuid.NewString()).SetCreatedBy(actor.ID).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.RemoveReference(ctx, org.ID, actor.ID, first.ID, created.ID, audit()); err != nil {
		t.Fatal(err)
	}
	if exists, err := client.OrderAttachmentAsset.Query().Where(orderattachmentasset.IDEQ(created.AssetID)).Exist(ctx); err != nil || !exists {
		t.Fatal("共享资产被过早删除", err)
	}
	if count, err := client.ObjectStorageDeletion.Query().Where(objectstoragedeletion.ObjectKeyEQ(input.ObjectKey)).Count(ctx); err != nil || count != 0 {
		t.Fatal("仍有引用却登记删除任务", err)
	}
	if _, err := repo.Get(ctx, org.ID, second.ID, shared.ID); err != nil {
		t.Fatal("原票解除引用影响共享附件读取", err)
	}
	// 原票重放登记与另一票解除最后引用锁不同订单。保持删除事务未提交，
	// 让重放请求等待旧资产锁，再提交删除，证明墓碑检查不能放在资产查询前。
	removed, removalDone, registerDone := make(chan error, 1), make(chan error, 1), make(chan error, 1)
	release := make(chan struct{})
	var releaseOnce sync.Once
	unblock := func() { releaseOnce.Do(func() { close(release) }) }
	t.Cleanup(unblock)
	go func() {
		removalDone <- d.WithinTransaction(ctx, func(txCtx context.Context) error {
			err := repo.RemoveReference(txCtx, org.ID, actor.ID, second.ID, shared.ID, audit())
			removed <- err
			if err == nil {
				<-release
			}
			return err
		})
	}()
	if err := <-removed; err != nil {
		t.Fatal(err)
	}
	go func() {
		_, err := repo.Create(ctx, org.ID, actor.ID, first.ID, input, audit())
		registerDone <- err
	}()
	waitCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()
	for {
		var blocked bool
		// 只观察当前隔离测试 Schema 的实际锁等待，避免以固定延时猜测并发顺序。
		err := d.sqlDB.QueryRowContext(waitCtx, `SELECT EXISTS (
			SELECT 1 FROM pg_stat_activity WHERE datname = current_database()
			AND wait_event_type = 'Lock' AND query LIKE '%order_attachment_assets%'
			AND pid IN (SELECT pid FROM pg_locks WHERE relation = $1::regclass)
		)`, "order_attachment_assets").Scan(&blocked)
		if err != nil {
			unblock()
			t.Fatal(fmt.Errorf("等待资产行锁失败: %w", err))
		}
		if blocked {
			break
		}
		select {
		case <-waitCtx.Done():
			unblock()
			t.Fatal("重放登记未进入资产锁等待")
		case <-ticker.C:
		}
	}
	unblock()
	if err := <-removalDone; err != nil {
		t.Fatal(err)
	}
	if err := <-registerDone; err != biz.ErrOrderAttachmentUploadInvalid {
		t.Fatal("删除事务提交后重放登记使资产复活", err)
	}
	if exists, err := client.OrderAttachmentAsset.Query().Where(orderattachmentasset.IDEQ(created.AssetID)).Exist(ctx); err != nil || exists {
		t.Fatal("最后引用解除后资产未删除", err)
	}
	if count, err := client.ObjectStorageDeletion.Query().Where(objectstoragedeletion.ObjectKeyEQ(input.ObjectKey)).Count(ctx); err != nil || count != 1 {
		t.Fatal("最后引用解除未原子登记唯一删除任务", err)
	}
	if _, err := repo.Create(ctx, org.ID, actor.ID, first.ID, input, audit()); err != biz.ErrOrderAttachmentUploadInvalid {
		t.Fatal("删除任务登记后旧授权仍可重新登记", err)
	}
}
