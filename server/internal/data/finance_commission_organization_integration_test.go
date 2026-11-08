package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

// 内嵌调整与独立调整必须投影同一真实公司，且保持取消历史和组织读取边界。
func TestCommissionAdjustmentOrganizationProjectionPostgres(t *testing.T) {
	f := newCommissionPostgresFixture(t)
	ctx := context.Background()
	repo := NewCommissionRepo(f.data)
	uc := f.newUsecase(repo)
	input := f.input("organization-projection")
	created, err := uc.Create(ctx, f.organizationID, f.actorID, input)
	if err != nil {
		t.Fatal(err)
	}
	created, err = uc.Confirm(ctx, f.organizationID, f.actorID, created.ID, created.Version)
	if err != nil {
		t.Fatal(err)
	}
	company, err := f.data.db.Organization.Get(ctx, f.organizationID)
	if err != nil {
		t.Fatal(err)
	}
	adjustments := make(map[uuid.UUID]*biz.FinanceCommissionAdjustment)
	for _, cancelled := range []bool{false, true} {
		item, err := uc.CreateAdjustment(ctx, f.organizationID, f.actorID, biz.CreateCommissionAdjustmentInput{
			CommissionID: created.ID, OrderID: f.orderID, Direction: biz.CommissionAdjustmentIncrease,
			Amount: decimal.NewFromInt(1), Reason: "公司投影核验", IdempotencyKey: uuid.NewString(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if cancelled {
			item, err = uc.CancelAdjustment(ctx, f.organizationID, f.actorID, item.ID, item.Version, "保留取消历史")
			if err != nil {
				t.Fatal(err)
			}
		}
		adjustments[item.ID] = item
	}
	assertAdjustment := func(t *testing.T, item *biz.FinanceCommissionAdjustment) {
		t.Helper()
		want := adjustments[item.ID]
		if want == nil || item.OrganizationID != company.ID || item.OrganizationName != company.Name {
			t.Fatalf("调整真实公司投影不符: %+v，公司=%s/%s", item, company.ID, company.Name)
		}
		if item.Status != want.Status || item.Version != want.Version || !item.Amount.Equal(want.Amount) {
			t.Fatalf("读取不得改变状态、版本或金额: got=%+v want=%+v", item, want)
		}
	}
	assertCommission := func(t *testing.T, item *biz.FinanceCommission, err error) {
		t.Helper()
		if err != nil || item == nil {
			t.Fatalf("读取提成失败: item=%+v err=%v", item, err)
		}
		if item.ID != created.ID || item.OrganizationID != company.ID || item.OrganizationName != company.Name || len(item.Adjustments) != 2 {
			t.Fatalf("父单或内嵌调整不符: %+v", item)
		}
		for _, adjustment := range item.Adjustments {
			assertAdjustment(t, adjustment)
		}
	}
	filter := biz.CommissionFilter{Page: 1, PageSize: 20}
	t.Run("详情", func(t *testing.T) {
		item, err := repo.GetScoped(ctx, []uuid.UUID{f.organizationID}, created.ID)
		assertCommission(t, item, err)
	})
	t.Run("幂等读取", func(t *testing.T) {
		item, err := repo.GetByKey(ctx, f.organizationID, input.IdempotencyKey)
		assertCommission(t, item, err)
	})
	t.Run("列表", func(t *testing.T) {
		result, err := repo.ListScoped(ctx, []uuid.UUID{f.organizationID}, filter)
		if err != nil || result == nil || result.Total != 1 || len(result.Items) != 1 {
			t.Fatalf("列表结果不符: %+v err=%v", result, err)
		}
		assertCommission(t, result.Items[0], nil)
	})
	t.Run("导出", func(t *testing.T) {
		items, err := repo.ExportBatchScoped(ctx, []uuid.UUID{f.organizationID}, filter)
		if err != nil || len(items) != 1 {
			t.Fatalf("导出结果不符: %+v err=%v", items, err)
		}
		assertCommission(t, items[0], nil)
	})
	t.Run("独立调整", func(t *testing.T) {
		result, err := repo.ListAdjustmentsScoped(ctx, []uuid.UUID{f.organizationID}, biz.CommissionAdjustmentFilter{Page: 1, PageSize: 20})
		if err != nil || result == nil || result.Total != 2 || len(result.Items) != 2 {
			t.Fatalf("调整列表结果不符: %+v err=%v", result, err)
		}
		for _, item := range result.Items {
			assertAdjustment(t, item)
			detail, err := repo.GetAdjustmentScoped(ctx, []uuid.UUID{f.organizationID}, item.ID)
			if err != nil {
				t.Fatal(err)
			}
			assertAdjustment(t, detail)
		}
	})
	t.Run("范围外读取拒绝", func(t *testing.T) {
		otherScope := []uuid.UUID{uuid.New()}
		if _, err := repo.GetScoped(ctx, otherScope, created.ID); !errors.Is(err, biz.ErrCommissionNotFound) {
			t.Fatalf("跨公司详情应不可见: %v", err)
		}
		if item, err := repo.GetByKey(ctx, otherScope[0], input.IdempotencyKey); err != nil || item != nil {
			t.Fatalf("跨公司幂等读取应为空: %+v err=%v", item, err)
		}
		list, err := repo.ListScoped(ctx, otherScope, filter)
		if err != nil || list.Total != 0 || len(list.Items) != 0 {
			t.Fatalf("跨公司列表应为空: %+v err=%v", list, err)
		}
		items, err := repo.ExportBatchScoped(ctx, otherScope, filter)
		if err != nil || len(items) != 0 {
			t.Fatalf("跨公司导出应为空: %+v err=%v", items, err)
		}
		for id := range adjustments {
			if _, err := repo.GetAdjustmentScoped(ctx, otherScope, id); !errors.Is(err, biz.ErrCommissionAdjustmentNotFound) {
				t.Fatalf("跨公司调整详情应不可见: %v", err)
			}
		}
		listAdjustments, err := repo.ListAdjustmentsScoped(ctx, otherScope, biz.CommissionAdjustmentFilter{Page: 1, PageSize: 20})
		if err != nil || listAdjustments.Total != 0 || len(listAdjustments.Items) != 0 {
			t.Fatalf("跨公司调整列表应为空: %+v err=%v", listAdjustments, err)
		}
	})
}
