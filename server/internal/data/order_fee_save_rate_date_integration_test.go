package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	"github.com/shopspring/decimal"
)

// saveAfterDateChangeRepo 在用例完成费用准备后、仓储开始拿订单锁前提交改期，
// 确定性重现准备与写入之间的竞争，不依赖 sleep 或调度时机。
type saveAfterDateChangeRepo struct {
	biz.OrderFeeRepo
	beforeSave func() error
}

func (r *saveAfterDateChangeRepo) Save(ctx context.Context, orgID, orderID uuid.UUID, input *biz.OrderFeeSaveInput, audits map[string]*biz.AuditEvent) (*biz.OrderFeeSaveResult, error) {
	if err := r.beforeSave(); err != nil {
		return nil, err
	}
	return r.OrderFeeRepo.Save(ctx, orgID, orderID, input, audits)
}

func (r *saveAfterDateChangeRepo) Add(ctx context.Context, orgID, orderID uuid.UUID, input *biz.OrderFee, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	if err := r.beforeSave(); err != nil {
		return nil, err
	}
	return r.OrderFeeRepo.Add(ctx, orgID, orderID, input, audit)
}

func (r *saveAfterDateChangeRepo) Update(ctx context.Context, orgID, orderID, id uuid.UUID, input *biz.OrderFee, rate *biz.ResolvedRate, audit *biz.AuditEvent) (*biz.OrderFee, error) {
	if err := r.beforeSave(); err != nil {
		return nil, err
	}
	return r.OrderFeeRepo.Update(ctx, orgID, orderID, id, input, rate, audit)
}

func TestOrderFeeSaveDateChangedBeforeLockPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	e := &rateRefreshEnv{t: t, data: f.data, orgID: f.organizationID, actorID: f.approverID}
	custom := biz.NewFinanceCustomSettingUsecase(NewFinanceCustomSettingRepo(f.data))
	credit := biz.NewPartnerCreditUsecase(NewFinanceBillRepo(f.data).(biz.PartnerCreditRepo), custom)
	usecase := func(o *ent.Order) *biz.OrderFeeUsecase {
		r := &saveAfterDateChangeRepo{OrderFeeRepo: NewOrderFeeRepo(f.data), beforeSave: func() error {
			return e.changeDates(o.ID, rateRefreshW2, o.Eta)
		}}
		return biz.NewOrderFeeUsecase(r, biz.NewExchangeRateUsecase(NewExchangeRateRepo(f.data), nil), custom, credit, nil)
	}

	for _, manual := range []bool{false, true} {
		t.Run(map[bool]string{false: "新增系统汇率", true: "新增显式手工汇率"}[manual], func(t *testing.T) {
			o := f.createOrder("SAVE-DATE-" + uuid.NewString())
			if manual {
				addSideFee(t, f, o, biz.OrderFeeReceivable, 100)
			}
			row := saveRow(f, biz.OrderFeePayable, "new", nil, 50)
			if manual {
				one := decimal.NewFromInt(1)
				row.Fee.ExchangeRateOverride = &one
			}
			_, err := usecase(o).Save(f.ctx, f.organizationID, f.approverID, o.ID, biz.OrderFeePayable, uuid.NewString(), []*biz.OrderFeeSaveRowInput{row}, allSavePerms)
			if !errors.Is(err, biz.ErrOrderStatusConflict) {
				t.Fatalf("准备后已改期应拒绝旧日期费用，实际 %v", err)
			}
			fees, err := NewOrderFeeRepo(f.data).List(f.ctx, f.organizationID, o.ID)
			if err != nil {
				t.Fatal(err)
			}
			for _, fee := range fees {
				if fee.Direction == biz.OrderFeePayable {
					t.Fatal("保存失败不得插入旧日期费用")
				}
			}
		})
	}

	t.Run("单行新增接口同样拒绝旧日期", func(t *testing.T) {
		o := f.createOrder("ADD-DATE-" + uuid.NewString())
		row := saveRow(f, biz.OrderFeePayable, "new", nil, 50)
		row.Fee.IdempotencyKey = uuid.NewString()
		_, err := usecase(o).Add(f.ctx, f.organizationID, f.approverID, o.ID, row.Fee, false, true)
		if !errors.Is(err, biz.ErrOrderStatusConflict) {
			t.Fatalf("单行新增准备后已改期应拒绝，实际 %v", err)
		}
	})

	t.Run("单行显式手改也复核新解析日期", func(t *testing.T) {
		o := f.createOrder("UPDATE-DATE-" + uuid.NewString())
		addSideFee(t, f, o, biz.OrderFeeReceivable, 100)
		one := decimal.NewFromInt(1)
		seed := saveRow(f, biz.OrderFeePayable, "manual", nil, 30)
		seed.Fee.ExchangeRateOverride = &one
		created, err := normalFeeUsecaseForRollback(f).Save(f.ctx, f.organizationID, f.approverID, o.ID, biz.OrderFeePayable, uuid.NewString(), []*biz.OrderFeeSaveRowInput{seed}, allSavePerms)
		if err != nil {
			t.Fatal(err)
		}
		old := created.Fees[0]
		row := saveRow(f, biz.OrderFeePayable, "update", old, 40)
		row.Fee.ExchangeRateOverride = &one
		row.Fee.Version = old.Version
		_, err = usecase(o).Update(f.ctx, f.organizationID, f.approverID, o.ID, old.ID, row.Fee, true, true)
		if !errors.Is(err, biz.ErrOrderStatusConflict) {
			t.Fatalf("显式手改准备后已改期应拒绝，实际 %v", err)
		}
	})

	t.Run("已有手工汇率不因新日期失效", func(t *testing.T) {
		o := f.createOrder("SAVE-MANUAL-DATE-" + uuid.NewString())
		addSideFee(t, f, o, biz.OrderFeeReceivable, 100)
		one := decimal.NewFromInt(1)
		seed := saveRow(f, biz.OrderFeePayable, "manual", nil, 30)
		seed.Fee.ExchangeRateOverride = &one
		created, err := normalFeeUsecaseForRollback(f).Save(f.ctx, f.organizationID, f.approverID, o.ID, biz.OrderFeePayable, uuid.NewString(), []*biz.OrderFeeSaveRowInput{seed}, allSavePerms)
		if err != nil {
			t.Fatal(err)
		}
		old := created.Fees[0]
		row := saveRow(f, biz.OrderFeePayable, "update", old, 40)
		result, err := usecase(o).Save(f.ctx, f.organizationID, f.approverID, o.ID, biz.OrderFeePayable, uuid.NewString(), []*biz.OrderFeeSaveRowInput{row}, allSavePerms)
		if err != nil {
			t.Fatal(err)
		}
		got := result.Fees[0]
		if got.ExchangeRateDate != old.ExchangeRateDate || got.ExchangeRateSource != biz.ExchangeRateSourceManual || !got.TotalAmount.Equal(decimal.NewFromInt(40)) {
			t.Fatalf("普通修改须保留已有手工快照：%+v", got)
		}
	})
}
