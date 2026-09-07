package data

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	numbersequenceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seaorderspliteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitevent"
	seasharedcontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainer"
	seasharedcontainerallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seasharedcontainerallocation"
)

func splitUint64Ptr(v uint64) *uint64 { return &v }

func decimalRequire(s string) decimal.Decimal {
	d, err := decimal.NewFromString(s)
	if err != nil {
		panic(err)
	}
	return d
}

// splitTestEnv 保存一次集成测试运行共享的组织级主数据
type splitTestEnv struct {
	data       *Data
	orgID      uuid.UUID
	userID     uuid.UUID
	customerID uuid.UUID
	carrierID  uuid.UUID
	carrier2ID uuid.UUID
	specID     uuid.UUID
	teID       uuid.UUID
	mblID      uuid.UUID
	assetID    uuid.UUID
	ruleID     uuid.UUID
	uc         *biz.SeaOrderChangeUsecase
}

type splitTestFixture struct {
	order               *ent.Order
	link                *ent.SeaMasterBillOrderLink
	cargoItem           *ent.OrderCargoItem
	cntr1               *ent.OrderContainer
	cntr2               *ent.OrderContainer
	hbl                 *ent.SeaHouseBill
	voidedHBL           *ent.SeaHouseBill
	fee1                *ent.OrderFee
	attRef              *ent.OrderAttachment
	sharedContainerID   uuid.UUID
	sharedContainerVer  uint64
	sharedAllocationID  uuid.UUID
	sharedAllocationPkg int
	sharedAllocationWt  string
	sharedAllocationVol string
}

type splitFixtureOptions struct {
	withVoidedHBL bool
	withSharedBox bool
}

func newSplitTestEnv(t *testing.T) *splitTestEnv {
	t.Helper()
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	org, err := data.db.Organization.Create().
		SetCode("ORG-SPLIT-" + uuid.New().String()[:8]).
		SetName("拆票改配集成测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建组织失败: %v", err)
	}
	rule, err := data.db.NumberRule.Create().
		SetOrganizationID(org.ID).
		SetDocumentType(numberruleent.DocumentTypeOrder).
		SetPrefix("SE-").
		SetDateFormat(numberruleent.DateFormatYyyyMMdd).
		SetSequenceLength(5).
		SetResetPolicy(numberruleent.ResetPolicyDaily).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建订单编号规则失败: %v", err)
	}
	user, err := data.db.User.Create().
		SetUsername("split_user_" + uuid.New().String()[:8]).
		SetDisplayName("拆票测试操作员").
		SetEmail("split@example.com").
		SetPasswordHash("dummyhash").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建用户失败: %v", err)
	}
	customer, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CUST-" + uuid.New().String()[:8]).
		SetLegalName("拆票测试发货人").
		SetNormalizedName("拆票测试发货人").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建客户失败: %v", err)
	}
	carrier, err := data.db.ShippingLine.Create().
		SetOrganizationID(org.ID).
		SetScacCode("TSTL").
		SetNameZh("测试船公司").
		SetNameEn("Test Shipping Line").
		SetCountryCode("CN").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建船公司失败: %v", err)
	}
	carrier2, err := data.db.ShippingLine.Create().
		SetOrganizationID(org.ID).
		SetScacCode("TSNL").
		SetNameZh("测试船公司二号").
		SetNameEn("Test Shipping Line Two").
		SetCountryCode("CN").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建第二船公司失败: %v", err)
	}
	spec, err := data.db.MasterDataItem.Create().
		SetOrganizationID(org.ID).
		SetKind(masterdataitement.KindContainerSpec).
		SetCode("40HQ").
		SetName("40HQ超高箱").
		SetSortOrder(1).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建箱型失败: %v", err)
	}
	now := time.Now().UTC()
	te, err := data.db.SeaTransportExecution.Create().
		SetOrganizationID(org.ID).
		SetShippingLineID(carrier.ID).
		SetVesselName("MAERSK MC-KINNEY MOLLER").
		SetVoyageNo("2601W").
		SetEtd(now.Add(24 * time.Hour)).
		SetEta(now.Add(240 * time.Hour)).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建运输执行失败: %v", err)
	}
	mblNo := "MSK" + uuid.New().String()[:8]
	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetShippingLineID(carrier.ID).
		SetMasterNo(mblNo).
		SetNormalizedMasterNo(mblNo).
		SetStatus(seamasterbillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建MBL失败: %v", err)
	}
	asset, err := data.db.OrderAttachmentAsset.Create().
		SetOrganizationID(org.ID).
		SetObjectKey("orders/attachments/sample.pdf").
		SetFileName("订舱单.pdf").
		SetMimeType("application/pdf").
		SetFileSize(2048).
		SetUploadedBy(user.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建附件资产失败: %v", err)
	}
	return &splitTestEnv{
		data: data, orgID: org.ID, userID: user.ID, customerID: customer.ID,
		carrierID: carrier.ID, carrier2ID: carrier2.ID, specID: spec.ID,
		teID: te.ID, mblID: mbl.ID, assetID: asset.ID, ruleID: rule.ID,
		uc: biz.NewSeaOrderChangeUsecase(NewSeaOrderChangeRepo(data), data),
	}
}

