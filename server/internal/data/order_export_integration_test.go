package data

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataent "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

func TestOrderExportCompleteSnapshotPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	f := newAirExportFixture(t, d)
	f.deferCleanupToSchema = true
	uc := f.newUsecase()
	sea := f.validInput()
	direct := biz.SeaDocumentStructureDirect
	sea.SeaDocumentInput = &biz.SeaOrderDocumentInput{DocumentStructure: &direct}
	se, err := uc.Create(ctx, f.organizationID, f.actorID, sea)
	if err != nil {
		t.Fatal(err)
	}
	si := f.create(t, seaImportInput(f.orderPostgresFixture, true))
	ae := f.create(t, directAirInputForBusiness(f, biz.OrderBusinessAE, "78112345675"))
	ai := f.create(t, airInputForBusiness(f, biz.OrderBusinessAI, "78112345675", "IMPORT-HOUSE"))
	repo := NewOrderRepo(d)
	for _, o := range []*biz.Order{se, si, ae, ai} {
		scope := []biz.OrderOrganizationScope{{BusinessType: o.BusinessType, OrganizationIDs: []uuid.UUID{o.OrganizationID}}}
		exported, err := repo.Export(ctx, scope, biz.OrderListOptions{BusinessType: o.BusinessType, SortBy: "orderNo", SortOrder: "ascend"})
		if err != nil || len(exported.Items) != 1 || exported.Items[0].ID != o.ID || exported.GeneratedAt.IsZero() {
			t.Fatalf("四类完整结果错误: %+v %v", exported, err)
		}
		if o.BusinessType.IsSea() && exported.Items[0].SeaMasterBill == nil || o.BusinessType.IsAir() && exported.Items[0].AirExport == nil {
			t.Fatalf("完整行丢单证投影: %+v", exported.Items[0])
		}
	}
	// 大结果通过一次无分页查询返回；每行保留真实海运DIRECT关联，不靠超大pageSize。
	link := d.db.SeaMasterBillOrderLink.Query().Where().AllX(ctx)
	var execution uuid.UUID
	for _, l := range link {
		if l.OrderID == se.ID {
			execution = l.TransportExecutionID
		}
	}
	builders := make([]*ent.OrderCreate, 0, 205)
	for i := 0; i < 205; i++ {
		builders = append(builders, d.db.Order.Create().SetID(uuid.New()).SetOrganizationID(f.organizationID).SetCustomerID(f.partnerID).SetOrderNo(fmt.Sprintf("EXPORT-%03d", i)).SetIdempotencyKey(uuid.NewString()).SetBusinessType(orderent.BusinessTypeSE).SetTradeDirection(orderent.TradeDirectionExport).SetPaymentTerm(orderent.PaymentTermPREPAID).SetCustomerReferenceNo("中文,引号\"多行\n记录").SetNotes("完整导出备注").SetTotalPackages(0).SetTotalGrossWeightKg(6.7183))
	}
	rows := d.db.Order.CreateBulk(builders...).SaveX(ctx)
	links := make([]*ent.SeaMasterBillOrderLinkCreate, 0, len(rows))
	for _, o := range rows {
		links = append(links, d.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(o.ID).SetMasterBillID(se.SeaMasterBill.MasterBillID).SetTransportExecutionID(execution).SetDocumentStructure("DIRECT"))
	}
	d.db.SeaMasterBillOrderLink.CreateBulk(links...).SaveX(ctx)
	spec := d.db.MasterDataItem.Create().SetKind(masterdataent.KindContainerSpec).SetCode("EXPORT-OLD").SetName("原箱型名称").SaveX(ctx)
	d.db.OrderContainerRequest.Create().SetOrderID(rows[0].ID).SetContainerSpecID(spec.ID).SetQuantity(3).SaveX(ctx)
	scope := []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{f.organizationID}}}
	filter := biz.OrderListOptions{BusinessType: biz.OrderBusinessSE, Keyword: "EXPORT-", SortBy: "orderNo", SortOrder: "ascend"}
	exported, err := repo.Export(ctx, scope, filter)
	if err != nil {
		t.Fatal(err)
	}
	if len(exported.Items) != 205 {
		t.Fatalf("完整导出不应截200/当前页: %d", len(exported.Items))
	}
	for i, o := range exported.Items {
		if o.OrderNo != fmt.Sprintf("EXPORT-%03d", i) || o.CustomerReferenceNo != "中文,引号\"多行\n记录" || o.TotalPackages == nil || *o.TotalPackages != 0 || o.TotalGrossWeightKg == nil || *o.TotalGrossWeightKg != 6.7183 || o.SeaMasterBill == nil {
			t.Fatalf("排序/原文/存在性/精度/投影丢失: %+v", o)
		}
	}
	filter.Page, filter.PageSize = 1, 200
	page, err := uc.List(ctx, scope, filter)
	if err != nil || len(page.Items) != 200 || page.Total != 205 {
		t.Fatalf("常规分页边界改变: %+v %v", page, err)
	}
	filter.PageSize = 201
	if _, err = uc.List(ctx, scope, filter); err == nil {
		t.Fatal("普通列表错误接受201")
	}
	filter.Page, filter.PageSize = 0, 0
	empty, err := repo.Export(ctx, []biz.OrderOrganizationScope{{BusinessType: biz.OrderBusinessSE, OrganizationIDs: []uuid.UUID{uuid.New()}}}, filter)
	if err != nil || len(empty.Items) != 0 {
		t.Fatal("跨公司导出泄漏", err)
	}
	filter.SortOrder = "descend"
	reverse, err := repo.Export(ctx, scope, filter)
	if err != nil || reverse.Items[0].OrderNo != "EXPORT-204" {
		t.Fatal("完整排序未在全筛选上生效", err)
	}
	// 建立真实REPEATABLE READ快照后另一连接修改行与客户投影，导出仍须同快照。
	err = d.withReadSnapshot(ctx, func(snapshotCtx context.Context) error {
		c, e := d.client(snapshotCtx)
		if e != nil {
			return e
		}
		if _, e = c.Order.Query().Count(snapshotCtx); e != nil {
			return e
		}
		if _, e = d.db.Order.UpdateOneID(rows[0].ID).SetNotes("后提交备注").Save(ctx); e != nil {
			return e
		}
		if _, e = d.db.Partner.UpdateOneID(f.partnerID).SetLegalName("后提交客户名").Save(ctx); e != nil {
			return e
		}
		if _, e = d.db.MasterDataItem.UpdateOneID(spec.ID).SetName("后提交箱型名称").Save(ctx); e != nil {
			return e
		}
		snapshot, e := repo.Export(snapshotCtx, scope, filter)
		if e != nil {
			return e
		}
		for _, o := range snapshot.Items {
			if o.ID == rows[0].ID && (o.Notes != "完整导出备注" || o.CustomerName == "后提交客户名" || len(o.ContainerRequests) != 1 || o.ContainerRequests[0].ContainerSpecCode != "EXPORT-OLD" || o.ContainerRequests[0].ContainerSpecName != "原箱型名称") {
				t.Fatal("导出混入快照外新事实")
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	latest, err := repo.Export(ctx, scope, filter)
	if err != nil {
		t.Fatal(err)
	}
	for _, o := range latest.Items {
		if o.ID == rows[0].ID && (o.Notes != "后提交备注" || o.CustomerName != "后提交客户名" || len(o.ContainerRequests) != 1 || o.ContainerRequests[0].ContainerSpecCode != "EXPORT-OLD" || o.ContainerRequests[0].ContainerSpecName != "后提交箱型名称") {
			t.Fatal("后续导出未采用新快照")
		}
	}
	if err = d.WithinTransaction(ctx, func(txCtx context.Context) error {
		_, e := repo.Export(txCtx, scope, filter)
		if e == nil {
			t.Fatal("写事务中不应伪装一致读快照")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
}

// Ent与领域都应按Unicode字符限制，不按UTF-8字节截断本期已批准文本字段。
func TestOrderBusinessTextUnicodeBoundariesPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	f := newAirExportFixture(t, d)
	f.deferCleanupToSchema = true
	for _, kind := range []biz.OrderBusinessType{biz.OrderBusinessAE, biz.OrderBusinessAI} {
		in := directAirInputForBusiness(f, kind, "78112345675")
		in.CustomerReferenceNo = strings.Repeat("客", 100)
		in.InternalReferenceNo = strings.Repeat("内", 100)
		in.ContractNo = strings.Repeat("合", 100)
		in.FactoryName = strings.Repeat("厂", 200)
		in.Notes = strings.Repeat("备", 1000)
		in.BookingNotes = strings.Repeat("订", 1000)
		in.AllocationNotes = strings.Repeat("配", 1000)
		in.OperationNotes = strings.Repeat("操", 1000)
		in.GoodsDescription = strings.Repeat("货", 1000)
		in.SpecialRequirements = strings.Repeat("特", 1000)
		in.ShipperShortName = strings.Repeat("发", 200)
		in.ConsigneeShortName = strings.Repeat("收", 200)
		in.BookingNo = strings.Repeat("订", 100)
		in.AirExportInput.Master.Notes = strings.Repeat("航", 1000)
		in.UNNumber = "1234"
		in.HazardClass = "3"
		in.DeclarationCutoffAt = "2026-11-16T00:00:00Z"
		packing := biz.OrderDangerousPackingII
		in.DangerousPackingGroup = &packing
		o := f.create(t, in)
		if o.CustomerReferenceNo != in.CustomerReferenceNo || o.InternalReferenceNo != in.InternalReferenceNo || o.ContractNo != in.ContractNo || o.FactoryName != in.FactoryName || o.Notes != in.Notes || o.BookingNotes != in.BookingNotes || o.AllocationNotes != in.AllocationNotes || o.OperationNotes != in.OperationNotes || o.GoodsDescription != in.GoodsDescription || o.SpecialRequirements != in.SpecialRequirements || o.ShipperShortName != in.ShipperShortName || o.ConsigneeShortName != in.ConsigneeShortName || o.BookingNo != in.BookingNo || o.AirExport.Master.Content.Notes != in.AirExportInput.Master.Notes {
			t.Fatal("合法中文边界落库丢失")
		}
		// 更新沿已合并的完整领域值：保持原PG或显式清除；wire存在性另由service真实ProtoJSON验证。
		preserved := *o
		preserved.AirExportInput = nil
		preserved.Notes = "省略危险包装字段"
		o, err := f.newUsecase().UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &preserved)
		if err != nil {
			t.Fatal(err)
		}
		if o.DangerousPackingGroup == nil || *o.DangerousPackingGroup != packing {
			t.Fatal("省略PG不应被清空")
		}
		cleared := *o
		cleared.AirExportInput = nil
		cleared.UNNumber = ""
		cleared.HazardClass = ""
		cleared.DeclarationCutoffAt = ""
		cleared.DangerousPackingGroup = nil
		o, err = f.newUsecase().UpdateDraft(ctx, f.organizationID, f.actorID, o.ID, o.Version, &cleared)
		if err != nil {
			t.Fatal(err)
		}
		if o.UNNumber != "" || o.HazardClass != "" || o.DeclarationCutoffAt != "" || o.DangerousPackingGroup != nil {
			t.Fatal("危险字段显式清空后重读复活")
		}
		for _, tc := range []struct {
			name  string
			limit int
			set   func(*ent.OrderUpdateOne, string)
		}{{"客户号", 100, func(q *ent.OrderUpdateOne, s string) { q.SetCustomerReferenceNo(s) }}, {"内部号", 100, func(q *ent.OrderUpdateOne, s string) { q.SetInternalReferenceNo(s) }}, {"合约", 100, func(q *ent.OrderUpdateOne, s string) { q.SetContractNo(s) }}, {"工厂", 200, func(q *ent.OrderUpdateOne, s string) { q.SetFactoryName(s) }}, {"备注", 1000, func(q *ent.OrderUpdateOne, s string) { q.SetNotes(s) }}, {"订舱", 1000, func(q *ent.OrderUpdateOne, s string) { q.SetBookingNotes(s) }}, {"配舱", 1000, func(q *ent.OrderUpdateOne, s string) { q.SetAllocationNotes(s) }}, {"操作", 1000, func(q *ent.OrderUpdateOne, s string) { q.SetOperationNotes(s) }}, {"货描", 1000, func(q *ent.OrderUpdateOne, s string) { q.SetGoodsDescription(s) }}, {"特殊要求", 1000, func(q *ent.OrderUpdateOne, s string) { q.SetSpecialRequirements(s) }}, {"发货简称", 200, func(q *ent.OrderUpdateOne, s string) { q.SetShipperShortName(s) }}, {"收货简称", 200, func(q *ent.OrderUpdateOne, s string) { q.SetConsigneeShortName(s) }}, {"订舱号", 100, func(q *ent.OrderUpdateOne, s string) { q.SetBookingNo(s) }}} {
			boundary := d.db.Order.UpdateOneID(o.ID)
			tc.set(boundary, strings.Repeat("中", tc.limit))
			if _, err = boundary.Save(ctx); err != nil {
				t.Fatalf("Ent拒绝合法中文%s: %v", tc.name, err)
			}
			q := d.db.Order.UpdateOneID(o.ID)
			tc.set(q, strings.Repeat("中", tc.limit+1))
			if _, err = q.Save(ctx); err == nil {
				t.Fatalf("Ent未拒绝超限中文%s", tc.name)
			}
		}
		if _, err = d.db.AirMasterWaybill.UpdateOneID(o.AirExport.Master.ID).SetNotes(strings.Repeat("航", 1001)).Save(ctx); err == nil {
			t.Fatal("航空主单备注超限未拒绝")
		}
		if _, err = f.newUsecase().Get(ctx, f.organizationID, o.ID); err != nil {
			t.Fatal(err)
		}
	}
}

func TestSeaOrderVesselVoyageUnicodePostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	f := newOrderPostgresFixture(t, d)
	f.deferCleanupToSchema = true
	for _, business := range []biz.OrderBusinessType{biz.OrderBusinessSE, biz.OrderBusinessSI} {
		for i, text := range []struct{ vessel, voyage string }{{strings.Repeat("船", 90), strings.Repeat("航", 7)}, {"船", strings.Repeat("航", 64)}} {
			in := f.validInput()
			if business == biz.OrderBusinessSI {
				in = seaImportInput(f, true)
			} else {
				mode := biz.SeaDocumentStructureDirect
				in.SeaDocumentInput = &biz.SeaOrderDocumentInput{DocumentStructure: &mode}
			}
			in.SeaMasterBillInput.MasterNo = fmt.Sprintf("UNICODE%s%03d", business, i)
			in.VesselVoyage = text.vessel + " / " + text.voyage
			saved, err := f.newUsecase().Create(ctx, f.organizationID, f.actorID, in)
			if err != nil {
				t.Fatal(err)
			}
			current, err := f.newUsecase().Get(ctx, f.organizationID, saved.ID)
			if err != nil {
				t.Fatal(err)
			}
			if current.VesselVoyage != in.VesselVoyage || current.SeaMasterBill == nil || current.SeaMasterBill.VesselName != text.vessel || current.SeaMasterBill.VoyageNo != text.voyage {
				t.Fatalf("海运中文船名航次边界未完整往返: %+v", current.SeaMasterBill)
			}
			// 相同合法船名航次还必须能进入不可变版本，不能只当前行通过。
			if err = d.WithTx(ctx, func(tx *ent.Tx) error {
				execution, err := tx.SeaTransportExecution.Get(ctx, current.SeaMasterBill.TransportExecutionID)
				if err != nil {
					return err
				}
				version, err := ensureSeaTransportExecutionVersion(ctx, tx, f.organizationID, execution, &f.actorID, "ORDER_LOCK", nil, nil, nil, nil)
				if err != nil {
					return err
				}
				if version.VesselName != text.vessel || version.VoyageNo != text.voyage {
					t.Fatal("航程不可变版本丢失中文边界")
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
			if _, err = d.db.Order.UpdateOneID(current.ID).SetVesselVoyage(strings.Repeat("船", 101)).Save(ctx); err == nil {
				t.Fatal("订单船名航次101字符应拒绝")
			}
			te := current.SeaMasterBill.TransportExecutionID
			if _, err = d.db.SeaTransportExecution.UpdateOneID(te).SetVesselName(strings.Repeat("船", 128)).SetVoyageNo(strings.Repeat("航", 64)).Save(ctx); err != nil {
				t.Fatalf("独立共享航程原字符上限被字节误限: %v", err)
			}
			if _, err = d.db.SeaTransportExecution.UpdateOneID(te).SetVesselName(strings.Repeat("船", 129)).Save(ctx); err == nil {
				t.Fatal("船名129字符应拒绝")
			}
			if _, err = d.db.SeaTransportExecution.UpdateOneID(te).SetVoyageNo(strings.Repeat("航", 65)).Save(ctx); err == nil {
				t.Fatal("航次65字符应拒绝")
			}
		}
	}
}
