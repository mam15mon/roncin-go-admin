package data

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	airlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/airmasterwaybillorderlink"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	sealink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

func TestSeaMasterIdentityCandidatesAndDirectIDPostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newOrderPostgresFixture(t, d)
	ctx := context.Background()
	uc := f.newUsecase()
	in := f.validInput()
	no := in.SeaMasterBillInput.MasterNo
	importMaster := d.db.SeaMasterBill.Create().SetOrganizationID(f.organizationID).SetBusinessType("SI").SetShippingLineID(f.shippingLineID).SetMasterNo(no).SetNormalizedMasterNo(no).SaveX(ctx)
	match, err := uc.MatchSeaMasterBillCandidate(ctx, f.organizationID, f.shippingLineID, no, nil, biz.OrderBusinessSE)
	if err != nil || match.Matched {
		t.Fatalf("进口孤立主单被当作出口候选: %+v %v", match, err)
	}
	o, err := uc.Create(ctx, f.organizationID, f.actorID, in)
	if err != nil {
		t.Fatal(err)
	}
	master := d.db.SeaMasterBill.GetX(ctx, o.SeaMasterBill.MasterBillID)
	if master.ID == importMaster.ID || master.BusinessType != "SE" {
		t.Fatal("跨方向同号未创建独立出口主单")
	}
	match, err = uc.MatchSeaMasterBillCandidate(ctx, f.organizationID, f.shippingLineID, no, nil, biz.OrderBusinessSE)
	if err != nil || !match.Matched || match.Candidate.ID != master.ID || match.Candidate.BusinessType != biz.OrderBusinessSE {
		t.Fatalf("出口候选身份错误: %+v %v", match, err)
	}
	beforeOrders := d.db.Order.Query().CountX(ctx)
	bad := f.validInput()
	teID, teVersion := o.SeaMasterBill.TransportExecutionID, o.SeaMasterBill.TransportExecutionVersion
	bad.SeaMasterBillInput.CandidateID = &importMaster.ID
	bad.SeaMasterBillInput.ExpectedCandidateVersion = &importMaster.Version
	bad.SeaMasterBillInput.CandidateTEID = &teID
	bad.SeaMasterBillInput.ExpectedCandidateTEVersion = &teVersion
	if _, err = uc.Create(ctx, f.organizationID, f.actorID, bad); err == nil {
		t.Fatal("直接关联进口主单未拒绝")
	}
	if d.db.Order.Query().CountX(ctx) != beforeOrders {
		t.Fatal("失败关联留下部分订单")
	}
	link := d.db.SeaMasterBillOrderLink.Query().Where(sealink.OrderIDEQ(o.ID)).OnlyX(ctx)
	d.db.SeaMasterBillOrderLink.UpdateOneID(link.ID).SetMasterBillID(importMaster.ID).SaveX(ctx)
	if _, _, err := loadCurrentDocumentBase(ctx, d.db, f.organizationID, o.ID, biz.SeaDocumentTypeMasterBill, importMaster.ID, o.Version, importMaster.Version, uuid.New()); err == nil {
		t.Fatal("海运单证预览接受了跨方向主单")
	}
	d.db.SeaMasterBillOrderLink.UpdateOneID(link.ID).SetMasterBillID(master.ID).SaveX(ctx)
	// 主单身份不从活动成员推导；清空关系后仍可按其自身类型匹配。
	d.db.SeaMasterBillOrderLink.Update().Where(sealink.MasterBillIDEQ(master.ID)).SetStatus(sealink.StatusENDED).SaveX(ctx)
	match, err = uc.MatchSeaMasterBillCandidate(ctx, f.organizationID, f.shippingLineID, no, nil, biz.OrderBusinessSE)
	if err != nil || !match.Matched || match.Candidate.ID != master.ID || match.Candidate.MemberCount != 0 {
		t.Fatalf("无活动成员主单丢失自身身份: %+v %v", match, err)
	}
}