func createTestSplitFixture(t *testing.T, env *splitTestEnv, suffix string, opts splitFixtureOptions) *splitTestFixture {
	t.Helper()
	ctx := context.Background()
	data := env.data

	order, err := data.db.Order.Create().
		SetOrganizationID(env.orgID).
		SetOrderNo("SE20260907" + suffix).
		SetCustomerID(env.customerID).
		SetBusinessType(orderent.BusinessTypeSE).
		SetCustomerReferenceNo("PO-" + suffix).
		SetTradeDirection(orderent.TradeDirectionExport).
		SetTradeTerm(orderent.TradeTermFOB).
		SetPaymentTerm(orderent.PaymentTermPREPAID).
		SetShipmentType(orderent.ShipmentTypeFCL).
		SetFlowStatus(orderent.FlowStatusDRAFT).
		SetTerminationStatus(orderent.TerminationStatusACTIVE).
		SetClosureStatus(orderent.ClosureStatusOPEN).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建订单 %s 失败: %v", suffix, err)
	}

	link, err := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(env.orgID).
		SetOrderID(order.ID).
		SetMasterBillID(env.mblID).
		SetTransportExecutionID(env.teID).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建主单关联失败: %v", err)
	}

	cargoItem, err := data.db.OrderCargoItem.Create().
		SetOrganizationID(env.orgID).
		SetOrderID(order.ID).
		SetCargoName("机械设备部件" + suffix).
		SetPackageCount(100).
		SetGrossWeightKg(2000.0).
		SetVolumeCbm(15.0).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建货物失败: %v", err)
	}

	cntr1, err := data.db.OrderContainer.Create().
		SetOrganizationID(env.orgID).
		SetOrderID(order.ID).
		SetContainerNo("MSKU100" + suffix).
		SetContainerSpecID(env.specID).
		SetPackageCount(60).
		SetGrossWeightKg(1200.0).
		SetVolumeCbm(9.0).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建箱1失败: %v", err)
	}
	cntr2, err := data.db.OrderContainer.Create().
		SetOrganizationID(env.orgID).
		SetOrderID(order.ID).
		SetContainerNo("MSKU200" + suffix).
		SetContainerSpecID(env.specID).
		SetPackageCount(40).
		SetGrossWeightKg(800.0).
		SetVolumeCbm(6.0).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建箱2失败: %v", err)
	}

	houseNo := "HBL-CUR-" + suffix
	hbl, err := data.db.SeaHouseBill.Create().
		SetOrganizationID(env.orgID).
		SetOrderID(order.ID).
		SetMasterBillID(env.mblID).
		SetHouseNo(houseNo).
		SetNormalizedHouseNo(houseNo).
		SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
		SetIssuerOrganizationID(env.orgID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建当前 HBL 失败: %v", err)
	}

	f := &splitTestFixture{
		order: order, link: link, cargoItem: cargoItem, cntr1: cntr1, cntr2: cntr2, hbl: hbl,
	}

	if opts.withVoidedHBL {
		voidedNo := "HBL-VOID-" + suffix
		voided, err := data.db.SeaHouseBill.Create().
			SetOrganizationID(env.orgID).
			SetOrderID(order.ID).
			SetMasterBillID(env.mblID).
			SetHouseNo(voidedNo).
			SetNormalizedHouseNo(voidedNo).
			SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
			SetIssuerOrganizationID(env.orgID).
			SetStatus(seahousebillent.StatusVOIDED).
			SetVersion(3).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建历史 VOIDED HBL 失败: %v", err)
		}
		f.voidedHBL = voided
	}

	if opts.withSharedBox {
		sc, err := data.db.SeaSharedContainer.Create().
			SetOrganizationID(env.orgID).
			SetTransportExecutionID(env.teID).
			SetContainerNo("SHCU-SPLIT-" + suffix).
			SetContainerSpecID(env.specID).
			SetPackageCount(100).
			SetGrossWeightKg("2000.000").
			SetVolumeCbm("15.000000").
			SetStatus(seasharedcontainerent.StatusDRAFT).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建共享箱失败: %v", err)
		}
		alloc, err := data.db.SeaSharedContainerAllocation.Create().
			SetOrganizationID(env.orgID).
			SetSharedContainerID(sc.ID).
			SetOrderID(order.ID).
			SetHouseBillID(hbl.ID).
			SetCargoItemID(cargoItem.ID).
			SetPackageCount(100).
			SetGrossWeightKg("2000.000").
			SetVolumeCbm("15.000000").
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建共享箱分配失败: %v", err)
		}
		f.sharedContainerID = sc.ID
		f.sharedContainerVer = sc.Version
		f.sharedAllocationID = alloc.ID
		f.sharedAllocationPkg = alloc.PackageCount
		f.sharedAllocationWt = alloc.GrossWeightKg
		f.sharedAllocationVol = alloc.VolumeCbm
	}

	fee1, err := data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("fee-" + suffix).
		SetDirection(orderfeeent.DirectionRECEIVABLE).
		SetStatus(orderfeeent.StatusDRAFT).
		SetFeeCode("OFT").
		SetFeeName("海运基本费").
		SetSettlementPartyID(env.customerID).
		SetBillingUnit("BL").
		SetQuantity("1.0000").
		SetUnitPrice("1500.0000").
		SetTotalAmount("1500.00000000").
		SetTaxInclusive(true).
		SetNetAmount("1500.00000000").
		SetTaxAmount("0.00000000").
		SetCurrency("USD").
		SetExchangeRate("7.10000000").
		SetExchangeRateSource(orderfeeent.ExchangeRateSourceMANUAL).
		SetExchangeRateDate("2026-09-07").
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("10650.00000000").
		SetExpenseDate("2026-09-07").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建费用失败: %v", err)
	}
	f.fee1 = fee1

	attRef, err := data.db.OrderAttachment.Create().
		SetOrderID(order.ID).
		SetAssetID(env.assetID).
		SetDocType("BOOKING_NOTE").
		SetIdempotencyKey("att-" + suffix).
		SetCreatedBy(env.userID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建附件引用失败: %v", err)
	}
	f.attRef = attRef
	return f
}

func (f *splitTestFixture) expectedVersions() *biz.SeaOrderSplitExpectedVersions {
	currentHBLVersion := f.hbl.Version
	ev := &biz.SeaOrderSplitExpectedVersions{
		OrderVersion:      f.order.Version,
		LinkVersion:       f.link.Version,
		CurrentHBLVersion: &currentHBLVersion,
		CargoItemVersions: map[uuid.UUID]uint64{f.cargoItem.ID: f.cargoItem.Version},
		ContainerVersions: map[uuid.UUID]uint64{f.cntr1.ID: f.cntr1.Version, f.cntr2.ID: f.cntr2.Version},
		FeeVersions:       map[uuid.UUID]uint64{f.fee1.ID: f.fee1.Version},
		AttachmentReferenceFingerprint: biz.ComputeAttachmentFingerprint([]*biz.SeaOrderSplitAttachmentItem{
			{ID: f.attRef.ID, AssetID: f.attRef.AssetID, DocType: f.attRef.DocType},
		}),
	}
	if f.sharedContainerID != uuid.Nil {
		ev.SharedContainerVersions = map[uuid.UUID]uint64{f.sharedContainerID: f.sharedContainerVer}
	}
	return ev
}

// standardSplitInput 构造 60/40 拆分：原票保留 HBL 与箱1，新票新建 HBL 与箱2
func (f *splitTestFixture) standardSplitInput(idempotencyKey, fingerprint string, opts splitFixtureOptions) *biz.SeaOrderSplitInput {
	newHouseNo := "HBL-NEW-" + uuid.NewString()[:8]
	originalResult := &biz.SeaOrderSplitResultInput{
		ClientResultKey: "res-origin",
		ResultRole:      biz.ResultRoleOriginal,
		ClientTargetKey: "target-current",
		DraftFeeIDs:     []uuid.UUID{f.fee1.ID},
		CargoAllocations: []*biz.SeaOrderSplitCargoAllocationInput{
			{CargoItemID: f.cargoItem.ID, PackageCount: 60, GrossWeightKg: decimalRequire("1200"), VolumeCbm: decimalRequire("9")},
		},
		ContainerIDs: []uuid.UUID{f.cntr1.ID},
	}
	createdResult := &biz.SeaOrderSplitResultInput{
		ClientResultKey: "res-new-1",
		ResultRole:      biz.ResultRoleCreated,
		ClientTargetKey: "target-current",
		HouseBill:       &biz.SeaOrderSplitHouseBillInput{HouseNo: newHouseNo, IssuerSource: string(biz.SeaHouseBillIssuerSourceSelfOrganization)},
		CargoAllocations: []*biz.SeaOrderSplitCargoAllocationInput{
			{CargoItemID: f.cargoItem.ID, PackageCount: 40, GrossWeightKg: decimalRequire("800"), VolumeCbm: decimalRequire("6")},
		},
		ContainerIDs:           []uuid.UUID{f.cntr2.ID},
		AttachmentReferenceIDs: []uuid.UUID{f.attRef.ID},
	}
	if opts.withSharedBox {
		originalResult.SharedContainerAllocations = []*biz.SeaOrderSplitSharedContainerAllocationInput{
			{AllocationID: f.sharedAllocationID, PackageCount: int32(f.sharedAllocationPkg) - 40, GrossWeightKg: decimalRequire("1200"), VolumeCbm: decimalRequire("9")},
		}
		createdResult.SharedContainerAllocations = []*biz.SeaOrderSplitSharedContainerAllocationInput{
			{AllocationID: f.sharedAllocationID, PackageCount: 40, GrossWeightKg: decimalRequire("800"), VolumeCbm: decimalRequire("6")},
		}
	}
	return &biz.SeaOrderSplitInput{
		OrderID:            f.order.ID,
		IdempotencyKey:     idempotencyKey,
		RequestFingerprint: fingerprint,
		Targets: []*biz.SeaOrderSplitTargetInput{
			{ClientTargetKey: "target-current", TargetType: biz.SplitTargetTypeCurrent},
		},
		Results:          []*biz.SeaOrderSplitResultInput{originalResult, createdResult},
		ExpectedVersions: f.expectedVersions(),
	}
}

