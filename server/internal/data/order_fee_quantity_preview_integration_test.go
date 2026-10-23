package data

import (
	"context"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	billingunitent "github.com/roncin/roncin-go-admin/server/internal/data/ent/billingunit"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
)

func TestFeeQuantityPreviewPostgres(t *testing.T) {
	f := newSeaDocumentChangeFixture(t)
	ctx := context.Background()
	client := f.data.db
	repo := NewOrderFeeRepo(f.data)
	uc := biz.NewOrderFeeUsecase(repo, nil, nil, nil, nil)
	unit := func(code string, container bool) *ent.BillingUnit {
		t.Helper()
		item, err := client.BillingUnit.Query().Where(billingunitent.CodeEQ(code)).Only(ctx)
		if ent.IsNotFound(err) {
			item, err = client.BillingUnit.Create().SetCode(code).SetName(code).SetIsContainerUnit(container).Save(ctx)
		}
		if err != nil {
			t.Fatal(err)
		}
		return item
	}
	preview := func(code, source string) *biz.FeeQuantityPreview {
		t.Helper()
		got, err := uc.PreviewFeeQuantity(ctx, f.orgID, f.orderID, unit(code, code == "40HQ").ID, source, "GROSS_WEIGHT")
		if err != nil {
			t.Fatal(err)
		}
		return got
	}
	expect := func(code, source, want string) {
		t.Helper()
		got := preview(code, source)
		if !got.Available || got.Quantity != want {
			t.Fatalf("%s/%s=%+v，期望%s", code, source, got, want)
		}
	}
	if got := preview("CBM", "ACTUAL"); got.Available || !strings.Contains(got.Reason, "尚未登记") {
		t.Fatalf("缺少货物不能默认数量：%+v", got)
	}
	cargo := client.OrderCargoItem.Create().SetOrganizationID(f.orgID).SetOrderID(f.orderID).SetCargoName("本票实际货物").SetPackageCount(10).SetGrossWeightKg(2500).SetVolumeCbm(8).SaveX(ctx)
	// 独占箱上也有件重尺，但不得与订单货物明细相加。
	spec, err := client.MasterDataItem.Query().Where(masterdataent.KindEQ(masterdataent.KindContainerSpec), masterdataent.CodeEQ("40HQ")).Only(ctx)
	if ent.IsNotFound(err) {
		spec, err = client.MasterDataItem.Create().SetKind(masterdataent.KindContainerSpec).SetCode("40HQ").SetName("40HQ").Save(ctx)
	}
	if err != nil {
		t.Fatal(err)
	}
	client.OrderContainerRequest.Create().SetOrderID(f.orderID).SetContainerSpecID(spec.ID).SetQuantity(3).SaveX(ctx)
	for _, no := range []string{"QTY-CNTR-1", "QTY-CNTR-2"} {
		client.OrderContainer.Create().SetOrganizationID(f.orgID).SetOrderID(f.orderID).SetContainerNo(no).SetContainerSpecID(spec.ID).SetPackageCount(10).SetGrossWeightKg(2500).SetVolumeCbm(8).SaveX(ctx)
	}
	for code, want := range map[string]string{"KG": "2500", "CBM": "8", "TON": "2.5", "RT": "8"} {
		expect(code, "ACTUAL", want)
	}
	expect("40HQ", "PLANNED", "3")
	expect("40HQ", "ACTUAL", "2")
	expect("CONT", "ACTUAL", "2")
	// 一个不同订单即使同公司也不贡献货量；拆票后逐票货物拥有独立归属。
	second := client.Order.Create().SetOrganizationID(f.orgID).SetCustomerID(f.partnerID).SetOrderNo("QTY-OTHER-" + uuid.NewString()).SetIdempotencyKey(uuid.NewString()).SetBusinessType("SE").SetTradeDirection("export").SetTradeTerm("FOB").SetPaymentTerm("PREPAID").SaveX(ctx)
	client.OrderCargoItem.Create().SetOrganizationID(f.orgID).SetOrderID(second.ID).SetCargoName("另一票货物").SetPackageCount(1).SetGrossWeightKg(9999).SetVolumeCbm(99).SaveX(ctx)
	expect("CBM", "ACTUAL", "8")
	if _, err := repo.QuantitySource(ctx, uuid.New(), f.orderID, unit("KG", false).ID, "ACTUAL"); err != biz.ErrOrderFeeNotFound {
		t.Fatalf("跨组织必须拒绝，得到%v", err)
	}
	if got := preview("KG", "PLANNED"); got.Available || !strings.Contains(got.Reason, "仅支持") {
		t.Fatalf("货量计划不得回退：%+v", got)
	}
	// 精确 numeric 大值与六位体积由 SQL 直接构造，避免测试输入先经过 float64。
	if _, err := f.data.sqlDB.ExecContext(ctx, "UPDATE order_cargo_items SET gross_weight_kg=$1, volume_cbm=$2 WHERE id=$3", "999999999999999.123", "8.123456", cargo.ID); err != nil {
		t.Fatal(err)
	}
	raw, err := repo.QuantitySource(ctx, f.orgID, f.orderID, unit("KG", false).ID, "ACTUAL")
	if err != nil || raw.GrossWeightKG.String() != "999999999999999.123" || raw.VolumeCBM.String() != "8.123456" {
		t.Fatalf("numeric聚合不得丢失精度：%+v %v", raw, err)
	}
	if got := preview("CBM", "ACTUAL"); got.Available || !strings.Contains(got.Reason, "四位小数") {
		t.Fatalf("不得自动舍入：%+v", got)
	}
	client.OrderCargoItem.UpdateOneID(cargo.ID).SetGrossWeightKg(2500).SetVolumeCbm(8).SaveX(ctx)
	shared := createHouseBillSharedAllocation(t, f, uuid.NewString()[:8])
	// 模拟另一票占用同一共享箱的货量，整箱合计绝不能计入当前订单。
	client.SeaSharedContainer.UpdateOne(shared).SetGrossWeightKg("9999").SetVolumeCbm("99").SaveX(ctx)
	expect("CBM", "ACTUAL", "9") // 本票货物8 + 本票共享分配货物1，只各计一次。
	if got := preview("CONT", "ACTUAL"); got.Available || !strings.Contains(got.Reason, "共享箱") {
		t.Fatalf("共享箱必须显式拒绝整箱推导：%+v", got)
	}
	expect("40HQ", "PLANNED", "3")
	unknown := unit("UNMAPPED-"+uuid.NewString()[:8], true)
	got, err := uc.PreviewFeeQuantity(ctx, f.orgID, f.orderID, unknown.ID, "PLANNED", "GROSS_WEIGHT")
	if err != nil || got.Available || !strings.Contains(got.Reason, "精确") {
		t.Fatalf("无箱型映射不得回退：%+v %v", got, err)
	}
	client.BillingUnit.UpdateOne(unknown).SetEnabled(false).SaveX(ctx)
	got, err = uc.PreviewFeeQuantity(ctx, f.orgID, f.orderID, unknown.ID, "ACTUAL", "GROSS_WEIGHT")
	if err != nil || got.Available || !strings.Contains(got.Reason, "停用") {
		t.Fatalf("停用单位：%+v %v", got, err)
	}
}
