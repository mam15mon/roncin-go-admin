package data

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	lockent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlockrecord"
)

// seedManualOrderReviewForTest 为隔离数据库夹具建立当前整票批准证据。
// 测试中直接修改费用后必须显式重建批准，不能依靠运行期兼容或回退。
func seedManualOrderReviewForTest(t *testing.T, d *Data, orderID, actorID uuid.UUID) {
	t.Helper()
	err := d.WithinTransaction(context.Background(), func(ctx context.Context) error {
		client, err := d.client(ctx)
		if err != nil {
			return err
		}
		o, err := client.Order.Query().Where(orderent.IDEQ(orderID)).ForUpdate().Only(ctx)
		if err != nil {
			return err
		}
		snapshot, fingerprint, err := loadFeeReviewSnapshot(ctx, client, orderID)
		if err != nil {
			return err
		}
		payload, err := json.Marshal(snapshot)
		if err != nil {
			return err
		}
		now := time.Now().UTC()
		if o.LockedAt != nil {
			_, err := client.OrderLockRecord.Update().Where(lockent.OrderIDEQ(orderID), lockent.UnlockedAtIsNil()).SetUnlockedAt(now).SetUnlockedBy(actorID).SetOrderVersionAtUnlock(o.Version).SetUnlockReason("测试重审").SetUnlockMode(lockent.UnlockModeROLE_DIRECT).Save(ctx)
			if err != nil {
				return err
			}
		}
		_, err = client.OrderLockRecord.Create().SetOrganizationID(o.OrganizationID).SetOrderID(o.ID).SetOrderNo(o.OrderNo).SetBusinessType(lockent.BusinessType(o.BusinessType)).SetGeneration(o.LockGeneration + 1).SetLockedBy(actorID).SetLockedAt(now).SetOrderVersionAtLock(o.Version + 1).SetFeeSnapshot(payload).SetFeeContentVersion(o.FeeContentVersion).SetFeeContentFingerprint(fingerprint).SetReviewComment("集成夹具人工审核").SetIdempotencyKey(uuid.NewString()).SetRequestFingerprint(fingerprint).Save(ctx)
		if err != nil {
			return err
		}
		return client.Order.UpdateOneID(orderID).SetLockedAt(now).SetLockedBy(actorID).SetLockSource(orderent.LockSourceMANUAL).SetLockGeneration(o.LockGeneration + 1).AddVersion(1).Exec(ctx)

	})
	if err != nil {
		t.Fatalf("建立整票费用批准夹具: %v", err)
	}
}

// markCommissionPaidForTest 通过真实预览及净额发放契约登记付款。
func markCommissionPaidForTest(ctx context.Context, usecase *biz.CommissionUsecase, orgID, actorID, commissionID uuid.UUID, version uint64) (*biz.FinanceCommission, error) {
	preview, err := usecase.PreviewPayment(ctx, orgID, commissionID, version)
	if err != nil {
		return nil, err
	}
	return usecase.MarkPaid(ctx, orgID, actorID, commissionID, version, biz.CommissionPaymentInput{PreviewToken: preview.PreviewToken, ActualAmount: preview.ActualAmount, IdempotencyKey: uuid.NewString()})
}