func TestSeaOrderSplitAndReassignment_PostgresIntegration(t *testing.T) {
	env := newSplitTestEnv(t)
	ctx := context.Background()

	// A. 单当前 HBL 的 HOUSE 订单可拆票：守恒、归属、幂等
	t.Run("单当前HBL拆票守恒与幂等", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "001", splitFixtureOptions{})

		actions, err := env.uc.GetChangeActions(ctx, env.orgID, f.order.ID)
		if err != nil || !actions.CanSplit {
			t.Fatalf("唯一当前 HBL 的 HOUSE 订单应可拆票: actions=%+v err=%v", actions, err)
		}

		splitInput := f.standardSplitInput("split-test-idemp-001", "fp-split-001", splitFixtureOptions{})

		preview, err := env.uc.PreviewSplit(ctx, env.orgID, splitInput)
		if err != nil || !preview.IsValid || !preview.ConservationPassed {
			t.Fatalf("拆票预览应通过: preview=%+v err=%v", preview, err)
		}

		splitEvent, splitErr := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, splitInput)
		if splitErr != nil {
			t.Fatalf("ExecuteSplit 失败: %v", splitErr)
		}
		if len(splitEvent.Results) != 2 {
			t.Fatalf("拆票事件返回结果异常: %+v", splitEvent)
		}

		var createdOrderID uuid.UUID
		for _, r := range splitEvent.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
			}
		}
		if createdOrderID == uuid.Nil {
			t.Fatal("未生成有效的新订单")
		}

		// 原订单保留唯一当前 HBL，新订单各自创建一张 HBL
		originHBLs, _ := env.data.db.SeaHouseBill.Query().Where(seahousebillent.OrderIDEQ(f.order.ID), seahousebillent.StatusNEQ(seahousebillent.StatusVOIDED)).All(ctx)
		if len(originHBLs) != 1 || originHBLs[0].ID != f.hbl.ID {
			t.Fatalf("原订单应保留唯一当前 HBL: %+v", originHBLs)
		}
		createdHBLs, _ := env.data.db.SeaHouseBill.Query().Where(seahousebillent.OrderIDEQ(createdOrderID)).All(ctx)
		if len(createdHBLs) != 1 || createdHBLs[0].HouseNo != splitInput.Results[1].HouseBill.HouseNo || createdHBLs[0].Version != 1 {
			t.Fatalf("新订单应恰有一张新建 HBL: %+v", createdHBLs)
		}

		// 货物守恒
		originCargo, _ := env.data.db.OrderCargoItem.Query().Where(ordercargoitement.OrderIDEQ(f.order.ID)).Only(ctx)
		if originCargo.PackageCount != 60 || originCargo.GrossWeightKg != 1200.0 || originCargo.VolumeCbm != 9.0 {
			t.Errorf("源订单货物数量守恒异常: %+v", originCargo)
		}
		newCargo, _ := env.data.db.OrderCargoItem.Query().Where(ordercargoitement.OrderIDEQ(createdOrderID)).Only(ctx)
		if newCargo.PackageCount != 40 || newCargo.GrossWeightKg != 800.0 || newCargo.VolumeCbm != 6.0 {
			t.Errorf("新订单货物数量守恒异常: %+v", newCargo)
		}

		// 箱归属
		cntr1Check, _ := env.data.db.OrderContainer.Get(ctx, f.cntr1.ID)
		cntr2Check, _ := env.data.db.OrderContainer.Get(ctx, f.cntr2.ID)
		if cntr1Check.OrderID != f.order.ID || cntr2Check.OrderID != createdOrderID {
			t.Errorf("集装箱归属异常: cntr1=%v cntr2=%v", cntr1Check.OrderID, cntr2Check.OrderID)
		}

		// 幂等同键同指纹重试
		sameEvt, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, splitInput)
		if err != nil || sameEvt.ID != splitEvent.ID {
			t.Fatalf("幂等重试应返回同一事件: err=%v", err)
		}
		// 幂等同键异指纹拦截
		conflictInput := *splitInput
		conflictInput.RequestFingerprint = "diff-fp-split-999"
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, &conflictInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_IDEMPOTENCY_CONFLICT" {
			t.Fatalf("不同指纹重试应拦截，实际: %v", err)
		}
	})

	// B. 历史 VOIDED HBL 不参与当前结构门禁
	t.Run("历史VOIDED HBL不阻断拆票", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "002", splitFixtureOptions{withVoidedHBL: true})
		if f.voidedHBL == nil {
			t.Fatal("fixture 缺少历史 VOIDED HBL")
		}

		actions, err := env.uc.GetChangeActions(ctx, env.orgID, f.order.ID)
		if err != nil || !actions.CanSplit {
			t.Fatalf("历史 VOIDED HBL 不应阻断拆票: actions=%+v err=%v", actions, err)
		}

		splitInput := f.standardSplitInput("split-voided-hbl-002", "fp-voided-hbl-002", splitFixtureOptions{})
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, splitInput); err != nil {
			t.Fatalf("存在历史 VOIDED HBL 时拆票应成功: %v", err)
		}

		voidedCheck, _ := env.data.db.SeaHouseBill.Get(ctx, f.voidedHBL.ID)
		if voidedCheck.Status != seahousebillent.StatusVOIDED || voidedCheck.OrderID != f.order.ID {
			t.Fatalf("历史 VOIDED HBL 不应被拆票改写: %+v", voidedCheck)
		}
	})

	// C. 事务中途回滚
	t.Run("拆票中途错误全事务回滚", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "003", splitFixtureOptions{})

		orderCountBefore, _ := env.data.db.Order.Query().Count(ctx)
		sourceBefore, _ := env.data.db.Order.Get(ctx, f.order.ID)
		seqBefore, _ := env.data.db.NumberSequence.Query().Where(numbersequenceent.RuleIDEQ(env.ruleID)).Only(ctx)
		var seqValueBefore int64
		if seqBefore != nil {
			seqValueBefore = seqBefore.CurrentValue
		}

		rollbackInput := f.standardSplitInput("split-mid-rollback-idemp", "fp-mid-rollback", splitFixtureOptions{})
		rollbackInput.ExpectedVersions.LinkVersion = 999999

		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, rollbackInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_VERSION_CONFLICT" {
			t.Fatalf("LinkVersion 冲突应返回版本冲突，实际: %v", err)
		}

		orderCountAfter, _ := env.data.db.Order.Query().Count(ctx)
		if orderCountAfter != orderCountBefore {
			t.Errorf("事务回滚失败，订单总数变化: before=%d after=%d", orderCountBefore, orderCountAfter)
		}
		sourceAfter, _ := env.data.db.Order.Get(ctx, f.order.ID)
		if sourceAfter.Version != sourceBefore.Version {
			t.Errorf("源订单版本号不应递增: before=%d after=%d", sourceBefore.Version, sourceAfter.Version)
		}
		eventExists, _ := env.data.db.SeaOrderSplitEvent.Query().Where(seaorderspliteventent.IdempotencyKeyEQ("split-mid-rollback-idemp")).Exist(ctx)
		if eventExists {
			t.Errorf("事务回滚失败，拆票事件记录不应存在")
		}
		seqAfter, _ := env.data.db.NumberSequence.Query().Where(numbersequenceent.RuleIDEQ(env.ruleID)).Only(ctx)
		var seqValueAfter int64
		if seqAfter != nil {
			seqValueAfter = seqAfter.CurrentValue
		}
		if seqValueAfter != seqValueBefore {
			t.Errorf("号码序列未回滚: before=%d after=%d", seqValueBefore, seqValueAfter)
		}
	})

	// D. DIRECT 拒绝拆票与整票改配
	t.Run("DIRECT拆票阻断与整票改配成功", func(t *testing.T) {
		directOrder, err := env.data.db.Order.Create().
			SetOrganizationID(env.orgID).
			SetOrderNo("SE20260907088").
			SetCustomerID(env.customerID).
			SetBusinessType(orderent.BusinessTypeSE).
			SetTradeDirection(orderent.TradeDirectionExport).
			SetTradeTerm(orderent.TradeTermFOB).
			SetPaymentTerm(orderent.PaymentTermPREPAID).
			SetShipmentType(orderent.ShipmentTypeFCL).
			SetFlowStatus(orderent.FlowStatusDRAFT).
			SetTerminationStatus(orderent.TerminationStatusACTIVE).
			SetClosureStatus(orderent.ClosureStatusOPEN).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建DIRECT订单失败: %v", err)
		}
		directLink, err := env.data.db.SeaMasterBillOrderLink.Create().
			SetOrganizationID(env.orgID).
			SetOrderID(directOrder.ID).
			SetMasterBillID(env.mblID).
			SetTransportExecutionID(env.teID).
			SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureDIRECT).
			SetStatus(seamasterbillorderlinkent.StatusACTIVE).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建DIRECT关联失败: %v", err)
		}

		directSplit := &biz.SeaOrderSplitInput{
			OrderID:            directOrder.ID,
			IdempotencyKey:     "direct-split-fail",
			RequestFingerprint: "fp-direct",
			Targets:            []*biz.SeaOrderSplitTargetInput{{ClientTargetKey: "t1", TargetType: biz.SplitTargetTypeCurrent}},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "r1", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "t1"},
				{ClientResultKey: "r2", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "t1"},
			},
			ExpectedVersions: &biz.SeaOrderSplitExpectedVersions{
				OrderVersion: 1, LinkVersion: 1,
				CurrentHBLVersion: splitUint64Ptr(1),
			},
		}
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, directSplit); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_BLOCKED" {
			t.Fatalf("DIRECT 拆票应被阻断，实际: %v", err)
		}

		confirmation := &biz.SeaExternalConfirmation{ConfirmedByParty: "船代窗口", ConfirmedAt: time.Now().UTC(), ConfirmationNote: "船代确认改配"}
		reassignInput := &biz.SeaOrderReassignmentInput{
			OrderID:            directOrder.ID,
			IdempotencyKey:     "reas-direct-001",
			RequestFingerprint: "fp-reas-001",
			Reason:             "船期延误客户申请改配至新母单",
			ResponsibilityType: biz.ResponsibilityTypeCarrier,
			Confirmation:       confirmation,
			Target: &biz.SeaOrderReassignmentTargetInput{
				TargetType:     biz.SplitTargetTypeNew,
				MasterNo:       "NEWMBL" + uuid.NewString()[:6],
				ShippingLineID: &env.carrier2ID,
				VesselName:     "COSCO SHIPPING GEMINI",
				VoyageNo:       "088E",
			},
			ExpectedOrderVersion: directOrder.Version,
			ExpectedLinkVersion:  directLink.Version,
		}
		reasEvt, err := env.uc.ExecuteReassignment(ctx, env.orgID, env.userID, reassignInput)
		if err != nil {
			t.Fatalf("DIRECT 整票改配失败: %v", err)
		}
		oldLinkAfter, _ := env.data.db.SeaMasterBillOrderLink.Get(ctx, directLink.ID)
		if oldLinkAfter.Status != seamasterbillorderlinkent.StatusENDED {
			t.Errorf("旧主单关联应为 ENDED, 实际: %v", oldLinkAfter.Status)
		}
		newLinkAfter, err := env.data.db.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.OrderIDEQ(directOrder.ID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).Only(ctx)
		if err != nil || newLinkAfter.MasterBillID == env.mblID {
			t.Fatalf("新活动关联应指向新 MBL: %+v err=%v", newLinkAfter, err)
		}
		if reasEvt.Confirmation == nil || reasEvt.Confirmation.ConfirmedByParty != confirmation.ConfirmedByParty {
			t.Errorf("改配事件应保存外部确认: %+v", reasEvt.Confirmation)
		}

		sameReasEvt, err := env.uc.ExecuteReassignment(ctx, env.orgID, env.userID, reassignInput)
		if err != nil || sameReasEvt.ID != reasEvt.ID {
			t.Fatalf("改配幂等执行失败: err=%v", err)
		}
		diffFpInput := *reassignInput
		diffFpInput.RequestFingerprint = "diff-fp-reas-999"
		if _, err = env.uc.ExecuteReassignment(ctx, env.orgID, env.userID, &diffFpInput); kratoserrors.Reason(err) != "SEA_ORDER_REASSIGNMENT_IDEMPOTENCY_CONFLICT" {
			t.Fatalf("改配异指纹应冲突报错，实际: %v", err)
		}
	})

	// E. 货物与共享箱两套全局守恒通过但结果票内部超分必须拒绝
	t.Run("共享箱交叉守恒拒绝与零写入", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "004", splitFixtureOptions{withSharedBox: true})

		input := f.standardSplitInput("split-cross-conservation-004", "fp-cross-004", splitFixtureOptions{withSharedBox: true})
		// 货物拆分 60/40，共享箱分配拆成 30/70：全局守恒各自成立，但新票共享箱分配(70)超过其货物(40)
		input.Results[0].SharedContainerAllocations[0].PackageCount = 30
		input.Results[0].SharedContainerAllocations[0].GrossWeightKg = decimalRequire("600")
		input.Results[0].SharedContainerAllocations[0].VolumeCbm = decimalRequire("4.5")
		input.Results[1].SharedContainerAllocations[0].PackageCount = 70
		input.Results[1].SharedContainerAllocations[0].GrossWeightKg = decimalRequire("1400")
		input.Results[1].SharedContainerAllocations[0].VolumeCbm = decimalRequire("10.5")

		preview, err := env.uc.PreviewSplit(ctx, env.orgID, input)
		if err != nil {
			t.Fatalf("预览不应直接报错: %v", err)
		}
		foundCross := false
		for _, ve := range preview.ValidationErrors {
			if ve.Reason == "SHARED_ALLOCATION_EXCEEDS_RESULT_CARGO" {
				foundCross = true
			}
		}
		if preview.IsValid || preview.ConservationPassed || !foundCross {
			t.Fatalf("预览应拒绝结果票内部超分: valid=%v conservation=%v errors=%+v", preview.IsValid, preview.ConservationPassed, preview.ValidationErrors)
		}

		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, input); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_CONSERVATION_FAILED" {
			t.Fatalf("执行应拒绝交叉超分，实际: %v", err)
		}
		// 失败零写入：共享箱分配与版本不变、订单数不变
		allocAfter, err := env.data.db.SeaSharedContainerAllocation.Get(ctx, f.sharedAllocationID)
		if err != nil || allocAfter.PackageCount != f.sharedAllocationPkg || allocAfter.Version != 1 {
			t.Fatalf("失败事务不应改写共享箱分配: %+v err=%v", allocAfter, err)
		}
		scAfter, _ := env.data.db.SeaSharedContainer.Get(ctx, f.sharedContainerID)
		if scAfter.Version != f.sharedContainerVer {
			t.Fatalf("失败事务不应递增共享箱版本: %d", scAfter.Version)
		}
		orderCount, _ := env.data.db.Order.Query().Where(orderent.IDEQ(f.order.ID)).Count(ctx)
		if orderCount != 1 {
			t.Fatalf("失败事务不应产生新订单: %d", orderCount)
		}
	})

	// F. 共享箱只拆 Allocation 不复制物理箱，聚合版本闭环
	t.Run("共享箱只拆Allocation且版本递增", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "005", splitFixtureOptions{withSharedBox: true})

		input := f.standardSplitInput("split-shared-box-005", "fp-shared-box-005", splitFixtureOptions{withSharedBox: true})
		splitEvt, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, input)
		if err != nil {
			t.Fatalf("共享箱拆票失败: %v", err)
		}

		var createdOrderID uuid.UUID
		for _, r := range splitEvt.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
			}
		}
		if createdOrderID == uuid.Nil {
			t.Fatal("未找到拆票创建的新订单")
		}

		// 物理箱不复制：本共享箱行数不变
		thisScCount, _ := env.data.db.SeaSharedContainer.Query().Where(seasharedcontainerent.IDEQ(f.sharedContainerID)).Count(ctx)
		if thisScCount != 1 {
			t.Fatalf("拆票不得复制共享物理箱: %d", thisScCount)
		}
		// 原分配更新为 60，新分配指向新订单新货物
		originAlloc, err := env.data.db.SeaSharedContainerAllocation.Query().Where(seasharedcontainerallocationent.IDEQ(f.sharedAllocationID)).Only(ctx)
		if err != nil {
			t.Fatalf("原共享箱分配应保留并更新: %v", err)
		}
		if originAlloc.PackageCount != 60 || originAlloc.OrderID != f.order.ID {
			t.Fatalf("原共享箱分配应更新为 60 件并留在原订单: %+v", originAlloc)
		}
		newCargo, _ := env.data.db.OrderCargoItem.Query().Where(ordercargoitement.OrderIDEQ(createdOrderID)).Only(ctx)
		newAlloc, err := env.data.db.SeaSharedContainerAllocation.Query().Where(seasharedcontainerallocationent.OrderIDEQ(createdOrderID)).Only(ctx)
		if err != nil || newAlloc.CargoItemID != newCargo.ID || newAlloc.SharedContainerID != f.sharedContainerID || newAlloc.PackageCount != 40 {
			t.Fatalf("新共享箱分配应指向新货物且同属原物理箱: %+v err=%v", newAlloc, err)
		}
		// 聚合版本恰好递增一次
		scAfter, _ := env.data.db.SeaSharedContainer.Get(ctx, f.sharedContainerID)
		if scAfter.Version != f.sharedContainerVer+1 {
			t.Fatalf("受影响共享箱版本应递增一次: got %d want %d", scAfter.Version, f.sharedContainerVer+1)
		}

		// 工作台旧版本保存返回 409
		sharedUC := biz.NewSeaSharedContainerUsecase(NewSeaSharedContainerRepo(env.data))
		staleInputs := []*biz.SeaSharedContainerAllocationInput{
			{OrderID: f.order.ID, HouseBillID: f.hbl.ID, CargoItemID: originAlloc.CargoItemID, PackageCount: 60, GrossWeightKg: decimalRequire("1200"), VolumeCbm: decimalRequire("9"), ExpectedOrderVersion: f.order.Version + 1, ExpectedLinkVersion: 1, ExpectedHouseBillVersion: 1, ExpectedCargoItemVersion: 2},
		}
		if _, err := sharedUC.SaveDraft(ctx, env.orgID, env.userID, f.sharedContainerID, f.sharedContainerVer, staleInputs); kratoserrors.Reason(err) != "SEA_SHARED_CONTAINER_CONFLICT" {
			t.Fatalf("拆票后工作台旧版本保存应返回 409，实际: %v", err)
		}
	})

	// G. SharedContainerVersions 必填与版本强校验
	t.Run("SharedContainerVersions必填校验", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "006", splitFixtureOptions{withSharedBox: true})

		// 缺 Map
		noMapInput := f.standardSplitInput("split-sc-nomap-006", "fp-sc-nomap-006", splitFixtureOptions{withSharedBox: true})
		noMapInput.ExpectedVersions.SharedContainerVersions = nil
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, noMapInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_INVALID_ARGUMENT" {
			t.Fatalf("缺少共享箱版本 Map 应返回参数错误，实际: %v", err)
		}
		// 缺 key
		missingKeyInput := f.standardSplitInput("split-sc-missing-key-006", "fp-sc-missing-key-006", splitFixtureOptions{withSharedBox: true})
		missingKeyInput.ExpectedVersions.SharedContainerVersions = map[uuid.UUID]uint64{uuid.New(): 1}
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, missingKeyInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_INVALID_ARGUMENT" {
			t.Fatalf("缺少共享箱版本 key 应返回参数错误，实际: %v", err)
		}
		// 版本 0
		zeroInput := f.standardSplitInput("split-sc-zero-006", "fp-sc-zero-006", splitFixtureOptions{withSharedBox: true})
		zeroInput.ExpectedVersions.SharedContainerVersions = map[uuid.UUID]uint64{f.sharedContainerID: 0}
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, zeroInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_INVALID_ARGUMENT" {
			t.Fatalf("共享箱版本为 0 应返回参数错误，实际: %v", err)
		}
		// 版本不一致
		wrongInput := f.standardSplitInput("split-sc-wrong-006", "fp-sc-wrong-006", splitFixtureOptions{withSharedBox: true})
		wrongInput.ExpectedVersions.SharedContainerVersions = map[uuid.UUID]uint64{f.sharedContainerID: 99}
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, wrongInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_VERSION_CONFLICT" {
			t.Fatalf("共享箱版本不一致应返回 409，实际: %v", err)
		}
		// 全部失败后共享箱未被改写
		scCheck, _ := env.data.db.SeaSharedContainer.Get(ctx, f.sharedContainerID)
		if scCheck.Version != f.sharedContainerVer {
			t.Fatalf("失败事务不应递增共享箱版本: %d", scCheck.Version)
		}
	})

	// H. 独占箱不得跨结果票
	t.Run("独占箱跨结果拒绝", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "007", splitFixtureOptions{})

		input := f.standardSplitInput("split-container-cross-007", "fp-container-cross-007", splitFixtureOptions{})
		input.Results[1].ContainerIDs = []uuid.UUID{f.cntr1.ID, f.cntr2.ID}

		preview, err := env.uc.PreviewSplit(ctx, env.orgID, input)
		if err != nil {
			t.Fatalf("预览不应直接报错: %v", err)
		}
		foundCross := false
		for _, ve := range preview.ValidationErrors {
			if ve.Reason == "CONTAINER_CROSSES_RESULTS" {
				foundCross = true
			}
		}
		if preview.IsValid || !foundCross {
			t.Fatalf("预览应拒绝独占箱跨结果: %+v", preview.ValidationErrors)
		}
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, input); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_ENTITY_CROSSES_RESULTS" {
			t.Fatalf("执行应拒绝独占箱跨结果，实际: %v", err)
		}
	})

	// I. 并发拆票竞争仅一成功
	t.Run("两并发拆票竞争仅一成功", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "008", splitFixtureOptions{})

		inputA := f.standardSplitInput("conc-split-a", "fp-conc-a", splitFixtureOptions{})
		inputB := f.standardSplitInput("conc-split-b", "fp-conc-b", splitFixtureOptions{})

		resultsChan := make(chan error, 2)
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, inputA)
			resultsChan <- err
		}()
		go func() {
			defer wg.Done()
			_, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, inputB)
			resultsChan <- err
		}()
		wg.Wait()
		close(resultsChan)

		var successCount, conflictCount int
		for err := range resultsChan {
			if err == nil {
				successCount++
			} else if kratoserrors.Reason(err) == "SEA_ORDER_SPLIT_VERSION_CONFLICT" {
				conflictCount++
			} else {
				t.Errorf("并发拆票返回非预期错误: %v", err)
			}
		}
		if successCount != 1 || conflictCount != 1 {
			t.Errorf("并发拆票竞争结果异常: success=%d conflict=%d", successCount, conflictCount)
		}
	})

	// J. 注入审计失败全事务回滚
	t.Run("注入审计失败触发全事务回滚", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "009", splitFixtureOptions{})

		orderCountBefore, _ := env.data.db.Order.Query().Count(ctx)
		splitInput := f.standardSplitInput("split-audit-fail-009", "fp-audit-fail-009", splitFixtureOptions{})

		injectedCtx := biz.WithInjectedAuditFailure(ctx)
		if _, err := env.uc.ExecuteSplit(injectedCtx, env.orgID, env.userID, splitInput); err == nil {
			t.Fatal("注入审计失败时必须返回错误")
		}
		orderCountAfter, _ := env.data.db.Order.Query().Count(ctx)
		if orderCountAfter != orderCountBefore {
			t.Errorf("审计失败回滚后订单总数不应增加: before=%d after=%d", orderCountBefore, orderCountAfter)
		}
		eventExists, _ := env.data.db.SeaOrderSplitEvent.Query().Where(seaorderspliteventent.IdempotencyKeyEQ("split-audit-fail-009")).Exist(ctx)
		if eventExists {
			t.Errorf("审计失败回滚后不应残留拆票事件")
		}
	})

	// K. 附件指纹冲突拦截
	t.Run("附件指纹冲突拦截", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "010", splitFixtureOptions{})

		splitInput := f.standardSplitInput("split-att-conflict-010", "fp-att-conflict-010", splitFixtureOptions{})
		splitInput.ExpectedVersions.AttachmentReferenceFingerprint = "wrong-mismatch-fingerprint"
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, splitInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_VERSION_CONFLICT" {
			t.Fatalf("附件指纹冲突应返回版本冲突，实际: %v", err)
		}
	})

	// L. 拆票换入新 MBL 目标：新 TE/MBL 创建、新票 HBL 迁移、内嵌改配事件归属新票
	t.Run("拆票结果换入新MBL目标", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "011", splitFixtureOptions{})

		splitInput := f.standardSplitInput("split-target-new-mbl-011", "fp-target-new-mbl-011", splitFixtureOptions{})
		splitInput.Targets = []*biz.SeaOrderSplitTargetInput{
			{ClientTargetKey: "target-current", TargetType: biz.SplitTargetTypeCurrent},
			{
				ClientTargetKey: "target-new", TargetType: biz.SplitTargetTypeNew,
				MasterNo: "NEWPLIT" + uuid.NewString()[:6], ShippingLineID: &env.carrierID,
				VesselName: "EVER GIVEN", VoyageNo: "001W",
			},
		}
		splitInput.Results[1].ClientTargetKey = "target-new"
		splitInput.Confirmation = &biz.SeaExternalConfirmation{ConfirmedByParty: "船代窗口", ConfirmedAt: time.Now().UTC(), ConfirmationNote: "船代确认拆票改配"}

		splitEvt, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, splitInput)
		if err != nil {
			t.Fatalf("ExecuteSplit 新MBL测试失败: %v", err)
		}
		var createdOrderID uuid.UUID
		for _, r := range splitEvt.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
			}
		}
		newLink, err := env.data.db.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.OrderIDEQ(createdOrderID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).Only(ctx)
		if err != nil || newLink.MasterBillID == env.mblID {
			t.Fatalf("新票 Link 应指向新 MBL: %+v err=%v", newLink, err)
		}
		createdHBL, _ := env.data.db.SeaHouseBill.Query().Where(seahousebillent.OrderIDEQ(createdOrderID)).Only(ctx)
		if createdHBL.MasterBillID != newLink.MasterBillID {
			t.Fatalf("新票 HBL.MasterBillID 未迁移至新 MBL: %+v", createdHBL)
		}
		createdTE, _ := env.data.db.SeaTransportExecution.Get(ctx, newLink.TransportExecutionID)
		createdMBL, _ := env.data.db.SeaMasterBill.Get(ctx, newLink.MasterBillID)
		if createdTE.ShippingLineID != env.carrierID || createdMBL.ShippingLineID != env.carrierID {
			t.Fatalf("新 MBL 与新 TE 船公司应一致: te=%+v mbl=%+v", createdTE, createdMBL)
		}
		if len(splitEvt.ReassignmentEventIDs) != 1 {
			t.Fatalf("新票改配应生成 1 条内嵌改配事件: %+v", splitEvt.ReassignmentEventIDs)
		}
		reassignEvt, err := env.data.db.SeaOrderReassignmentEvent.Get(ctx, splitEvt.ReassignmentEventIDs[0])
		if err != nil || reassignEvt.OrderID != createdOrderID {
			t.Fatalf("内嵌改配事件必须属于新票: %+v err=%v", reassignEvt, err)
		}
		// 原票保持当前 MBL 与 HBL
		originLink, _ := env.data.db.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.OrderIDEQ(f.order.ID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).Only(ctx)
		if originLink.ID != f.link.ID || originLink.MasterBillID != env.mblID {
			t.Fatalf("原票应保持原 Link 与 MBL: %+v", originLink)
		}
		originHBL, _ := env.data.db.SeaHouseBill.Get(ctx, f.hbl.ID)
		if originHBL.MasterBillID != env.mblID || originHBL.OrderID != f.order.ID {
			t.Fatalf("原票 HBL 不应被新目标改写: %+v", originHBL)
		}
	})

		// L2. 独占箱版本 Map 必须完整且非零，过期版本返回 409
		t.Run("独占箱版本缺失或过期拒绝", func(t *testing.T) {
			f := createTestSplitFixture(t, env, "016", splitFixtureOptions{})

			// 缺 Map
			noMapInput := f.standardSplitInput("split-cntr-nomap-016", "fp-cntr-nomap-016", splitFixtureOptions{})
			noMapInput.ExpectedVersions.ContainerVersions = nil
			if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, noMapInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_INVALID_ARGUMENT" {
				t.Fatalf("缺少独占箱版本 Map 应返回参数错误，实际: %v", err)
			}
			// 缺 key
			missingKeyInput := f.standardSplitInput("split-cntr-missing-016", "fp-cntr-missing-016", splitFixtureOptions{})
			missingKeyInput.ExpectedVersions.ContainerVersions = map[uuid.UUID]uint64{f.cntr1.ID: f.cntr1.Version}
			if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, missingKeyInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_INVALID_ARGUMENT" {
				t.Fatalf("缺少独占箱版本 key 应返回参数错误，实际: %v", err)
			}
			// 版本 0
			zeroInput := f.standardSplitInput("split-cntr-zero-016", "fp-cntr-zero-016", splitFixtureOptions{})
			zeroInput.ExpectedVersions.ContainerVersions = map[uuid.UUID]uint64{f.cntr1.ID: 0, f.cntr2.ID: f.cntr2.Version}
			if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, zeroInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_INVALID_ARGUMENT" {
				t.Fatalf("独占箱版本为 0 应返回参数错误，实际: %v", err)
			}
			// 携带过期版本（模拟读取拆票上下文后其他用户修改了独占箱）
			staleInput := f.standardSplitInput("split-cntr-stale-016", "fp-cntr-stale-016", splitFixtureOptions{})
			modifiedCntr, err := env.data.db.OrderContainer.UpdateOneID(f.cntr2.ID).
				SetContainerNo("MSKU200-CHANGED").
				SetVersion(f.cntr2.Version + 1).
				Save(ctx)
			if err != nil {
				t.Fatalf("并发修改独占箱失败: %v", err)
			}
			if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, staleInput); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_VERSION_CONFLICT" {
				t.Fatalf("独占箱版本过期应返回 409，实际: %v", err)
			}
			// 失败零写入：订单未新增、HBL 未变、被并发修改的箱保持修改后的状态
			orderCount, _ := env.data.db.Order.Query().Where(orderent.IDEQ(f.order.ID)).Count(ctx)
			if orderCount != 1 {
				t.Fatalf("失败事务不应产生新订单: %d", orderCount)
			}
			cntrCheck, _ := env.data.db.OrderContainer.Get(ctx, f.cntr2.ID)
			if cntrCheck.ContainerNo != modifiedCntr.ContainerNo || cntrCheck.OrderID != f.order.ID {
				t.Fatalf("失败事务不应改写并发修改后的独占箱: %+v", cntrCheck)
			}
			eventExists, _ := env.data.db.SeaOrderSplitEvent.Query().Where(seaorderspliteventent.IdempotencyKeyEQ("split-cntr-stale-016")).Exist(ctx)
			if eventExists {
				t.Fatal("失败事务不应残留拆票事件")
			}
		})

		// M. 候选 MBL 目标输入不符阻断、一致则成功
		t.Run("候选母单目标一致允许与输入不符阻断", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "012", splitFixtureOptions{})

		candidateTE, err := env.data.db.SeaTransportExecution.Create().
			SetOrganizationID(env.orgID).
			SetShippingLineID(env.carrierID).
			SetVesselName("PACIFIC GLORY").
			SetVoyageNo("2026E").
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建候选TE失败: %v", err)
		}
		candidateMBLNo := "CANDMBL" + uuid.NewString()[:6]
		candidateMBL, err := env.data.db.SeaMasterBill.Create().
			SetOrganizationID(env.orgID).
			SetMasterNo(candidateMBLNo).
			SetNormalizedMasterNo(candidateMBLNo).
			SetShippingLineID(env.carrierID).
			SetStatus(seamasterbillent.StatusDRAFT).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建候选MBL失败: %v", err)
		}
		carrierOrder, err := env.data.db.Order.Create().
			SetOrganizationID(env.orgID).
			SetOrderNo("SE-CAND-" + uuid.NewString()[:8]).
			SetCustomerID(env.customerID).
			SetBusinessType(orderent.BusinessTypeSE).
			SetTradeDirection(orderent.TradeDirectionExport).
			SetTradeTerm(orderent.TradeTermFOB).
			SetPaymentTerm(orderent.PaymentTermPREPAID).
			SetFlowStatus(orderent.FlowStatusDRAFT).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建候选承载订单失败: %v", err)
		}
		if _, err := env.data.db.SeaMasterBillOrderLink.Create().
			SetOrganizationID(env.orgID).
			SetOrderID(carrierOrder.ID).
			SetMasterBillID(candidateMBL.ID).
			SetTransportExecutionID(candidateTE.ID).
			SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
			SetStatus(seamasterbillorderlinkent.StatusACTIVE).
			SetVersion(1).
			Save(ctx); err != nil {
			t.Fatalf("创建候选 Link 失败: %v", err)
		}

		mismatchInput := f.standardSplitInput("split-cand-mismatch-012", "fp-cand-mismatch-012", splitFixtureOptions{})
		mismatchInput.Targets = []*biz.SeaOrderSplitTargetInput{
			{ClientTargetKey: "target-current", TargetType: biz.SplitTargetTypeCurrent},
			{
				ClientTargetKey: "target-cand", TargetType: biz.SplitTargetTypeCandidate,
				CandidateID: &candidateMBL.ID, CandidateVersion: &candidateMBL.Version,
				CandidateTEID: &candidateTE.ID, CandidateTEVersion: &candidateTE.Version,
				ShippingLineID: &env.carrierID,
				VesselName:     "WRONG SHIP NAME",
				VoyageNo:       "2026E",
			},
		}
		mismatchInput.Results[1].ClientTargetKey = "target-cand"
		mismatchInput.Confirmation = &biz.SeaExternalConfirmation{ConfirmedByParty: "船代窗口", ConfirmedAt: time.Now().UTC(), ConfirmationNote: "船代确认拆票改配"}
		mismatchInput.ExpectedVersions.CandidateMBLVersions = map[uuid.UUID]uint64{candidateMBL.ID: candidateMBL.Version}
		mismatchInput.ExpectedVersions.CandidateTEVersions = map[uuid.UUID]uint64{candidateTE.ID: candidateTE.Version}

		err = func() error {
			_, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, mismatchInput)
			return err
		}()
		if kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_BLOCKED" {
			t.Fatalf("候选输入与权威TE不一致必须阻断，实际: %v", err)
		}
		if md := kratoserrors.FromError(err).Metadata; md["reason"] != "CANDIDATE_MBL_INPUT_MISMATCH" {
			t.Fatalf("阻断原因错误: %v", md["reason"])
		}

		matchInput := f.standardSplitInput("split-cand-match-012", "fp-cand-match-012", splitFixtureOptions{})
		matchInput.Targets = []*biz.SeaOrderSplitTargetInput{
			{ClientTargetKey: "target-current", TargetType: biz.SplitTargetTypeCurrent},
			{
				ClientTargetKey: "target-cand", TargetType: biz.SplitTargetTypeCandidate,
				CandidateID: &candidateMBL.ID, CandidateVersion: &candidateMBL.Version,
				CandidateTEID: &candidateTE.ID, CandidateTEVersion: &candidateTE.Version,
				ShippingLineID: &env.carrierID,
				VesselName:     "PACIFIC GLORY",
				VoyageNo:       "2026E",
			},
		}
		matchInput.Results[1].ClientTargetKey = "target-cand"
		matchInput.Confirmation = &biz.SeaExternalConfirmation{ConfirmedByParty: "船代窗口", ConfirmedAt: time.Now().UTC(), ConfirmationNote: "船代确认拆票改配"}
		matchInput.ExpectedVersions.CandidateMBLVersions = map[uuid.UUID]uint64{candidateMBL.ID: candidateMBL.Version}
		matchInput.ExpectedVersions.CandidateTEVersions = map[uuid.UUID]uint64{candidateTE.ID: candidateTE.Version}

		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, matchInput); err != nil {
			t.Fatalf("候选输入与权威TE一致应执行成功: %v", err)
		}
	})

	// N. 草稿费用整行克隆与 ResultSnapshot 映射
	t.Run("DRAFT费用整行克隆与快照映射", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "013", splitFixtureOptions{})

		input := f.standardSplitInput("split-fee-clone-013", "fp-fee-clone-013", splitFixtureOptions{})
		input.Results[0].DraftFeeIDs = nil
		input.Results[1].DraftFeeIDs = []uuid.UUID{f.fee1.ID}

		splitEvt, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, input)
		if err != nil {
			t.Fatalf("ExecuteSplit 费用克隆测试失败: %v", err)
		}

		if _, err := env.data.db.OrderFee.Get(ctx, f.fee1.ID); err == nil || !ent.IsNotFound(err) {
			t.Fatalf("原费用行必须被删除: %v", err)
		}
		var createdRes *biz.SeaOrderSplitResult
		for _, r := range splitEvt.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdRes = r
			}
		}
		newFees, _ := env.data.db.OrderFee.Query().Where(orderfeeent.OrderIDEQ(createdRes.OrderID)).All(ctx)
		if len(newFees) != 1 {
			t.Fatalf("新订单下费用数量应为 1: %d", len(newFees))
		}
		if newFees[0].ID == f.fee1.ID || newFees[0].IdempotencyKey != f.fee1.IdempotencyKey+":split:res-new-1" {
			t.Fatalf("新费用必须是全新行并携带派生幂等键: %+v", newFees[0])
		}
		var snapData map[string]interface{}
		if err := json.Unmarshal(createdRes.ResultSnapshot, &snapData); err != nil {
			t.Fatalf("反序列化 ResultSnapshot 失败: %v", err)
		}
		feeMap, ok := snapData["fee_old_new_id_map"].(map[string]interface{})
		if !ok || feeMap[f.fee1.ID.String()] != newFees[0].ID.String() {
			t.Fatalf("ResultSnapshot 未正确记录 old/new fee ID 映射: %+v", feeMap)
		}
	})

	// O. 单个货物行完全移出物理删除无零值残留（原票保留另一货物行）
	t.Run("货物完全移出物理删除", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "014", splitFixtureOptions{})

		cargo2, err := env.data.db.OrderCargoItem.Create().
			SetOrganizationID(env.orgID).
			SetOrderID(f.order.ID).
			SetCargoName("货物2-完全移出").
			SetPackageCount(50).
			SetGrossWeightKg(1000.0).
			SetVolumeCbm(8.0).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建货物2失败: %v", err)
		}

		input := f.standardSplitInput("split-cargo-move-014", "fp-cargo-move-014", splitFixtureOptions{})
		// 原票保留货物1全部 100 件，货物2 50 件整体移至新票
		input.Results[1].CargoAllocations = append(input.Results[1].CargoAllocations, &biz.SeaOrderSplitCargoAllocationInput{
			CargoItemID: cargo2.ID, PackageCount: 50, GrossWeightKg: decimalRequire("1000"), VolumeCbm: decimalRequire("8"),
		})
		input.ExpectedVersions.CargoItemVersions[cargo2.ID] = cargo2.Version

		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, input); err != nil {
			t.Fatalf("货物完全移出拆票失败: %v", err)
		}
		if _, err := env.data.db.OrderCargoItem.Get(ctx, cargo2.ID); err == nil || !ent.IsNotFound(err) {
			t.Fatalf("完全移出的原货物行必须被物理删除: %v", err)
		}
		originCargoes, _ := env.data.db.OrderCargoItem.Query().Where(ordercargoitement.OrderIDEQ(f.order.ID)).All(ctx)
		if len(originCargoes) != 1 || originCargoes[0].PackageCount <= 0 {
			t.Fatalf("原订单应仅保留非零货物行: %+v", originCargoes)
		}
	})

	// P. 版本 0 拒绝
	t.Run("版本0拒绝", func(t *testing.T) {
		f := createTestSplitFixture(t, env, "015", splitFixtureOptions{})
		input := f.standardSplitInput("split-ver-zero-015", "fp-ver-zero-015", splitFixtureOptions{})
		input.ExpectedVersions.OrderVersion = 0
		if _, err := env.uc.ExecuteSplit(ctx, env.orgID, env.userID, input); kratoserrors.Reason(err) != "SEA_ORDER_SPLIT_INVALID_ARGUMENT" {
			t.Fatalf("OrderVersion=0 必须被驳回，实际: %v", err)
		}
	})
}