func TestSeaMasterIdentitySharedMembersAndRestorePostgres(t *testing.T) {
	f, o, source := restoreFixture(t, biz.SeaDocumentStructureHouse)
	ctx := context.Background()
	d := f.data
	// 构造错误存量，验证恢复只接受同方向主单，且状态与历史均不部分提交。
	if _, err := d.sqlDB.ExecContext(ctx, "UPDATE sea_master_bills SET business_type = 'SI' WHERE id = $1", source.MasterBillID); err != nil {
		t.Fatal(err)
	}
	beforeLinks := d.db.SeaMasterBillOrderLink.Query().CountX(ctx)
	if _, err := f.newUsecase().TransitionTermination(ctx, f.organizationID, f.actorID, o.ID, o.Version, biz.OrderTerminationActive, nil, "恢复"); err == nil {
		t.Fatal("跨方向恢复未拒绝")
	}
	if d.db.Order.GetX(ctx, o.ID).TerminationStatus != orderent.TerminationStatusTERMINATED || d.db.SeaMasterBillOrderLink.Query().CountX(ctx) != beforeLinks {
		t.Fatal("失败恢复留下状态或关联")
	}
	if _, err := d.sqlDB.ExecContext(ctx, "UPDATE sea_master_bills SET business_type = 'SE' WHERE id = $1", source.MasterBillID); err != nil {
		t.Fatal(err)
	}
	member := d.db.Order.Create().SetOrganizationID(f.organizationID).SetBusinessType(orderent.BusinessTypeSI).SetFlowStatus(orderent.FlowStatusIMPORT_NOT_APPLICABLE).SetTradeDirection(orderent.TradeDirectionImport).SetCustomerID(f.partnerID).SetOrderNo("SI-" + uuid.NewString()).SetTradeTerm(orderent.TradeTermFOB).SetPaymentTerm(orderent.PaymentTermPREPAID).SetIdempotencyKey(uuid.NewString()).SaveX(ctx)
	link := d.db.SeaMasterBillOrderLink.Create().SetOrganizationID(f.organizationID).SetOrderID(member.ID).SetMasterBillID(source.MasterBillID).SetTransportExecutionID(source.TransportExecutionID).SetDocumentStructure(sealink.DocumentStructureHOUSE).SaveX(ctx)
	if err := d.WithTx(ctx, func(tx *ent.Tx) error { _, err := ensureSharedMBLNotLocked(ctx, tx, source.MasterBillID); return err }); err == nil {
		t.Fatal("共享主单隐藏了跨方向成员")
	}
	// Link 公司错误同样阻断，不能被当前公司查询静默过滤。
	d.db.Order.UpdateOneID(member.ID).SetBusinessType(orderent.BusinessTypeSE).SetFlowStatus(orderent.FlowStatusDRAFT).SetTradeDirection(orderent.TradeDirectionExport).SaveX(ctx)
	otherOrg := d.db.Organization.Create().SetCode(uuid.NewString()).SetName("另一公司").SetKind("company").SetBaseCurrency("CNY").SaveX(ctx)
	t.Cleanup(func() {
		d.db.SeaMasterBillOrderLink.UpdateOneID(link.ID).SetOrganizationID(f.organizationID).SaveX(ctx)
	})
	d.db.SeaMasterBillOrderLink.UpdateOneID(link.ID).SetOrganizationID(otherOrg.ID).SaveX(ctx)
	if _, err := querySeaMasterBillActiveMemberLinks(ctx, d.db, f.organizationID, source.MasterBillID, false); err == nil {
		t.Fatal("共享主单隐藏了跨公司关联")
	}
}

func TestAirMasterIdentityCandidatesWritesAndRestorePostgres(t *testing.T) {
	d, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	f := newAirExportFixture(t, d)
	ctx := context.Background()
	in := f.input("78112345675", "H001")
	importMaster := d.db.AirMasterWaybill.Create().SetBusinessType("AI").SetOrganizationID(f.organizationID).SetMasterNo("78112345675").SetAirlineID(f.airlineID).SetOriginAirportID(f.originID).SetDestinationAirportID(f.destinationID).SaveX(ctx)
	air := biz.NewAirExportUsecase(NewAirExportRepo(d))
	options := biz.SelectorListOptions{Page: 1, PageSize: 20}
	rows, total, err := air.List(ctx, f.organizationID, options)
	if err != nil || total != 0 || len(rows) != 0 {
		t.Fatalf("进口主单泄漏到出口候选: %d %v", total, err)
	}
	o := f.create(t, in)
	master := o.AirExport.Master
	rows, total, err = air.List(ctx, f.organizationID, options)
	if err != nil || total != 1 || rows[0].ID != master.ID || rows[0].BusinessType != biz.OrderBusinessAE {
		t.Fatalf("同号候选未按主单自身身份隔离: %d %v", total, err)
	}
	bad := f.input("78112345675", "H002")
	bad.AirExportInput.MasterWaybillID = &importMaster.ID
	bad.AirExportInput.ExpectedMasterVersion = importMaster.Version
	if _, err := f.newUsecase().Create(ctx, f.organizationID, f.actorID, bad); err == nil {
		t.Fatal("空运创建接受进口主单ID")
	}
	if _, err := air.Preview(ctx, f.organizationID, importMaster.ID, importMaster.Version, &master.Content); err == nil {
		t.Fatal("空运预览接受进口主单ID")
	}
	if d.db.Order.Query().CountX(ctx) != 1 {
		t.Fatal("跨方向建票失败留下部分订单")
	}
	uc := f.newUsecase()
	kind := biz.OrderTerminationOther
	for _, state := range []biz.OrderTerminationStatus{biz.OrderTerminationTerminating, biz.OrderTerminationTerminated} {
		o, err = uc.TransitionTermination(ctx, f.organizationID, f.actorID, o.ID, o.Version, state, &kind, "终止")
		if err != nil {
			t.Fatal(err)
		}
	}
	if err := air.Link(ctx, f.organizationID, f.actorID, importMaster.ID, o.ID, importMaster.Version, o.Version, "H001", "恢复"); err == nil {
		t.Fatal("空运恢复接受进口主单ID")
	}
	if d.db.AirMasterWaybillOrderLink.Query().Where(airlink.OrderIDEQ(o.ID)).CountX(ctx) != 1 || d.db.Order.GetX(ctx, o.ID).TerminationStatus != orderent.TerminationStatusTERMINATED {
		t.Fatal("失败恢复留下新关联或改写状态")
	}
	rows, total, err = air.List(ctx, f.organizationID, options)
	if err != nil || total != 1 || rows[0].ID != master.ID || rows[0].MemberCount != 0 {
		t.Fatalf("航空无活动成员丢失出口身份: %d %v", total, err)
	}
}

func TestMasterContentHashIncludesBusinessIdentity(t *testing.T) {
	master := &ent.SeaMasterBill{BusinessType: "SE", MasterNo: "SAME", NormalizedMasterNo: "SAME"}
	exportHash := computeMBLContentHash(master)
	master.BusinessType = "SI"
	if computeMBLContentHash(master) == exportHash {
		t.Fatal("主单哈希未包含业务类型")
	}
}
