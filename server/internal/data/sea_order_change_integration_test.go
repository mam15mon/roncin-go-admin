package data

import (
	"bytes"
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"

	"github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	masterdataitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/masterdataitem"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	numbersequenceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercargoitement "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercargoitem"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	ordercontainerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainer"
	ordercontainerrequestent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercontainerrequest"
	orderenterprisetagent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderenterprisetag"
	orderfeeent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfee"
	orderfeeenterprisetagent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderfeeenterprisetag"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	seacargoallocationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seacargoallocation"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seaorderreassignmenteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaorderreassignmentevent"
	seaorderspliteventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seaordersplitevent"
	seatransportexecutionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
)

func strPtr(s string) *string {
	return &s
}

type splitTestFixture struct {
	order     *ent.Order
	link      *ent.SeaMasterBillOrderLink
	cargoItem *ent.OrderCargoItem
	cntr1     *ent.OrderContainer
	cntr2     *ent.OrderContainer
	hbl1      *ent.SeaHouseBill
	hbl2      *ent.SeaHouseBill
	fee1      *ent.OrderFee
	attRef    *ent.OrderAttachment
}

func createTestSplitFixture(t *testing.T, ctx context.Context, data *Data, orgID, customerID, carrierID, specID, mblID, userID, assetID uuid.UUID, suffix string) *splitTestFixture {
	t.Helper()

	order, err := data.db.Order.Create().
		SetOrganizationID(orgID).
		SetOrderNo("SE20260903" + suffix).
		SetCustomerID(customerID).
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

	now := time.Now().UTC()
	link, err := data.db.SeaMasterBillOrderLink.Create().
		SetOrganizationID(orgID).
		SetOrderID(order.ID).
		SetMasterBillID(mblID).
		SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
		SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED).
		SetCargoAllocationVersion(1).
		SetCargoAllocationConfirmedAt(now).
		SetCargoAllocationConfirmedBy(userID).
		SetStatus(seamasterbillorderlinkent.StatusACTIVE).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建主单关联失败: %v", err)
	}

	cargoItem, err := data.db.OrderCargoItem.Create().
		SetOrganizationID(orgID).
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
		SetOrganizationID(orgID).
		SetOrderID(order.ID).
		SetContainerNo("MSKU100" + suffix).
		SetContainerSpecID(specID).
		SetPackageCount(60).
		SetGrossWeightKg(1200.0).
		SetVolumeCbm(9.0).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建箱1失败: %v", err)
	}

	cntr2, err := data.db.OrderContainer.Create().
		SetOrganizationID(orgID).
		SetOrderID(order.ID).
		SetContainerNo("MSKU200" + suffix).
		SetContainerSpecID(specID).
		SetPackageCount(40).
		SetGrossWeightKg(800.0).
		SetVolumeCbm(6.0).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建箱2失败: %v", err)
	}

	hbl1, err := data.db.SeaHouseBill.Create().
		SetOrganizationID(orgID).
		SetOrderID(order.ID).
		SetMasterBillID(mblID).
		SetHouseNo("HBL-1-" + suffix).
		SetNormalizedHouseNo("HBL-1-" + suffix).
		SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
		SetIssuerOrganizationID(orgID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建HBL1失败: %v", err)
	}

	hbl2, err := data.db.SeaHouseBill.Create().
		SetOrganizationID(orgID).
		SetOrderID(order.ID).
		SetMasterBillID(mblID).
		SetHouseNo("HBL-2-" + suffix).
		SetNormalizedHouseNo("HBL-2-" + suffix).
		SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
		SetIssuerOrganizationID(orgID).
		SetStatus(seahousebillent.StatusDRAFT).
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建HBL2失败: %v", err)
	}

	_, err = data.db.SeaCargoAllocation.Create().
		SetOrganizationID(orgID).
		SetOrderID(order.ID).
		SetMasterBillOrderLinkID(link.ID).
		SetCargoItemID(cargoItem.ID).
		SetHouseBillID(hbl1.ID).
		SetContainerID(cntr1.ID).
		SetPackageCount(60).
		SetGrossWeightKg("1200.000").
		SetVolumeCbm("9.000000").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建分配1失败: %v", err)
	}

	_, err = data.db.SeaCargoAllocation.Create().
		SetOrganizationID(orgID).
		SetOrderID(order.ID).
		SetMasterBillOrderLinkID(link.ID).
		SetCargoItemID(cargoItem.ID).
		SetHouseBillID(hbl2.ID).
		SetContainerID(cntr2.ID).
		SetPackageCount(40).
		SetGrossWeightKg("800.000").
		SetVolumeCbm("6.000000").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建分配2失败: %v", err)
	}

	fee1, err := data.db.OrderFee.Create().
		SetOrderID(order.ID).
		SetIdempotencyKey("fee-" + suffix).
		SetDirection(orderfeeent.DirectionRECEIVABLE).
		SetStatus(orderfeeent.StatusDRAFT).
		SetFeeCode("OFT").
		SetFeeName("海运基本费").
		SetSettlementPartyID(customerID).
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
		SetExchangeRateDate("2026-09-03").
		SetBaseCurrency("CNY").
		SetBaseCurrencyAmount("10650.00000000").
		SetExpenseDate("2026-09-03").
		SetVersion(1).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建费用失败: %v", err)
	}

	attRef, err := data.db.OrderAttachment.Create().
		SetOrderID(order.ID).
		SetAssetID(assetID).
		SetDocType("BOOKING_NOTE").
		SetIdempotencyKey("att-" + suffix).
		SetCreatedBy(userID).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建附件引用失败: %v", err)
	}

	return &splitTestFixture{
		order:     order,
		link:      link,
		cargoItem: cargoItem,
		cntr1:     cntr1,
		cntr2:     cntr2,
		hbl1:      hbl1,
		hbl2:      hbl2,
		fee1:      fee1,
		attRef:    attRef,
	}
}

func buildFixtureExpectedVersions(f *splitTestFixture) *biz.SeaOrderSplitExpectedVersions {
	hblVersions := map[uuid.UUID]uint64{
		f.hbl1.ID: f.hbl1.Version,
		f.hbl2.ID: f.hbl2.Version,
	}
	cargoVersions := map[uuid.UUID]uint64{
		f.cargoItem.ID: f.cargoItem.Version,
	}
	cntrVersions := map[uuid.UUID]uint64{
		f.cntr1.ID: f.cntr1.Version,
		f.cntr2.ID: f.cntr2.Version,
	}
	feeVersions := map[uuid.UUID]uint64{
		f.fee1.ID: f.fee1.Version,
	}
	attItems := []*biz.SeaOrderSplitAttachmentItem{
		{ID: f.attRef.ID, AssetID: f.attRef.AssetID, DocType: f.attRef.DocType},
	}
	attFp := biz.ComputeAttachmentFingerprint(attItems)

	return &biz.SeaOrderSplitExpectedVersions{
		OrderVersion:                   f.order.Version,
		LinkVersion:                    f.link.Version,
		AllocationVersion:              1,
		HouseBillVersions:              hblVersions,
		CargoItemVersions:              cargoVersions,
		ContainerVersions:              cntrVersions,
		FeeVersions:                    feeVersions,
		AttachmentReferenceFingerprint: attFp,
	}
}

func TestSeaOrderSplitAndReassignment_PostgresIntegration(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	ctx := context.Background()
	repo := NewSeaOrderChangeRepo(data)
	uc := biz.NewSeaOrderChangeUsecase(repo, data)

	// 1. 基础测试主数据准备
	org, err := data.db.Organization.Create().
		SetCode("ORG-SPLIT-" + uuid.New().String()[:8]).
		SetName("拆票改配集成测试组织").
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建组织失败: %v", err)
	}

	orderRule, err := data.db.NumberRule.Create().
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

	_, err = data.db.Membership.Create().
		SetOrganizationID(org.ID).
		SetUserID(user.ID).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建用户成员关系失败: %v", err)
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
	if _, err := data.db.PartnerRole.Create().
		SetPartnerID(customer.ID).
		SetRoleType(partnerroleent.RoleTypeCustomer).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建客户角色失败: %v", err)
	}

	carrier, err := data.db.Partner.Create().
		SetOrganizationID(org.ID).
		SetCode("CARR-" + uuid.New().String()[:8]).
		SetLegalName("测试船公司").
		SetNormalizedName("测试船公司").
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建船公司失败: %v", err)
	}
	if _, err := data.db.PartnerRole.Create().
		SetPartnerID(carrier.ID).
		SetRoleType(partnerroleent.RoleTypeCarrier).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建船公司角色失败: %v", err)
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
		SetVesselName("MAERSK MC-KINNEY MOLLER").
		SetVoyageNo("2601W").
		SetEtd(now.Add(24 * time.Hour)).
		SetEta(now.Add(240 * time.Hour)).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建运输执行失败: %v", err)
	}

	mbl, err := data.db.SeaMasterBill.Create().
		SetOrganizationID(org.ID).
		SetTransportExecutionID(te.ID).
		SetIssuerPartnerID(carrier.ID).
		SetMasterNo("MSK987654321").
		SetNormalizedMasterNo("MSK987654321").
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

	// -------------------------------------------------------------------------
	// A. 拆票成功、守恒与同键同/异指纹重试
	// -------------------------------------------------------------------------
	t.Run("拆票守恒与同键重试指纹校验", func(t *testing.T) {
		f := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "001")

		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            f.order.ID,
			IdempotencyKey:     "split-test-idemp-001",
			RequestFingerprint: "fp-split-001",
			Note:               strPtr("业务部分拆票"),
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey:        "res-origin",
					ResultRole:             biz.ResultRoleOriginal,
					ClientTargetKey:        "res-origin",
					HouseBillIDs:           []uuid.UUID{f.hbl1.ID},
					DraftFeeIDs:            []uuid.UUID{f.fee1.ID},
					AttachmentReferenceIDs: []uuid.UUID{f.attRef.ID},
				},
				{
					ClientResultKey:        "res-new-1",
					ResultRole:             biz.ResultRoleCreated,
					ClientTargetKey:        "res-new-1",
					HouseBillIDs:           []uuid.UUID{f.hbl2.ID},
					DraftFeeIDs:            nil,
					AttachmentReferenceIDs: []uuid.UUID{f.attRef.ID},
				},
			},
			ExpectedVersions: buildFixtureExpectedVersions(f),
		}

		// 1. 首次拆票执行成功
		splitEvent, splitErr := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if splitErr != nil {
			t.Fatalf("ExecuteSplit 失败: %v", splitErr)
		}
		if splitEvent == nil || len(splitEvent.Results) != 2 {
			t.Fatalf("拆票事件返回结果异常: %+v", splitEvent)
		}

		var createdOrderID uuid.UUID
		var createdOrderNo string
		for _, r := range splitEvent.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
				createdOrderNo = r.OrderNo
			}
		}
		if createdOrderID == uuid.Nil || createdOrderNo == "" {
			t.Fatalf("未生成有效的新订单号与ID")
		}

		// 验证源订单和新订单货物守恒
		originCargo, _ := data.db.OrderCargoItem.Query().Where(ordercargoitement.OrderIDEQ(f.order.ID)).Only(ctx)
		if originCargo.PackageCount != 60 || originCargo.GrossWeightKg != 1200.0 || originCargo.VolumeCbm != 9.0 {
			t.Errorf("源订单货物数量守恒异常: %+v", originCargo)
		}
		newCargo, _ := data.db.OrderCargoItem.Query().Where(ordercargoitement.OrderIDEQ(createdOrderID)).Only(ctx)
		if newCargo.PackageCount != 40 || newCargo.GrossWeightKg != 800.0 || newCargo.VolumeCbm != 6.0 {
			t.Errorf("新订单货物数量守恒异常: %+v", newCargo)
		}

		// 验证分单、箱、费用归属
		hbl1Check, _ := data.db.SeaHouseBill.Get(ctx, f.hbl1.ID)
		hbl2Check, _ := data.db.SeaHouseBill.Get(ctx, f.hbl2.ID)
		if hbl1Check.OrderID != f.order.ID || hbl2Check.OrderID != createdOrderID {
			t.Errorf("分单归属异常: HBL1=%v, HBL2=%v", hbl1Check.OrderID, hbl2Check.OrderID)
		}
		cntr1Check, _ := data.db.OrderContainer.Get(ctx, f.cntr1.ID)
		cntr2Check, _ := data.db.OrderContainer.Get(ctx, f.cntr2.ID)
		if cntr1Check.OrderID != f.order.ID || cntr2Check.OrderID != createdOrderID {
			t.Errorf("集装箱归属异常: cntr1=%v, cntr2=%v", cntr1Check.OrderID, cntr2Check.OrderID)
		}
		feeCheck, _ := data.db.OrderFee.Get(ctx, f.fee1.ID)
		if feeCheck.OrderID != f.order.ID {
			t.Errorf("草稿费用应保留在源订单: got %v, want %v", feeCheck.OrderID, f.order.ID)
		}

		// 2. 幂等同键同指纹重试（先查幂等键，不因当前订单HBL状态已变而报错）
		t.Run("幂等同键同指纹重试返回既有事件", func(t *testing.T) {
			sameEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
			if err != nil {
				t.Fatalf("幂等重试应成功，实际: %v", err)
			}
			if sameEvt.ID != splitEvent.ID {
				t.Errorf("幂等重试应返回同一事件ID: got %v, want %v", sameEvt.ID, splitEvent.ID)
			}
		})

		// 3. 幂等同键异指纹拦截（先查幂等键并比对指纹，准确返回冲突错误）
		t.Run("幂等同键异指纹准确返回冲突", func(t *testing.T) {
			conflictInput := *splitInput
			conflictInput.RequestFingerprint = "diff-fp-split-999"
			_, err := uc.ExecuteSplit(ctx, org.ID, user.ID, &conflictInput)
			if err == nil {
				t.Fatal("不同指纹重试应拦截报错")
			}
			if reason := errors.Reason(err); reason != "SEA_ORDER_SPLIT_IDEMPOTENCY_CONFLICT" {
				t.Errorf("错误码应为 SEA_ORDER_SPLIT_IDEMPOTENCY_CONFLICT, 实际: %v", reason)
			}
		})
	})

	// -------------------------------------------------------------------------
	// B. 事务中途回滚测试（独立未拆分 fixture，OrderVersion 正确，LinkVersion 错误）
	// -------------------------------------------------------------------------
	t.Run("拆票中途错误全事务回滚", func(t *testing.T) {
		fRollback := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "002")

		// 记录回滚前的订单总数、源订单版本、以及当前的序列号记录
		orderCountBefore, _ := data.db.Order.Query().Count(ctx)
		sourceBefore, _ := data.db.Order.Get(ctx, fRollback.order.ID)

		seqBefore, _ := data.db.NumberSequence.Query().
			Where(numbersequenceent.RuleIDEQ(orderRule.ID)).
			Only(ctx)
		var seqValueBefore int64
		if seqBefore != nil {
			seqValueBefore = seqBefore.CurrentValue
		}

		// 构造入参：OrderVersion 完全正确，但 LinkVersion 故意设置为 999999
		// 执行路径：锁定源订单成功 -> 分配订单号序列成功 -> 锁定主单关联 LinkVersion 检查失败 -> 事务中途回滚
		rollbackInput := &biz.SeaOrderSplitInput{
			OrderID:            fRollback.order.ID,
			IdempotencyKey:     "split-mid-rollback-idemp",
			RequestFingerprint: "fp-mid-rollback",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "res-origin",
					HouseBillIDs:    []uuid.UUID{fRollback.hbl1.ID},
					DraftFeeIDs:     []uuid.UUID{fRollback.fee1.ID},
				},
				{
					ClientResultKey: "res-new-1",
					ResultRole:      biz.ResultRoleCreated,
					ClientTargetKey: "res-new-1",
					HouseBillIDs:    []uuid.UUID{fRollback.hbl2.ID},
				},
			},
		}
		expRollback := buildFixtureExpectedVersions(fRollback)
		expRollback.LinkVersion = 999999 // 故意错误的 LinkVersion
		rollbackInput.ExpectedVersions = expRollback

		_, err := uc.ExecuteSplit(ctx, org.ID, user.ID, rollbackInput)
		if err == nil {
			t.Fatal("LinkVersion 冲突应导致报错")
		}
		if reason := errors.Reason(err); reason != "SEA_ORDER_SPLIT_VERSION_CONFLICT" {
			t.Errorf("期望错误码 SEA_ORDER_SPLIT_VERSION_CONFLICT, 实际: %v", reason)
		}

		// 1. 验证订单数未增加
		orderCountAfter, _ := data.db.Order.Query().Count(ctx)
		if orderCountAfter != orderCountBefore {
			t.Errorf("事务回滚失败，订单总数变化: before=%d, after=%d", orderCountBefore, orderCountAfter)
		}

		// 2. 验证源订单版本未递增
		sourceAfter, _ := data.db.Order.Get(ctx, fRollback.order.ID)
		if sourceAfter.Version != sourceBefore.Version {
			t.Errorf("源订单版本号不应递增: before=%d, after=%d", sourceBefore.Version, sourceAfter.Version)
		}

		// 3. 验证拆票事件表无数据
		eventExists, _ := data.db.SeaOrderSplitEvent.Query().
			Where(seaorderspliteventent.IdempotencyKeyEQ("split-mid-rollback-idemp")).
			Exist(ctx)
		if eventExists {
			t.Errorf("事务回滚失败，拆票事件记录不应存在")
		}

		// 4. 验证预分配的订单号码序列已全量回滚（CurrentValue 未递增）
		seqAfter, _ := data.db.NumberSequence.Query().
			Where(numbersequenceent.RuleIDEQ(orderRule.ID)).
			Only(ctx)
		var seqValueAfter int64
		if seqAfter != nil {
			seqValueAfter = seqAfter.CurrentValue
		}
		if seqValueAfter != seqValueBefore {
			t.Errorf("号码序列未回滚: before=%d, after=%d", seqValueBefore, seqValueAfter)
		}
	})

	// -------------------------------------------------------------------------
	// C. DIRECT 模式整票改配与目标 MBL 兼容性
	// -------------------------------------------------------------------------
	t.Run("DIRECT模式整票改配与目标MBL兼容", func(t *testing.T) {
		directOrder, err := data.db.Order.Create().
			SetOrganizationID(org.ID).
			SetOrderNo("SE20260903088").
			SetCustomerID(customer.ID).
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

		directLink, err := data.db.SeaMasterBillOrderLink.Create().
			SetOrganizationID(org.ID).
			SetOrderID(directOrder.ID).
			SetMasterBillID(mbl.ID).
			SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureDIRECT).
			SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusDRAFT).
			SetStatus(seamasterbillorderlinkent.StatusACTIVE).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建DIRECT关联失败: %v", err)
		}

		// 1. DIRECT 模式尝试拆票应被阻断
		t.Run("DIRECT模式拆票被阻断", func(t *testing.T) {
			_, err := uc.ExecuteSplit(ctx, org.ID, user.ID, &biz.SeaOrderSplitInput{
				OrderID:            directOrder.ID,
				IdempotencyKey:     "direct-split-fail",
				RequestFingerprint: "fp-direct",
				Targets: []*biz.SeaOrderSplitTargetInput{
					{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
					{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
				},
				Results: []*biz.SeaOrderSplitResultInput{
					{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin"},
					{ClientResultKey: "res-new-1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new-1"},
				},
			})
			if err == nil {
				t.Fatal("DIRECT 拆票应被阻断")
			}
		})

		// 2. 改配至当前同一 MBL 产生冲突拦截
		t.Run("改配至当前同一MBL冲突拦截", func(t *testing.T) {
			sameMblInput := &biz.SeaOrderReassignmentInput{
				OrderID:            directOrder.ID,
				IdempotencyKey:     "reas-same-mbl",
				RequestFingerprint: "fp-same-mbl",
				Reason:             "尝试改配至相同母单",
				ResponsibilityType: biz.ResponsibilityTypeCustomer,
				Target: &biz.SeaOrderReassignmentTargetInput{
					TargetType:  biz.SplitTargetTypeCandidate,
					CandidateID: &mbl.ID,
				},
			}
			_, err := uc.ExecuteReassignment(ctx, org.ID, user.ID, sameMblInput)
			if err == nil {
				t.Fatal("改配至当前同一MBL应报错")
			}
		})

		// 3. DIRECT 模式执行整体改配至新 MBL 成功
		t.Run("DIRECT模式整票改配成功", func(t *testing.T) {
			reassignInput := &biz.SeaOrderReassignmentInput{
				OrderID:            directOrder.ID,
				IdempotencyKey:     "reas-direct-001",
				RequestFingerprint: "fp-reas-001",
				Reason:             "船期延误客户申请改配至新母单",
				ResponsibilityType: biz.ResponsibilityTypeCarrier,
				Target: &biz.SeaOrderReassignmentTargetInput{
					TargetType:          biz.SplitTargetTypeNew,
					MasterNo:            "NEWMBL888999",
					IssuerPartnerID:     &carrier.ID,
					VesselName:          "COSCO SHIPPING GEMINI",
					VoyageNo:            "088E",
					OriginLocationID:    nil,
					DischargeLocationID: nil,
				},
				ExpectedOrderVersion: directOrder.Version,
				ExpectedLinkVersion:  directLink.Version,
			}

			reasEvt, err := uc.ExecuteReassignment(ctx, org.ID, user.ID, reassignInput)
			if err != nil {
				t.Fatalf("DIRECT 改配执行失败: %v", err)
			}
			if reasEvt == nil || reasEvt.OrderNo != directOrder.OrderNo {
				t.Fatalf("改配返回结果异常: %+v", reasEvt)
			}

			// 验证旧关联 ENDED，新关联 ACTIVE
			oldLinkAfter, _ := data.db.SeaMasterBillOrderLink.Get(ctx, directLink.ID)
			if oldLinkAfter.Status != seamasterbillorderlinkent.StatusENDED {
				t.Errorf("旧主单关联应为 ENDED, 实际: %v", oldLinkAfter.Status)
			}

			newLinkAfter, err := data.db.SeaMasterBillOrderLink.Query().
				Where(
					seamasterbillorderlinkent.OrderIDEQ(directOrder.ID),
					seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
				).
				Only(ctx)
			if err != nil {
				t.Fatalf("查询新活动关联失败: %v", err)
			}
			if newLinkAfter.MasterBillID == mbl.ID {
				t.Errorf("新关联母单ID不应为旧MBL")
			}

			// 验证改配幂等性（先查幂等记录，同指纹直接返回成功）
			sameReasEvt, err := uc.ExecuteReassignment(ctx, org.ID, user.ID, reassignInput)
			if err != nil {
				t.Fatalf("改配幂等执行失败: %v", err)
			}
			if sameReasEvt.ID != reasEvt.ID {
				t.Errorf("改配幂等事件ID应一致: got %v, want %v", sameReasEvt.ID, reasEvt.ID)
			}

			// 验证改配异指纹冲突
			diffFpInput := *reassignInput
			diffFpInput.RequestFingerprint = "diff-fp-reas-999"
			_, err = uc.ExecuteReassignment(ctx, org.ID, user.ID, &diffFpInput)
			if err == nil {
				t.Fatal("改配异指纹应冲突报错")
			}
			if reason := errors.Reason(err); reason != "SEA_ORDER_REASSIGNMENT_IDEMPOTENCY_CONFLICT" {
				t.Errorf("期望错误码 SEA_ORDER_REASSIGNMENT_IDEMPOTENCY_CONFLICT, 实际: %v", reason)
			}
		})
	})

	// -------------------------------------------------------------------------
	// D. 子票语义继承、CONFIRMED状态保持、内部单号空值与人员提成归属快照
	// -------------------------------------------------------------------------
	t.Run("子票语义继承与CONFIRMED状态保持", func(t *testing.T) {
		fSem := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "003")

		// 明确设置源订单状态为 BOOKED，且设置了内部参考号和客户参考号
		data.db.Order.UpdateOneID(fSem.order.ID).
			SetFlowStatus(orderent.FlowStatusBOOKED).
			SetInternalReferenceNo("INT-REF-SRC-001").
			SetCustomerReferenceNo("PO-CUST-999").
			SaveX(ctx)

		// 为源订单绑定人员与提成归属快照
		data.db.OrderPersonnel.Create().
			SetOrganizationID(org.ID).
			SetOrderID(fSem.order.ID).
			SetUserID(user.ID).
			SetRole(orderpersonnelent.RoleOPERATOR).
			SaveX(ctx)

		data.db.OrderCommissionAttribution.Create().
			SetOrganizationID(org.ID).
			SetOrderID(fSem.order.ID).
			SetCustomerID(customer.ID).
			SetSourceAssignmentID(uuid.Must(uuid.NewV7())).
			SetEmployeeID(user.ID).
			SetEmployeeName("拆票测试操作员").
			SetPersonnelRole("OPERATOR").
			SetAttributedAt(time.Now().UTC()).
			SaveX(ctx)

		srcOrderReloaded, _ := data.db.Order.Get(ctx, fSem.order.ID)

		// 拆票输入：res-new-1 不输入内部编号 (nil)
		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fSem.order.ID,
			IdempotencyKey:     "split-semantics-003",
			RequestFingerprint: "fp-semantics-003",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "res-origin",
					HouseBillIDs:    []uuid.UUID{fSem.hbl1.ID},
					DraftFeeIDs:     []uuid.UUID{fSem.fee1.ID},
				},
				{
					ClientResultKey:     "res-new-1",
					ResultRole:          biz.ResultRoleCreated,
					ClientTargetKey:     "res-new-1",
					HouseBillIDs:        []uuid.UUID{fSem.hbl2.ID},
					InternalReferenceNo: nil, // 未输入时必须为空，绝不能复制原内部编号!
				},
			},
			ExpectedVersions: func() *biz.SeaOrderSplitExpectedVersions {
				ev := buildFixtureExpectedVersions(fSem)
				ev.OrderVersion = srcOrderReloaded.Version
				return ev
			}(),
		}

		splitEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err != nil {
			t.Fatalf("ExecuteSplit 语义测试失败: %v", err)
		}

		var createdOrderID uuid.UUID
		for _, r := range splitEvt.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
			}
		}

		// 验证新订单继承 FlowStatus=BOOKED (非 DRAFT)
		newOrder, err := data.db.Order.Get(ctx, createdOrderID)
		if err != nil {
			t.Fatalf("获取新订单失败: %v", err)
		}
		if newOrder.FlowStatus != orderent.FlowStatusBOOKED {
			t.Errorf("新子票应保持 BOOKED 状态, 实际: %v", newOrder.FlowStatus)
		}

		// 验证内部参考号未输入时为空字符串，客户参考号继承
		if newOrder.InternalReferenceNo != "" {
			t.Errorf("新子票内部参考号应为空, 实际: %s", newOrder.InternalReferenceNo)
		}
		if newOrder.CustomerReferenceNo != "PO-CUST-999" {
			t.Errorf("新子票应继承客户参考号 PO-CUST-999, 实际: %s", newOrder.CustomerReferenceNo)
		}

		// 验证人员快照已复制
		personnelCount, _ := data.db.OrderPersonnel.Query().Where(orderpersonnelent.OrderIDEQ(createdOrderID)).Count(ctx)
		if personnelCount != 1 {
			t.Errorf("新子票人员复制失败, 数量: %d", personnelCount)
		}

		// 验证提成归属快照已复制
		attrCount, _ := data.db.OrderCommissionAttribution.Query().Where(ordercommissionattributionent.OrderIDEQ(createdOrderID)).Count(ctx)
		if attrCount != 1 {
			t.Errorf("新子票提成归属复制失败, 数量: %d", attrCount)
		}

		// 验证新子票生命周期事件
		originEvt, err := data.db.OrderLifecycleEvent.Query().
			Where(
				orderlifecycleeventent.OrderIDEQ(createdOrderID),
				orderlifecycleeventent.ActionEQ("CREATED_BY_SPLIT"),
			).
			Only(ctx)
		if err != nil {
			t.Fatalf("新子票生命周期事件未找到: %v", err)
		}
		if originEvt.ToStatus != "BOOKED" {
			t.Errorf("生命周期事件 ToStatus 应为 BOOKED, 实际: %s", originEvt.ToStatus)
		}
	})

	// -------------------------------------------------------------------------
	// E. DRAFT费用整行克隆与删除、标签关联克隆与ResultSnapshot映射
	// -------------------------------------------------------------------------
	t.Run("DRAFT费用整行克隆删除与标签快照映射", func(t *testing.T) {
		fFee := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "004")

		// 给 fFee.fee1 关联企业标签
		tagRes, err := data.db.EnterpriseResource.Create().
			SetOrganizationID(org.ID).
			SetResourceType("TAG").
			SetShortName("测试费用业务标签").
			SetCreatedBy(user.ID).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建企业标签资源失败: %v", err)
		}
		tagResourceID := tagRes.ID
		data.db.OrderFeeEnterpriseTag.Create().
			SetOrganizationID(org.ID).
			SetOrderFeeID(fFee.fee1.ID).
			SetTagResourceID(tagResourceID).
			SaveX(ctx)

		// 将费用分配给新票 res-new-1
		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fFee.order.ID,
			IdempotencyKey:     "split-fee-clone-004",
			RequestFingerprint: "fp-fee-clone-004",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "res-origin",
					HouseBillIDs:    []uuid.UUID{fFee.hbl1.ID},
				},
				{
					ClientResultKey: "res-new-1",
					ResultRole:      biz.ResultRoleCreated,
					ClientTargetKey: "res-new-1",
					HouseBillIDs:    []uuid.UUID{fFee.hbl2.ID},
					DraftFeeIDs:     []uuid.UUID{fFee.fee1.ID}, // 费用移动至新票
				},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fFee),
		}

		splitEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err != nil {
			t.Fatalf("ExecuteSplit 费用克隆测试失败: %v", err)
		}

		// 1. 原费用必须已被彻底物理删除
		_, err = data.db.OrderFee.Get(ctx, fFee.fee1.ID)
		if err == nil || !ent.IsNotFound(err) {
			t.Errorf("原费用行必须被删除: %v", err)
		}

		// 2. 原费用的标签关联也已被删除
		oldTagCount, _ := data.db.OrderFeeEnterpriseTag.Query().
			Where(orderfeeenterprisetagent.OrderFeeIDEQ(fFee.fee1.ID)).
			Count(ctx)
		if oldTagCount != 0 {
			t.Errorf("原费用的标签关联未删除: %d", oldTagCount)
		}

		// 3. 找到新票并验证克隆的新费用
		var createdOrderID uuid.UUID
		var createdRes *biz.SeaOrderSplitResult
		for _, r := range splitEvt.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
				createdRes = r
			}
		}

		newFees, err := data.db.OrderFee.Query().
			Where(orderfeeent.OrderIDEQ(createdOrderID)).
			All(ctx)
		if err != nil || len(newFees) != 1 {
			t.Fatalf("新订单下费用查询失败或数量不为 1: count=%d, err=%v", len(newFees), err)
		}
		newFee := newFees[0]

		if newFee.ID == fFee.fee1.ID {
			t.Errorf("新费用必须具有全新 UUID，严禁原地只改 order_id!")
		}
		expectedKey := fFee.fee1.IdempotencyKey + ":split:res-new-1"
		if newFee.IdempotencyKey != expectedKey {
			t.Errorf("新费用幂等键不匹配: got %s, want %s", newFee.IdempotencyKey, expectedKey)
		}
		if newFee.TotalAmount != fFee.fee1.TotalAmount || newFee.Currency != fFee.fee1.Currency || newFee.ExchangeRate != fFee.fee1.ExchangeRate {
			t.Errorf("新费用财务要素与汇率快照未完整复制: %+v", newFee)
		}

		// 4. 验证新费用具备克隆的标签关联
		newTag, err := data.db.OrderFeeEnterpriseTag.Query().
			Where(orderfeeenterprisetagent.OrderFeeIDEQ(newFee.ID)).
			Only(ctx)
		if err != nil || newTag.TagResourceID != tagResourceID {
			t.Errorf("新费用的标签关联复制失败: %v", err)
		}

		// 5. 验证 ResultSnapshot 包含 fee_old_new_id_map 映射
		var snapData map[string]interface{}
		if err := json.Unmarshal(createdRes.ResultSnapshot, &snapData); err != nil {
			t.Fatalf("反序列化 ResultSnapshot 失败: %v", err)
		}
		feeMap, ok := snapData["fee_old_new_id_map"].(map[string]interface{})
		if !ok || feeMap[fFee.fee1.ID.String()] != newFee.ID.String() {
			t.Errorf("ResultSnapshot 未正确记录 old/new fee ID 映射: %+v", feeMap)
		}
	})

	// -------------------------------------------------------------------------
	// F. 拆票目标换入新 MBL：HBL.master_bill_id 迁移、新活动 Link 与 allocation 迁移
	// -------------------------------------------------------------------------
	t.Run("拆票结果换入新MBL并迁移HBL与Allocation", func(t *testing.T) {
		fMbl := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "005")

		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fMbl.order.ID,
			IdempotencyKey:     "split-target-new-mbl-005",
			RequestFingerprint: "fp-target-new-mbl-005",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{
					ClientTargetKey: "res-new-1",
					TargetType:      biz.SplitTargetTypeNew,
					MasterNo:        "NEWSPLITMBL9999",
					IssuerPartnerID: &carrier.ID,
					VesselName:      "EVER GIVEN",
					VoyageNo:        "001W",
				},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "res-origin",
					HouseBillIDs:    []uuid.UUID{fMbl.hbl1.ID},
					DraftFeeIDs:     []uuid.UUID{fMbl.fee1.ID},
				},
				{
					ClientResultKey: "res-new-1",
					ResultRole:      biz.ResultRoleCreated,
					ClientTargetKey: "res-new-1",
					HouseBillIDs:    []uuid.UUID{fMbl.hbl2.ID},
				},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fMbl),
		}

		splitEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err != nil {
			t.Fatalf("ExecuteSplit 新MBL测试失败: %v", err)
		}

		var createdOrderID uuid.UUID
		for _, r := range splitEvt.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
			}
		}

		// 1. 新订单必须拥有新活动 Link，且指向新的 MasterBillID
		newLink, err := data.db.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrderIDEQ(createdOrderID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Only(ctx)
		if err != nil {
			t.Fatalf("查询新票活动 Link 失败: %v", err)
		}
		if newLink.MasterBillID == mbl.ID {
			t.Errorf("新票 Link.MasterBillID 不应指向旧 MBL")
		}

		// 2. HBL2 的 MasterBillID 和 OrderID 必须真正迁移
		hbl2Reloaded, err := data.db.SeaHouseBill.Get(ctx, fMbl.hbl2.ID)
		if err != nil {
			t.Fatalf("获取 HBL2 失败: %v", err)
		}
		if hbl2Reloaded.OrderID != createdOrderID {
			t.Errorf("HBL2 OrderID 未迁移: got %v, want %v", hbl2Reloaded.OrderID, createdOrderID)
		}
		if hbl2Reloaded.MasterBillID != newLink.MasterBillID {
			t.Errorf("HBL2 MasterBillID 未迁移至新 MBL: got %v, want %v", hbl2Reloaded.MasterBillID, newLink.MasterBillID)
		}

		// 3. SeaCargoAllocation 必须指向新 Link 和新 CargoItem
		newCargoItem, err := data.db.OrderCargoItem.Query().
			Where(ordercargoitement.OrderIDEQ(createdOrderID)).
			Only(ctx)
		if err != nil {
			t.Fatalf("查询新货物失败: %v", err)
		}

		alloc2, err := data.db.SeaCargoAllocation.Query().
			Where(seacargoallocationent.HouseBillIDEQ(fMbl.hbl2.ID)).
			Only(ctx)
		if err != nil {
			t.Fatalf("查询 HBL2 分配失败: %v", err)
		}
		if alloc2.MasterBillOrderLinkID != newLink.ID {
			t.Errorf("Allocation MasterBillOrderLinkID 未更新: got %v, want %v", alloc2.MasterBillOrderLinkID, newLink.ID)
		}
		if alloc2.CargoItemID != newCargoItem.ID {
			t.Errorf("Allocation CargoItemID 未更新为新货物 ID: got %v, want %v", alloc2.CargoItemID, newCargoItem.ID)
		}
		if alloc2.OrderID != createdOrderID {
			t.Errorf("Allocation OrderID 未更新为新订单 ID: got %v, want %v", alloc2.OrderID, createdOrderID)
		}
	})

	// -------------------------------------------------------------------------
	// G. 严格版本校验门禁（版本 0 或缺失必须驳回）
	// -------------------------------------------------------------------------
	t.Run("严格版本校验门禁拒绝版本0", func(t *testing.T) {
		fVer := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "006")

		// 1. Split 中 ExpectedVersions 为空或 OrderVersion=0
		_, err := uc.ExecuteSplit(ctx, org.ID, user.ID, &biz.SeaOrderSplitInput{
			OrderID:            fVer.order.ID,
			IdempotencyKey:     "split-ver-zero-test",
			RequestFingerprint: "fp-ver-zero",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fVer.hbl1.ID}},
				{ClientResultKey: "res-new-1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new-1", HouseBillIDs: []uuid.UUID{fVer.hbl2.ID}},
			},
			ExpectedVersions: &biz.SeaOrderSplitExpectedVersions{
				OrderVersion: 0, // 0 必须被驳回
			},
		})
		if err == nil {
			t.Fatal("OrderVersion=0 必须被驳回")
		}

		// 2. Reassignment 中 ExpectedOrderVersion=0 或 ExpectedLinkVersion=0
		_, err = uc.ExecuteReassignment(ctx, org.ID, user.ID, &biz.SeaOrderReassignmentInput{
			OrderID:            fVer.order.ID,
			IdempotencyKey:     "reas-ver-zero-test",
			RequestFingerprint: "fp-reas-zero",
			Reason:             "测试版本0拦截",
			ResponsibilityType: biz.ResponsibilityTypeCustomer,
			Target: &biz.SeaOrderReassignmentTargetInput{
				TargetType: biz.SplitTargetTypeNew,
				MasterNo:   "VERZEROMBL",
			},
			ExpectedOrderVersion: 0, // 0 必须被驳回
			ExpectedLinkVersion:  1,
		})
		if err == nil {
			t.Fatal("Reassignment ExpectedOrderVersion=0 必须被驳回")
		}
	})

	// -------------------------------------------------------------------------
	// H. 完全移动某货物行至子票且原货物行彻底删除无零值残留
	// -------------------------------------------------------------------------
	t.Run("完全移动某货物行至子票且原货物行彻底删除无零值残留", func(t *testing.T) {
		fCargo := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "007")

		// 调整原货物行1为 60 件，使原订单总量与分配总量严格守恒 (60 + 50 = 110)
		data.db.OrderCargoItem.UpdateOneID(fCargo.cargoItem.ID).
			SetPackageCount(60).
			SetGrossWeightKg(1200.0).
			SetVolumeCbm(9.0).
			SaveX(ctx)

		// 额外添加第 2 个货物行：50 件，1000kg，8cbm
		cargo2, err := data.db.OrderCargoItem.Create().
			SetOrganizationID(org.ID).
			SetOrderID(fCargo.order.ID).
			SetCargoName("货物2-完全移动").
			SetPackageCount(50).
			SetGrossWeightKg(1000.0).
			SetVolumeCbm(8.0).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建货物2失败: %v", err)
		}

		// 把分配 2 关联到 cargo2
		data.db.SeaCargoAllocation.Update().
			Where(
				seacargoallocationent.HouseBillIDEQ(fCargo.hbl2.ID),
			).
			SetCargoItemID(cargo2.ID).
			SetPackageCount(50).
			SetGrossWeightKg("1000.000").
			SetVolumeCbm("8.000000").
			SaveX(ctx)

		exp := buildFixtureExpectedVersions(fCargo)
		exp.CargoItemVersions[cargo2.ID] = cargo2.Version

		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fCargo.order.ID,
			IdempotencyKey:     "split-cargo-delete-zero-007",
			RequestFingerprint: "fp-cargo-delete-zero-007",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "res-origin",
					HouseBillIDs:    []uuid.UUID{fCargo.hbl1.ID},
					DraftFeeIDs:     []uuid.UUID{fCargo.fee1.ID},
				},
				{
					ClientResultKey: "res-new-1",
					ResultRole:      biz.ResultRoleCreated,
					ClientTargetKey: "res-new-1",
					HouseBillIDs:    []uuid.UUID{fCargo.hbl2.ID}, // 关联 cargo2，全部移动至子票
				},
			},
			ExpectedVersions: exp,
		}

		splitEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err != nil {
			t.Fatalf("ExecuteSplit 货物删除测试失败: %v", err)
		}

		var createdOrderID uuid.UUID
		for _, r := range splitEvt.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
			}
		}

		// 1. 原订单中的 cargo2 必须被物理删除，绝不能保留为 0 件
		_, err = data.db.OrderCargoItem.Get(ctx, cargo2.ID)
		if err == nil || !ent.IsNotFound(err) {
			t.Errorf("完全移出的原货物行 cargo2 必须被物理删除，但仍能查到: %v", err)
		}

		// 检查原订单绝无 0 件残留货物行
		originCargoes, _ := data.db.OrderCargoItem.Query().
			Where(ordercargoitement.OrderIDEQ(fCargo.order.ID)).
			All(ctx)
		for _, c := range originCargoes {
			if c.PackageCount <= 0 {
				t.Errorf("原订单存在件数 <= 0 的货物残留行: %+v", c)
			}
		}

		// 2. 子票已创建新货物行，继承 50 件
		childCargo, err := data.db.OrderCargoItem.Query().
			Where(ordercargoitement.OrderIDEQ(createdOrderID)).
			Only(ctx)
		if err != nil {
			t.Fatalf("查询子订单货物行失败: %v", err)
		}
		if childCargo.PackageCount != 50 || childCargo.GrossWeightKg != 1000.0 || childCargo.VolumeCbm != 8.0 {
			t.Errorf("子订单货物行数据不符合预期: %+v", childCargo)
		}
	})

	// -------------------------------------------------------------------------
	// I. 旧分配全部物理删除且新分配生成全新UUID与映射一致
	// -------------------------------------------------------------------------
	t.Run("旧分配全部物理删除且新分配生成全新UUID与映射一致", func(t *testing.T) {
		fAlloc := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "008")

		oldAlloc1, _ := data.db.SeaCargoAllocation.Query().Where(seacargoallocationent.HouseBillIDEQ(fAlloc.hbl1.ID)).Only(ctx)
		oldAlloc2, _ := data.db.SeaCargoAllocation.Query().Where(seacargoallocationent.HouseBillIDEQ(fAlloc.hbl2.ID)).Only(ctx)

		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fAlloc.order.ID,
			IdempotencyKey:     "split-alloc-recreate-008",
			RequestFingerprint: "fp-alloc-recreate-008",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "res-origin",
					HouseBillIDs:    []uuid.UUID{fAlloc.hbl1.ID},
					DraftFeeIDs:     []uuid.UUID{fAlloc.fee1.ID},
				},
				{
					ClientResultKey: "res-new-1",
					ResultRole:      biz.ResultRoleCreated,
					ClientTargetKey: "res-new-1",
					HouseBillIDs:    []uuid.UUID{fAlloc.hbl2.ID},
				},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fAlloc),
		}

		splitEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err != nil {
			t.Fatalf("ExecuteSplit 分配测试失败: %v", err)
		}

		// 1. 旧分配必须被物理删除
		_, err = data.db.SeaCargoAllocation.Get(ctx, oldAlloc1.ID)
		if err == nil || !ent.IsNotFound(err) {
			t.Errorf("旧分配 1 必须被物理删除: %v", err)
		}
		_, err = data.db.SeaCargoAllocation.Get(ctx, oldAlloc2.ID)
		if err == nil || !ent.IsNotFound(err) {
			t.Errorf("旧分配 2 必须被物理删除: %v", err)
		}

		// 2. 新分配拥有全新 UUID
		newAlloc1, err := data.db.SeaCargoAllocation.Query().Where(seacargoallocationent.HouseBillIDEQ(fAlloc.hbl1.ID)).Only(ctx)
		if err != nil {
			t.Fatalf("查询新分配 1 失败: %v", err)
		}
		if newAlloc1.ID == oldAlloc1.ID {
			t.Errorf("新分配 1 不得复用旧 UUID: %v", newAlloc1.ID)
		}

		newAlloc2, err := data.db.SeaCargoAllocation.Query().Where(seacargoallocationent.HouseBillIDEQ(fAlloc.hbl2.ID)).Only(ctx)
		if err != nil {
			t.Fatalf("查询新分配 2 失败: %v", err)
		}
		if newAlloc2.ID == oldAlloc2.ID {
			t.Errorf("新分配 2 不得复用旧 UUID: %v", newAlloc2.ID)
		}

		// 3. 校验快照映射
		for _, res := range splitEvt.Results {
			var snap map[string]interface{}
			_ = json.Unmarshal([]byte(res.ResultSnapshot), &snap)
			allocMap := snap["allocation_old_new_id_map"].(map[string]interface{})
			if res.ResultRole == biz.ResultRoleOriginal {
				if allocMap[oldAlloc1.ID.String()] != newAlloc1.ID.String() {
					t.Errorf("原票快照中分配映射异常: %+v", allocMap)
				}
			} else {
				if allocMap[oldAlloc2.ID.String()] != newAlloc2.ID.String() {
					t.Errorf("新票快照中分配映射异常: %+v", allocMap)
				}
			}
		}
	})

	// -------------------------------------------------------------------------
	// J. 新票初始Link与内嵌改配事件归属及历史追踪
	// -------------------------------------------------------------------------
	t.Run("新票初始Link与内嵌改配事件归属及历史追踪", func(t *testing.T) {
		fReassignLink := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "009")

		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fReassignLink.order.ID,
			IdempotencyKey:     "split-child-reassign-link-009",
			RequestFingerprint: "fp-child-reassign-link-009",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{
					ClientTargetKey: "res-new-1",
					TargetType:      biz.SplitTargetTypeNew,
					MasterNo:        "MBLEMBEDDEDREASSIGN009",
					IssuerPartnerID: &carrier.ID,
					VesselName:      "TEST SHIP",
					VoyageNo:        "V100",
				},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "res-origin",
					HouseBillIDs:    []uuid.UUID{fReassignLink.hbl1.ID},
					DraftFeeIDs:     []uuid.UUID{fReassignLink.fee1.ID},
				},
				{
					ClientResultKey: "res-new-1",
					ResultRole:      biz.ResultRoleCreated,
					ClientTargetKey: "res-new-1",
					HouseBillIDs:    []uuid.UUID{fReassignLink.hbl2.ID},
				},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fReassignLink),
		}

		splitEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err != nil {
			t.Fatalf("ExecuteSplit 内嵌改配测试失败: %v", err)
		}

		var createdOrderID uuid.UUID
		for _, r := range splitEvt.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
			}
		}

		if len(splitEvt.ReassignmentEventIDs) != 1 {
			t.Fatalf("新票改配应生成 1 条内嵌改配事件: %+v", splitEvt.ReassignmentEventIDs)
		}

		reassignEvt, err := data.db.SeaOrderReassignmentEvent.Get(ctx, splitEvt.ReassignmentEventIDs[0])
		if err != nil {
			t.Fatalf("获取内嵌改配事件失败: %v", err)
		}

		// 1. 改配事件 OrderID 必须属于新票自己
		if reassignEvt.OrderID != createdOrderID {
			t.Errorf("内嵌改配事件 OrderID 必须属于新票: got %v, want %v", reassignEvt.OrderID, createdOrderID)
		}

		// 2. PreviousLinkID 必须属于新票自己的初始 Link，且状态为 ENDED
		prevLink, err := data.db.SeaMasterBillOrderLink.Get(ctx, reassignEvt.PreviousLinkID)
		if err != nil {
			t.Fatalf("获取 PreviousLink 失败: %v", err)
		}
		if prevLink.OrderID != createdOrderID {
			t.Errorf("PreviousLink 必须属于新票: got %v, want %v", prevLink.OrderID, createdOrderID)
		}
		if prevLink.Status != seamasterbillorderlinkent.StatusENDED {
			t.Errorf("PreviousLink 状态应为 ENDED: %v", prevLink.Status)
		}
		if prevLink.MasterBillID != mbl.ID {
			t.Errorf("新票初始 PreviousLink 应指向来源当前母单: %v", prevLink.MasterBillID)
		}

		// 3. TargetLinkID 必须属于新票，且状态为 ACTIVE，指向新母单
		targetLink, err := data.db.SeaMasterBillOrderLink.Get(ctx, reassignEvt.TargetLinkID)
		if err != nil {
			t.Fatalf("获取 TargetLink 失败: %v", err)
		}
		if targetLink.OrderID != createdOrderID {
			t.Errorf("TargetLink 必须属于新票: got %v, want %v", targetLink.OrderID, createdOrderID)
		}
		if targetLink.Status != seamasterbillorderlinkent.StatusACTIVE {
			t.Errorf("TargetLink 状态应为 ACTIVE: %v", targetLink.Status)
		}
		if targetLink.MasterBillID == mbl.ID {
			t.Errorf("TargetLink 应指向新母单，不能是原母单")
		}

		// 4. BeforeSnapshot 与 AfterSnapshot 独立且合法
		if bytes.Equal(reassignEvt.BeforeSnapshot, reassignEvt.AfterSnapshot) {
			t.Errorf("内嵌改配事件 BeforeSnapshot 与 AfterSnapshot 不应相同")
		}
	})

	// -------------------------------------------------------------------------
	// K. Link状态confirmed_at与by及配舱版本递增维护
	// -------------------------------------------------------------------------
	t.Run("Link状态confirmed_at与by及配舱版本递增维护", func(t *testing.T) {
		fLink := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "010")

		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fLink.order.ID,
			IdempotencyKey:     "split-link-confirmed-010",
			RequestFingerprint: "fp-link-confirmed-010",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "res-origin",
					HouseBillIDs:    []uuid.UUID{fLink.hbl1.ID},
					DraftFeeIDs:     []uuid.UUID{fLink.fee1.ID},
				},
				{
					ClientResultKey: "res-new-1",
					ResultRole:      biz.ResultRoleCreated,
					ClientTargetKey: "res-new-1",
					HouseBillIDs:    []uuid.UUID{fLink.hbl2.ID},
				},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fLink),
		}

		splitEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err != nil {
			t.Fatalf("ExecuteSplit Link 状态测试失败: %v", err)
		}

		var createdOrderID uuid.UUID
		for _, r := range splitEvt.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
			}
		}

		// 原订单 Link: CargoAllocationVersion 递增为 2，confirmed_at/by 正确
		origLinkReloaded, _ := data.db.SeaMasterBillOrderLink.Get(ctx, fLink.link.ID)
		if origLinkReloaded.CargoAllocationStatus != seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
			t.Errorf("原票 Link 配舱状态应为 CONFIRMED: %v", origLinkReloaded.CargoAllocationStatus)
		}
		if origLinkReloaded.CargoAllocationVersion != 2 {
			t.Errorf("原票 Link 配舱版本应递增为 2, got %d", origLinkReloaded.CargoAllocationVersion)
		}
		if origLinkReloaded.CargoAllocationConfirmedBy == nil || *origLinkReloaded.CargoAllocationConfirmedBy != user.ID {
			t.Errorf("原票 Link confirmed_by 错误: got %v, want %v", origLinkReloaded.CargoAllocationConfirmedBy, user.ID)
		}
		if origLinkReloaded.CargoAllocationConfirmedAt.IsZero() {
			t.Errorf("原票 Link confirmed_at 不应为空")
		}

		// 新订单 Link: CargoAllocationVersion 从 1 开始，confirmed_at/by 正确
		childLink, _ := data.db.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.OrderIDEQ(createdOrderID), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).
			Only(ctx)
		if childLink.CargoAllocationStatus != seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED {
			t.Errorf("新票 Link 配舱状态应为 CONFIRMED: %v", childLink.CargoAllocationStatus)
		}
		if childLink.CargoAllocationVersion != 1 {
			t.Errorf("新票 Link 配舱版本应为 1, got %d", childLink.CargoAllocationVersion)
		}
		if childLink.CargoAllocationConfirmedBy == nil || *childLink.CargoAllocationConfirmedBy != user.ID {
			t.Errorf("新票 Link confirmed_by 错误: got %v, want %v", childLink.CargoAllocationConfirmedBy, user.ID)
		}
		if childLink.CargoAllocationConfirmedAt.IsZero() {
			t.Errorf("新票 Link confirmed_at 不应为空")
		}
	})

	// -------------------------------------------------------------------------
	// L. FCL未落实计划余量计算与LCL计划清理
	// -------------------------------------------------------------------------
	t.Run("FCL未落实计划余量计算与LCL计划清理", func(t *testing.T) {
		fPlan := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "011")

		// 设置源订单拥有 10 个 40GP 箱计划 (OrderContainerRequest)
		data.db.OrderContainerRequest.Create().
			SetOrderID(fPlan.order.ID).
			SetContainerSpecID(spec.ID).
			SetQuantity(10).
			SaveX(ctx)

		// 扩展 4 个箱后总分配件数为 60 + 40 + 40 = 140 件，更新原货物行以保持守恒
		data.db.OrderCargoItem.UpdateOneID(fPlan.cargoItem.ID).
			SetPackageCount(140).
			SetGrossWeightKg(2800.0).
			SetVolumeCbm(21.0).
			SaveX(ctx)

		// 构造 6 个实际箱：cntr1, cntr2 已有；再建 4 个
		for i := 3; i <= 6; i++ {
			c, err := data.db.OrderContainer.Create().
				SetOrganizationID(org.ID).
				SetOrderID(fPlan.order.ID).
				SetContainerNo("MSKU-EXT-" + uuid.New().String()[:8]).
				SetContainerSpecID(spec.ID).
				SetPackageCount(10).
				SetGrossWeightKg(200.0).
				SetVolumeCbm(1.5).
				SetVersion(1).
				Save(ctx)
			if err != nil {
				t.Fatalf("创建扩展实际箱失败: %v", err)
			}
			// 关联到 allocation 1 (留原票)
			data.db.SeaCargoAllocation.Create().
				SetOrganizationID(org.ID).
				SetOrderID(fPlan.order.ID).
				SetMasterBillOrderLinkID(fPlan.link.ID).
				SetCargoItemID(fPlan.cargoItem.ID).
				SetHouseBillID(fPlan.hbl1.ID).
				SetContainerID(c.ID).
				SetPackageCount(10).
				SetGrossWeightKg("200.000").
				SetVolumeCbm("1.500000").
				SaveX(ctx)
		}

		// 目前原票实际箱分布：cntr1 + 4扩展箱 = 5箱留原票；cntr2 = 1箱去新票。
		// 10 计划 / 6 实际，拆出 1 箱实际箱到新票
		// 原票保留计划数应为: max(5实际, 10原计划 - 1拆出实际) = 9
		// 新票计划数应为: 1实际 = 1
		expPlan := buildFixtureExpectedVersions(fPlan)
		allContainers, _ := data.db.OrderContainer.Query().Where(ordercontainerent.OrderIDEQ(fPlan.order.ID)).All(ctx)
		expPlan.ContainerVersions = make(map[uuid.UUID]uint64)
		for _, c := range allContainers {
			expPlan.ContainerVersions[c.ID] = c.Version
		}

		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fPlan.order.ID,
			IdempotencyKey:     "split-fcl-plan-011",
			RequestFingerprint: "fp-fcl-plan-011",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "res-origin",
					HouseBillIDs:    []uuid.UUID{fPlan.hbl1.ID},
					DraftFeeIDs:     []uuid.UUID{fPlan.fee1.ID},
				},
				{
					ClientResultKey: "res-new-1",
					ResultRole:      biz.ResultRoleCreated,
					ClientTargetKey: "res-new-1",
					HouseBillIDs:    []uuid.UUID{fPlan.hbl2.ID}, // cntr2
				},
			},
			ExpectedVersions: expPlan,
		}

		splitEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err != nil {
			t.Fatalf("ExecuteSplit 箱计划测试失败: %v", err)
		}

		var createdOrderID uuid.UUID
		for _, r := range splitEvt.Results {
			if r.ResultRole == biz.ResultRoleCreated {
				createdOrderID = r.OrderID
			}
		}

		// 验证原票箱计划：原10 - 1拆出 = 9
		origPlan, err := data.db.OrderContainerRequest.Query().
			Where(ordercontainerrequestent.OrderIDEQ(fPlan.order.ID)).
			Only(ctx)
		if err != nil {
			t.Fatalf("查询原票箱计划失败: %v", err)
		}
		if origPlan.Quantity != 9 {
			t.Errorf("原票箱计划未落实余量计算错误: got %d, want 9", origPlan.Quantity)
		}

		// 验证新票箱计划：1 个拆出实际箱 = 1
		childPlan, err := data.db.OrderContainerRequest.Query().
			Where(ordercontainerrequestent.OrderIDEQ(createdOrderID)).
			Only(ctx)
		if err != nil {
			t.Fatalf("查询新票箱计划失败: %v", err)
		}
		if childPlan.Quantity != 1 {
			t.Errorf("新票箱计划计算错误: got %d, want 1", childPlan.Quantity)
		}
	})

	// -------------------------------------------------------------------------
	// M. 完整快照SchemaVersion与多实体主要映射校验
	// -------------------------------------------------------------------------
	t.Run("完整快照SchemaVersion与多实体主要映射校验", func(t *testing.T) {
		fSnap := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "012")

		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fSnap.order.ID,
			IdempotencyKey:     "split-snapshot-schema-012",
			RequestFingerprint: "fp-snapshot-schema-012",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "res-origin",
					HouseBillIDs:    []uuid.UUID{fSnap.hbl1.ID},
					DraftFeeIDs:     []uuid.UUID{fSnap.fee1.ID},
				},
				{
					ClientResultKey: "res-new-1",
					ResultRole:      biz.ResultRoleCreated,
					ClientTargetKey: "res-new-1",
					HouseBillIDs:    []uuid.UUID{fSnap.hbl2.ID},
				},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fSnap),
		}

		splitEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err != nil {
			t.Fatalf("ExecuteSplit 快照测试失败: %v", err)
		}

		// 1. BeforeSnapshot 检查
		var before map[string]interface{}
		if err := json.Unmarshal([]byte(splitEvt.BeforeSnapshot), &before); err != nil {
			t.Fatalf("BeforeSnapshot 不是有效 JSON: %v", err)
		}
		if before["schema_version"] != float64(1) {
			t.Errorf("BeforeSnapshot schema_version 应为 1: got %v", before["schema_version"])
		}
		if before["order"] == nil || before["active_link"] == nil || before["master_bill"] == nil {
			t.Errorf("BeforeSnapshot 缺少关键实体: %+v", before)
		}

		// 2. ConservationSnapshot 检查
		var cons map[string]interface{}
		if err := json.Unmarshal([]byte(splitEvt.ConservationSnapshot), &cons); err != nil {
			t.Fatalf("ConservationSnapshot 不是有效 JSON: %v", err)
		}
		if cons["schema_version"] != float64(1) {
			t.Errorf("ConservationSnapshot schema_version 应为 1: got %v", cons["schema_version"])
		}
		if cons["conservation_passed"] != true {
			t.Errorf("ConservationSnapshot conservation_passed 应为 true")
		}

		// 3. ResultSnapshot 检查
		for _, r := range splitEvt.Results {
			var rSnap map[string]interface{}
			if err := json.Unmarshal([]byte(r.ResultSnapshot), &rSnap); err != nil {
				t.Fatalf("ResultSnapshot 不是有效 JSON: %v", err)
			}
			if rSnap["schema_version"] != float64(1) {
				t.Errorf("ResultSnapshot schema_version 应为 1: got %v", rSnap["schema_version"])
			}
			if rSnap["order_id"] == nil || rSnap["allocation_old_new_id_map"] == nil {
				t.Errorf("ResultSnapshot 缺少关键字段: %+v", rSnap)
			}
		}
	})

	// -------------------------------------------------------------------------
	// N. 候选母单异源航程目标一致允许与目标输入不符阻断
	// -------------------------------------------------------------------------
	t.Run("候选母单异源航程目标一致允许与目标输入不符阻断", func(t *testing.T) {
		fCand := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "013")

		// 建立一个候选 MBL，拥有独立的 TE (不同船名航次)
		candTe, err := data.db.SeaTransportExecution.Create().
			SetOrganizationID(org.ID).
			SetCarrierID(carrier.ID).
			SetVesselName("PACIFIC GLORY").
			SetVoyageNo("2026E").
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建候选TE失败: %v", err)
		}

		candMbl, err := data.db.SeaMasterBill.Create().
			SetOrganizationID(org.ID).
			SetMasterNo("CANDMBLSHARE013").
			SetNormalizedMasterNo("CANDMBLSHARE013").
			SetIssuerPartnerID(carrier.ID).
			SetTransportExecutionID(candTe.ID).
			SetStatus(seamasterbillent.StatusDRAFT).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建候选MBL失败: %v", err)
		}

		candMblVer := candMbl.Version
		candTeVer := candTe.Version

		// Case 1: 目标输入与候选权威 TE 不一致 (船名故意输入 WRONG) -> 必须阻断
		mismatchInput := &biz.SeaOrderSplitInput{
			OrderID:            fCand.order.ID,
			IdempotencyKey:     "split-cand-mismatch-013",
			RequestFingerprint: "fp-cand-mismatch-013",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{
					ClientTargetKey:    "res-new-1",
					TargetType:         biz.SplitTargetTypeCandidate,
					CandidateID:        &candMbl.ID,
					CandidateVersion:   &candMblVer,
					CandidateTEID:      &candTe.ID,
					CandidateTEVersion: &candTeVer,
					IssuerPartnerID:    &carrier.ID,
					VesselName:         "WRONG SHIP NAME", // 不一致
					VoyageNo:           "2026E",
				},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fCand.hbl1.ID}, DraftFeeIDs: []uuid.UUID{fCand.fee1.ID}},
				{ClientResultKey: "res-new-1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new-1", HouseBillIDs: []uuid.UUID{fCand.hbl2.ID}},
			},
			ExpectedVersions: func() *biz.SeaOrderSplitExpectedVersions {
				ev := buildFixtureExpectedVersions(fCand)
				ev.CandidateMBLVersions = map[uuid.UUID]uint64{candMbl.ID: candMbl.Version}
				ev.CandidateTEVersions = map[uuid.UUID]uint64{candTe.ID: candTe.Version}
				return ev
			}(),
		}

		_, err = uc.ExecuteSplit(ctx, org.ID, user.ID, mismatchInput)
		if err == nil {
			t.Fatal("候选输入与权威TE不一致必须阻断")
		}
		if reason := errors.Reason(err); reason != "SEA_ORDER_SPLIT_BLOCKED" {
			t.Errorf("期望错误码 SEA_ORDER_SPLIT_BLOCKED, 实际: %v", reason)
		}
		if md := errors.FromError(err).Metadata; md["reason"] != "CANDIDATE_MBL_INPUT_MISMATCH" {
			t.Errorf("期望原因 CANDIDATE_MBL_INPUT_MISMATCH, 实际: %v", md["reason"])
		}

		// Case 2: 目标输入与候选权威 TE 完全一致 (PACIFIC GLORY / 2026E) -> 允许成功
		matchInput := &biz.SeaOrderSplitInput{
			OrderID:            fCand.order.ID,
			IdempotencyKey:     "split-cand-match-013",
			RequestFingerprint: "fp-cand-match-013",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{
					ClientTargetKey:    "res-new-1",
					TargetType:         biz.SplitTargetTypeCandidate,
					CandidateID:        &candMbl.ID,
					CandidateVersion:   &candMblVer,
					CandidateTEID:      &candTe.ID,
					CandidateTEVersion: &candTeVer,
					IssuerPartnerID:    &carrier.ID,
					CarrierID:          &carrier.ID,
					VesselName:         "PACIFIC GLORY",
					VoyageNo:           "2026E",
				},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fCand.hbl1.ID}, DraftFeeIDs: []uuid.UUID{fCand.fee1.ID}},
				{ClientResultKey: "res-new-1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new-1", HouseBillIDs: []uuid.UUID{fCand.hbl2.ID}},
			},
			ExpectedVersions: func() *biz.SeaOrderSplitExpectedVersions {
				ev := buildFixtureExpectedVersions(fCand)
				ev.CandidateMBLVersions = map[uuid.UUID]uint64{candMbl.ID: candMbl.Version}
				ev.CandidateTEVersions = map[uuid.UUID]uint64{candTe.ID: candTe.Version}
				return ev
			}(),
		}

		wrongTEVersionInput := *matchInput
		wrongTEVersionInput.IdempotencyKey = "split-cand-wrong-te-version-013"
		wrongTEVersionInput.RequestFingerprint = "fp-cand-wrong-te-version-013"
		wrongTEVersions := *matchInput.ExpectedVersions
		wrongTEVersions.CandidateTEVersions = map[uuid.UUID]uint64{uuid.New(): candTe.Version}
		wrongTEVersionInput.ExpectedVersions = &wrongTEVersions
		if _, err = uc.ExecuteSplit(ctx, org.ID, user.ID, &wrongTEVersionInput); errors.Reason(err) != "SEA_ORDER_SPLIT_INVALID_ARGUMENT" {
			t.Fatalf("候选运输执行目标字段与版本字典不一致必须在业务边界阻断，实际错误: %v", err)
		}

		wrongTEID := uuid.New()
		wrongTargetTEInput := *matchInput
		wrongTargetTEInput.IdempotencyKey = "split-cand-wrong-target-te-013"
		wrongTargetTEInput.RequestFingerprint = "fp-cand-wrong-target-te-013"
		wrongTargetTE := *matchInput.Targets[1]
		wrongTargetTE.CandidateTEID = &wrongTEID
		wrongTargetTEInput.Targets = []*biz.SeaOrderSplitTargetInput{matchInput.Targets[0], &wrongTargetTE}
		wrongTargetTEVersions := *matchInput.ExpectedVersions
		wrongTargetTEVersions.CandidateTEVersions = map[uuid.UUID]uint64{wrongTEID: candTe.Version}
		wrongTargetTEInput.ExpectedVersions = &wrongTargetTEVersions
		if _, err = uc.ExecuteSplit(ctx, org.ID, user.ID, &wrongTargetTEInput); errors.Reason(err) != "SEA_ORDER_SPLIT_VERSION_CONFLICT" {
			t.Fatalf("候选目标携带的运输执行 ID 与 MBL 实际关系不一致必须在锁后阻断，实际错误: %v", err)
		}

		wrongMBLVersion := candMbl.Version + 1
		wrongTargetMBLVersionInput := *matchInput
		wrongTargetMBLVersionInput.IdempotencyKey = "split-cand-wrong-target-mbl-version-013"
		wrongTargetMBLVersionInput.RequestFingerprint = "fp-cand-wrong-target-mbl-version-013"
		wrongTargetMBL := *matchInput.Targets[1]
		wrongTargetMBL.CandidateVersion = &wrongMBLVersion
		wrongTargetMBLVersionInput.Targets = []*biz.SeaOrderSplitTargetInput{matchInput.Targets[0], &wrongTargetMBL}
		wrongTargetMBLVersions := *matchInput.ExpectedVersions
		wrongTargetMBLVersions.CandidateMBLVersions = map[uuid.UUID]uint64{candMbl.ID: wrongMBLVersion}
		wrongTargetMBLVersionInput.ExpectedVersions = &wrongTargetMBLVersions
		if _, err = uc.ExecuteSplit(ctx, org.ID, user.ID, &wrongTargetMBLVersionInput); errors.Reason(err) != "SEA_ORDER_SPLIT_VERSION_CONFLICT" {
			t.Fatalf("候选目标携带的 MBL 版本与锁后实际版本不一致必须阻断，实际错误: %v", err)
		}

		_, err = uc.ExecuteSplit(ctx, org.ID, user.ID, matchInput)
		if err != nil {
			t.Fatalf("候选输入与权威TE一致应执行成功: %v", err)
		}
	})

	// -------------------------------------------------------------------------
	// O. 两并发拆票竞争与拆票改配竞争防死锁仅一成功
	// -------------------------------------------------------------------------
	t.Run("两并发拆票竞争与拆票改配竞争防死锁仅一成功", func(t *testing.T) {
		fConc := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "014")

		var wg sync.WaitGroup
		resultsChan := make(chan error, 2)

		inputA := &biz.SeaOrderSplitInput{
			OrderID:            fConc.order.ID,
			IdempotencyKey:     "conc-split-a",
			RequestFingerprint: "fp-conc-a",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fConc.hbl1.ID}, DraftFeeIDs: []uuid.UUID{fConc.fee1.ID}},
				{ClientResultKey: "res-new-1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new-1", HouseBillIDs: []uuid.UUID{fConc.hbl2.ID}},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fConc),
		}

		inputB := &biz.SeaOrderSplitInput{
			OrderID:            fConc.order.ID,
			IdempotencyKey:     "conc-split-b",
			RequestFingerprint: "fp-conc-b",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fConc.hbl1.ID}, DraftFeeIDs: []uuid.UUID{fConc.fee1.ID}},
				{ClientResultKey: "res-new-1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new-1", HouseBillIDs: []uuid.UUID{fConc.hbl2.ID}},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fConc),
		}

		wg.Add(2)
		go func() {
			defer wg.Done()
			_, err := uc.ExecuteSplit(ctx, org.ID, user.ID, inputA)
			resultsChan <- err
		}()
		go func() {
			defer wg.Done()
			_, err := uc.ExecuteSplit(ctx, org.ID, user.ID, inputB)
			resultsChan <- err
		}()

		wg.Wait()
		close(resultsChan)

		var successCount, conflictCount int
		for err := range resultsChan {
			if err == nil {
				successCount++
			} else if errors.Reason(err) == "SEA_ORDER_SPLIT_VERSION_CONFLICT" {
				conflictCount++
			} else {
				t.Errorf("并发拆票返回非预期错误: %v", err)
			}
		}

		if successCount != 1 || conflictCount != 1 {
			t.Errorf("并发拆票竞争结果异常: success=%d, conflict=%d", successCount, conflictCount)
		}
	})

	// -------------------------------------------------------------------------
	// P. 注入审计失败触发全事务回滚无任何残余
	// -------------------------------------------------------------------------
	t.Run("注入审计失败触发全事务回滚无任何残余", func(t *testing.T) {
		fAudit := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "015")

		orderCountBefore, _ := data.db.Order.Query().Count(ctx)
		seqBefore, _ := data.db.NumberSequence.Query().Where(numbersequenceent.RuleIDEQ(orderRule.ID)).Only(ctx)
		var seqValueBefore int64
		if seqBefore != nil {
			seqValueBefore = seqBefore.CurrentValue
		}

		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fAudit.order.ID,
			IdempotencyKey:     "split-audit-fail-015",
			RequestFingerprint: "fp-audit-fail-015",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fAudit.hbl1.ID}, DraftFeeIDs: []uuid.UUID{fAudit.fee1.ID}},
				{ClientResultKey: "res-new-1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new-1", HouseBillIDs: []uuid.UUID{fAudit.hbl2.ID}},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fAudit),
		}

		// 注入审计失败 Context
		injectedCtx := biz.WithInjectedAuditFailure(ctx)
		_, err := uc.ExecuteSplit(injectedCtx, org.ID, user.ID, splitInput)
		if err == nil {
			t.Fatal("注入审计失败时必须返回错误")
		}

		// 验证 100% 回滚，无任何残余
		orderCountAfter, _ := data.db.Order.Query().Count(ctx)
		if orderCountAfter != orderCountBefore {
			t.Errorf("审计失败回滚后订单总数不应增加: before=%d, after=%d", orderCountBefore, orderCountAfter)
		}

		seqAfter, _ := data.db.NumberSequence.Query().Where(numbersequenceent.RuleIDEQ(orderRule.ID)).Only(ctx)
		var seqValueAfter int64
		if seqAfter != nil {
			seqValueAfter = seqAfter.CurrentValue
		}
		if seqValueAfter != seqValueBefore {
			t.Errorf("审计失败回滚后订单序列号不应增加: before=%d, after=%d", seqValueBefore, seqValueAfter)
		}

		eventExists, _ := data.db.SeaOrderSplitEvent.Query().
			Where(seaorderspliteventent.IdempotencyKeyEQ("split-audit-fail-015")).
			Exist(ctx)
		if eventExists {
			t.Errorf("审计失败回滚后不应残留拆票事件")
		}
	})

	// -------------------------------------------------------------------------
	// Q. 附件指纹版本集合变化冲突拦截
	// -------------------------------------------------------------------------
	t.Run("附件指纹版本集合变化冲突拦截", func(t *testing.T) {
		fAtt := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "016")

		expAtt := buildFixtureExpectedVersions(fAtt)
		expAtt.AttachmentReferenceFingerprint = "wrong-mismatch-fingerprint" // 故意错误的指纹

		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            fAtt.order.ID,
			IdempotencyKey:     "split-att-conflict-016",
			RequestFingerprint: "fp-att-conflict-016",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{ClientTargetKey: "res-new-1", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fAtt.hbl1.ID}, DraftFeeIDs: []uuid.UUID{fAtt.fee1.ID}},
				{ClientResultKey: "res-new-1", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new-1", HouseBillIDs: []uuid.UUID{fAtt.hbl2.ID}},
			},
			ExpectedVersions: expAtt,
		}

		_, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err == nil {
			t.Fatal("附件指纹冲突时必须被拦截")
		}
		if reason := errors.Reason(err); reason != "SEA_ORDER_SPLIT_VERSION_CONFLICT" {
			t.Errorf("期望错误码 SEA_ORDER_SPLIT_VERSION_CONFLICT, 实际: %v", reason)
		}
	})

	// -------------------------------------------------------------------------
	// R. 独立HOUSE与DIRECT改配关系正确流转
	// -------------------------------------------------------------------------
	t.Run("独立HOUSE与DIRECT改配关系正确流转", func(t *testing.T) {
		fHouse := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "017")

		// 1. 独立 HOUSE 改配
		reasEvt, err := uc.ExecuteReassignment(ctx, org.ID, user.ID, &biz.SeaOrderReassignmentInput{
			OrderID:            fHouse.order.ID,
			IdempotencyKey:     "reas-house-017",
			RequestFingerprint: "fp-house-017",
			Reason:             "船期变更",
			ResponsibilityType: biz.ResponsibilityTypeCarrier,
			Target: &biz.SeaOrderReassignmentTargetInput{
				TargetType:      biz.SplitTargetTypeNew,
				MasterNo:        "NEWREASMBL017",
				IssuerPartnerID: &carrier.ID,
				VesselName:      "NEW SHIP",
				VoyageNo:        "999S",
			},
			ExpectedOrderVersion: fHouse.order.Version,
			ExpectedLinkVersion:  fHouse.link.Version,
		})
		if err != nil {
			t.Fatalf("HOUSE 改配失败: %v", err)
		}
		if reasEvt == nil {
			t.Fatal("改配事件未生成")
		}

		// 验证旧 Link 结束，新 Link 建立
		prevLink, _ := data.db.SeaMasterBillOrderLink.Get(ctx, reasEvt.PreviousLinkID)
		if prevLink.Status != seamasterbillorderlinkent.StatusENDED {
			t.Errorf("旧 Link 应已结束: %v", prevLink.Status)
		}
		targetLink, _ := data.db.SeaMasterBillOrderLink.Get(ctx, reasEvt.TargetLinkID)
		if targetLink.Status != seamasterbillorderlinkent.StatusACTIVE {
			t.Errorf("新 Link 应为 ACTIVE: %v", targetLink.Status)
		}
		// HBL 迁移至新 MBL
		hbl1Check, _ := data.db.SeaHouseBill.Get(ctx, fHouse.hbl1.ID)
		if hbl1Check.MasterBillID != targetLink.MasterBillID {
			t.Errorf("HBL1 MasterBillID 未更新至新 MBL: got %v, want %v", hbl1Check.MasterBillID, targetLink.MasterBillID)
		}
	})

	// -------------------------------------------------------------------------
	// R.1 独立改配候选 MBL/TE 身份与版本锁后精确重验
	// -------------------------------------------------------------------------
	t.Run("独立改配候选MBL与TE身份版本锁后精确重验", func(t *testing.T) {
		fixture := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "021")
		candidateTE, err := data.db.SeaTransportExecution.Create().
			SetOrganizationID(org.ID).
			SetCarrierID(carrier.ID).
			SetVesselName("CANDIDATE SHIP").
			SetVoyageNo("020E").
			SetVersion(3).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建独立改配候选 TE 失败: %v", err)
		}
		candidateMBL, err := data.db.SeaMasterBill.Create().
			SetOrganizationID(org.ID).
			SetMasterNo("CANDIDATEREAS020").
			SetNormalizedMasterNo("CANDIDATEREAS020").
			SetIssuerPartnerID(carrier.ID).
			SetTransportExecutionID(candidateTE.ID).
			SetStatus(seamasterbillent.StatusDRAFT).
			SetVersion(4).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建独立改配候选 MBL 失败: %v", err)
		}

		candidateMBLVersion := candidateMBL.Version
		candidateTEVersion := candidateTE.Version
		wrongTEID := uuid.New()
		wrongInput := &biz.SeaOrderReassignmentInput{
			OrderID:            fixture.order.ID,
			IdempotencyKey:     "reas-candidate-wrong-te-020",
			RequestFingerprint: "fp-reas-candidate-wrong-te-020",
			Reason:             "验证候选运输执行身份",
			ResponsibilityType: biz.ResponsibilityTypeCarrier,
			Target: &biz.SeaOrderReassignmentTargetInput{
				TargetType:         biz.SplitTargetTypeCandidate,
				CandidateID:        &candidateMBL.ID,
				CandidateVersion:   &candidateMBLVersion,
				CandidateTEID:      &wrongTEID,
				CandidateTEVersion: &candidateTEVersion,
				IssuerPartnerID:    &carrier.ID,
			},
			ExpectedOrderVersion:        fixture.order.Version,
			ExpectedLinkVersion:         fixture.link.Version,
			ExpectedCandidateMBLVersion: &candidateMBLVersion,
			ExpectedCandidateTEVersion:  &candidateTEVersion,
		}
		if _, err = uc.ExecuteReassignment(ctx, org.ID, user.ID, wrongInput); errors.Reason(err) != "SEA_ORDER_REASSIGNMENT_VERSION_CONFLICT" {
			t.Fatalf("候选 TE ID 与 MBL 实际关系不一致必须锁后阻断，实际错误: %v", err)
		}

		validInput := *wrongInput
		validInput.IdempotencyKey = "reas-candidate-valid-020"
		validInput.RequestFingerprint = "fp-reas-candidate-valid-020"
		validTarget := *wrongInput.Target
		validTarget.CandidateTEID = &candidateTE.ID
		validTarget.MasterNo = candidateMBL.MasterNo
		validTarget.IssuerPartnerID = &carrier.ID
		validTarget.CarrierID = &carrier.ID
		validTarget.VesselName = candidateTE.VesselName
		validTarget.VoyageNo = candidateTE.VoyageNo
		validInput.Target = &validTarget
		event, err := uc.ExecuteReassignment(ctx, org.ID, user.ID, &validInput)
		if err != nil {
			t.Fatalf("完整一致的候选 MBL/TE 应允许独立改配: %v", err)
		}
		if event.TargetMasterBillID != candidateMBL.ID || event.TargetTransportExecutionID != candidateTE.ID {
			t.Fatalf("独立改配目标身份错误: %+v", event)
		}
	})

	// -------------------------------------------------------------------------
	// S. 同一NEW目标key被多票结果共享只建一张MBL/TE且子票继承完整白名单与Lineage反查
	// -------------------------------------------------------------------------
	t.Run("同一NEW目标key被多票结果共享只建一张MBL且子票继承完整白名单与Lineage反查", func(t *testing.T) {
		// 1. 创建基础数据条目：服务类型与品类
		stItem, err := data.db.MasterDataItem.Create().
			SetOrganizationID(org.ID).
			SetKind(masterdataitement.KindServiceType).
			SetCode("ST-EXP-" + uuid.New().String()[:6]).
			SetName("出口清关派送").
			SetSortOrder(1).
			SetEnabled(true).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建服务类型失败: %v", err)
		}
		ccItem, err := data.db.MasterDataItem.Create().
			SetOrganizationID(org.ID).
			SetKind(masterdataitement.KindCargoCategory).
			SetCode("CC-ELEC-" + uuid.New().String()[:6]).
			SetName("带电产品").
			SetSortOrder(1).
			SetEnabled(true).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建货物类别失败: %v", err)
		}

		// 2. 创建源订单并填充完整白名单字段
		sourceOrder, err := data.db.Order.Create().
			SetOrganizationID(org.ID).
			SetOrderNo("SE20260903998").
			SetCustomerID(customer.ID).
			SetBusinessType(orderent.BusinessTypeSE).
			SetCustomerReferenceNo("PO-MULTI-998").
			SetInternalReferenceNo("INT-SECRET-998").
			SetTradeDirection(orderent.TradeDirectionExport).
			SetTradeTerm(orderent.TradeTermFOB).
			SetPaymentTerm(orderent.PaymentTermPREPAID).
			SetShipmentType(orderent.ShipmentTypeFCL).
			SetContainerOwnership(orderent.ContainerOwnershipCOC).
			SetShipmentMode(orderent.ShipmentModeTRADITIONAL_FORWARDING).
			SetShipperShortName("发货人简称").
			SetConsigneeShortName("收货人简称").
			SetBookingAgentID(carrier.ID).
			SetForeignAgentID(carrier.ID).
			SetShippingAgentID(carrier.ID).
			SetNotes("通用订单备注").
			SetBookingNotes("订舱专项备注").
			SetOperationNotes("操作专项备注").
			SetAllocationNotes("配舱初始备注").
			SetFlowStatus(orderent.FlowStatusBOOKED).
			SetTerminationStatus(orderent.TerminationStatusACTIVE).
			SetClosureStatus(orderent.ClosureStatusOPEN).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建源订单失败: %v", err)
		}

		// 关联服务类型与品类
		err = data.WithTx(ctx, func(tx *ent.Tx) error {
			return replaceOrderSelections(ctx, tx, sourceOrder.ID, []uuid.UUID{stItem.ID}, []uuid.UUID{ccItem.ID})
		})
		if err != nil {
			t.Fatalf("关联服务类型品类失败: %v", err)
		}

		// 关联人员
		_, err = data.db.OrderPersonnel.Create().
			SetOrganizationID(org.ID).
			SetOrderID(sourceOrder.ID).
			SetUserID(user.ID).
			SetRole(orderpersonnelent.RoleOPERATOR).
			Save(ctx)
		if err != nil {
			t.Fatalf("绑定人员失败: %v", err)
		}

		// 关联提成归属快照
		_, err = data.db.OrderCommissionAttribution.Create().
			SetOrganizationID(org.ID).
			SetOrderID(sourceOrder.ID).
			SetCustomerID(customer.ID).
			SetSourceAssignmentID(uuid.Must(uuid.NewV7())).
			SetEmployeeID(user.ID).
			SetEmployeeName("拆票测试操作员").
			SetPersonnelRole("OPERATOR").
			SetAttributedAt(time.Now().UTC()).
			Save(ctx)
		if err != nil {
			t.Fatalf("绑定提成归属失败: %v", err)
		}

		// 关联组织标签
		tagRes, err := data.db.EnterpriseResource.Create().
			SetOrganizationID(org.ID).
			SetResourceType("TAG").
			SetShortName("订单标签").
			SetCreatedBy(user.ID).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建企业标签资源失败: %v", err)
		}
		_, err = data.db.OrderEnterpriseTag.Create().
			SetOrganizationID(org.ID).
			SetOrderID(sourceOrder.ID).
			SetTagResourceID(tagRes.ID).
			Save(ctx)
		if err != nil {
			t.Fatalf("绑定组织标签失败: %v", err)
		}

		// 创建源主单关联 (CONFIRMED)
		srcNow := time.Now().UTC()
		srcLink, err := data.db.SeaMasterBillOrderLink.Create().
			SetOrganizationID(org.ID).
			SetOrderID(sourceOrder.ID).
			SetMasterBillID(mbl.ID).
			SetDocumentStructure(seamasterbillorderlinkent.DocumentStructureHOUSE).
			SetCargoAllocationStatus(seamasterbillorderlinkent.CargoAllocationStatusCONFIRMED).
			SetCargoAllocationVersion(1).
			SetCargoAllocationConfirmedAt(srcNow).
			SetCargoAllocationConfirmedBy(user.ID).
			SetStatus(seamasterbillorderlinkent.StatusACTIVE).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建源主单关联失败: %v", err)
		}

		// 创建货物与箱
		cargoItem, err := data.db.OrderCargoItem.Create().
			SetOrganizationID(org.ID).
			SetOrderID(sourceOrder.ID).
			SetCargoName("共享MBL测试货物").
			SetPackageCount(100).
			SetGrossWeightKg(2000.0).
			SetVolumeCbm(15.0).
			SetVersion(1).
			Save(ctx)
		if err != nil {
			t.Fatalf("创建货物失败: %v", err)
		}

		c1, _ := data.db.OrderContainer.Create().SetOrganizationID(org.ID).SetOrderID(sourceOrder.ID).SetContainerNo("SHAR001").SetContainerSpecID(spec.ID).SetPackageCount(40).SetGrossWeightKg(800.0).SetVolumeCbm(6.0).SetVersion(1).Save(ctx)
		c2, _ := data.db.OrderContainer.Create().SetOrganizationID(org.ID).SetOrderID(sourceOrder.ID).SetContainerNo("SHAR002").SetContainerSpecID(spec.ID).SetPackageCount(30).SetGrossWeightKg(600.0).SetVolumeCbm(4.5).SetVersion(1).Save(ctx)
		c3, _ := data.db.OrderContainer.Create().SetOrganizationID(org.ID).SetOrderID(sourceOrder.ID).SetContainerNo("SHAR003").SetContainerSpecID(spec.ID).SetPackageCount(30).SetGrossWeightKg(600.0).SetVolumeCbm(4.5).SetVersion(1).Save(ctx)

		h1, _ := data.db.SeaHouseBill.Create().SetOrganizationID(org.ID).SetOrderID(sourceOrder.ID).SetMasterBillID(mbl.ID).SetHouseNo("SH-HBL-1").SetNormalizedHouseNo("SH-HBL-1").SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).SetIssuerOrganizationID(org.ID).SetStatus(seahousebillent.StatusDRAFT).SetVersion(1).Save(ctx)
		h2, _ := data.db.SeaHouseBill.Create().SetOrganizationID(org.ID).SetOrderID(sourceOrder.ID).SetMasterBillID(mbl.ID).SetHouseNo("SH-HBL-2").SetNormalizedHouseNo("SH-HBL-2").SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).SetIssuerOrganizationID(org.ID).SetStatus(seahousebillent.StatusDRAFT).SetVersion(1).Save(ctx)
		h3, _ := data.db.SeaHouseBill.Create().SetOrganizationID(org.ID).SetOrderID(sourceOrder.ID).SetMasterBillID(mbl.ID).SetHouseNo("SH-HBL-3").SetNormalizedHouseNo("SH-HBL-3").SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).SetIssuerOrganizationID(org.ID).SetStatus(seahousebillent.StatusDRAFT).SetVersion(1).Save(ctx)

		data.db.SeaCargoAllocation.Create().SetOrganizationID(org.ID).SetOrderID(sourceOrder.ID).SetMasterBillOrderLinkID(srcLink.ID).SetCargoItemID(cargoItem.ID).SetHouseBillID(h1.ID).SetContainerID(c1.ID).SetPackageCount(40).SetGrossWeightKg("800.000").SetVolumeCbm("6.000000").SaveX(ctx)
		data.db.SeaCargoAllocation.Create().SetOrganizationID(org.ID).SetOrderID(sourceOrder.ID).SetMasterBillOrderLinkID(srcLink.ID).SetCargoItemID(cargoItem.ID).SetHouseBillID(h2.ID).SetContainerID(c2.ID).SetPackageCount(30).SetGrossWeightKg("600.000").SetVolumeCbm("4.500000").SaveX(ctx)
		data.db.SeaCargoAllocation.Create().SetOrganizationID(org.ID).SetOrderID(sourceOrder.ID).SetMasterBillOrderLinkID(srcLink.ID).SetCargoItemID(cargoItem.ID).SetHouseBillID(h3.ID).SetContainerID(c3.ID).SetPackageCount(30).SetGrossWeightKg("600.000").SetVolumeCbm("4.500000").SaveX(ctx)

		// 3. 执行拆票：res-origin 留当前 MBL，res-new-1 和 res-new-2 共同指向同一个 NEW 目标 "target-shared-new"
		splitInput := &biz.SeaOrderSplitInput{
			OrderID:            sourceOrder.ID,
			IdempotencyKey:     "split-shared-new-001",
			RequestFingerprint: "fp-shared-new-001",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "target-current", TargetType: biz.SplitTargetTypeCurrent},
				{
					ClientTargetKey: "target-shared-new",
					TargetType:      biz.SplitTargetTypeNew,
					MasterNo:        "SHAREDMBL88888",
					IssuerPartnerID: &carrier.ID,
					VesselName:      "SHARED VESSEL ONE",
					VoyageNo:        "SH001W",
				},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{
					ClientResultKey: "res-origin",
					ResultRole:      biz.ResultRoleOriginal,
					ClientTargetKey: "target-current",
					HouseBillIDs:    []uuid.UUID{h1.ID},
				},
				{
					ClientResultKey:     "res-new-1",
					ResultRole:          biz.ResultRoleCreated,
					ClientTargetKey:     "target-shared-new",
					HouseBillIDs:        []uuid.UUID{h2.ID},
					InternalReferenceNo: nil,
				},
				{
					ClientResultKey:     "res-new-2",
					ResultRole:          biz.ResultRoleCreated,
					ClientTargetKey:     "target-shared-new",
					HouseBillIDs:        []uuid.UUID{h3.ID},
					InternalReferenceNo: strPtr("CHILD-INT-002"),
				},
			},
			ExpectedVersions: &biz.SeaOrderSplitExpectedVersions{
				OrderVersion:      sourceOrder.Version,
				LinkVersion:       srcLink.Version,
				AllocationVersion: 1,
				HouseBillVersions: map[uuid.UUID]uint64{
					h1.ID: h1.Version,
					h2.ID: h2.Version,
					h3.ID: h3.Version,
				},
				CargoItemVersions: map[uuid.UUID]uint64{
					cargoItem.ID: cargoItem.Version,
				},
				ContainerVersions: map[uuid.UUID]uint64{
					c1.ID: c1.Version,
					c2.ID: c2.Version,
					c3.ID: c3.Version,
				},
				FeeVersions:                    map[uuid.UUID]uint64{},
				AttachmentReferenceFingerprint: biz.ComputeAttachmentFingerprint(nil),
			},
		}

		splitEvt, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitInput)
		if err != nil {
			t.Fatalf("共享NEW目标拆票失败: %v", err)
		}
		if len(splitEvt.Results) != 3 {
			t.Fatalf("拆票结果数量应为 3, 实际: %d", len(splitEvt.Results))
		}

		// 4. 断言 Item A：数据库中只创建了 1 张 MBL 和 1 条 TE
		mblCount, err := data.db.SeaMasterBill.Query().
			Where(
				seamasterbillent.OrganizationIDEQ(org.ID),
				seamasterbillent.MasterNoEQ("SHAREDMBL88888"),
			).Count(ctx)
		if err != nil || mblCount != 1 {
			t.Errorf("共享NEW目标只应创建 1 张 MBL: got count=%d, err=%v", mblCount, err)
		}
		teCount, err := data.db.SeaTransportExecution.Query().
			Where(
				seatransportexecutionent.OrganizationIDEQ(org.ID),
				seatransportexecutionent.VesselNameEQ("SHARED VESSEL ONE"),
				seatransportexecutionent.VoyageNoEQ("SH001W"),
			).Count(ctx)
		if err != nil || teCount != 1 {
			t.Errorf("共享NEW目标只应创建 1 条 TE: got count=%d, err=%v", teCount, err)
		}

		sharedMbl, err := data.db.SeaMasterBill.Query().
			Where(seamasterbillent.MasterNoEQ("SHAREDMBL88888")).
			Only(ctx)
		if err != nil {
			t.Fatalf("查询共享MBL失败: %v", err)
		}

		var newOrderID1, newOrderID2 uuid.UUID
		for _, r := range splitEvt.Results {
			if r.ClientResultKey == "res-new-1" {
				newOrderID1 = r.OrderID
			} else if r.ClientResultKey == "res-new-2" {
				newOrderID2 = r.OrderID
			}
		}

		// 两个新票的活动 Link 都指向同一 sharedMbl
		link1, err := data.db.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.OrderIDEQ(newOrderID1), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).
			Only(ctx)
		if err != nil || link1.MasterBillID != sharedMbl.ID {
			t.Errorf("新票1 Link未指向共享MBL: got %v, want %v, err=%v", link1.MasterBillID, sharedMbl.ID, err)
		}
		link2, err := data.db.SeaMasterBillOrderLink.Query().
			Where(seamasterbillorderlinkent.OrderIDEQ(newOrderID2), seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE)).
			Only(ctx)
		if err != nil || link2.MasterBillID != sharedMbl.ID {
			t.Errorf("新票2 Link未指向共享MBL: got %v, want %v, err=%v", link2.MasterBillID, sharedMbl.ID, err)
		}

		// 5. 断言 Item B: 子票白名单继承完整性
		o1, _ := data.db.Order.Get(ctx, newOrderID1)
		if o1.CustomerID != sourceOrder.CustomerID {
			t.Errorf("子票1委托单位未继承: got %v, want %v", o1.CustomerID, sourceOrder.CustomerID)
		}
		if o1.BusinessType != sourceOrder.BusinessType {
			t.Errorf("子票1业务类型未继承: got %v, want %v", o1.BusinessType, sourceOrder.BusinessType)
		}
		if o1.TradeDirection != sourceOrder.TradeDirection || o1.TradeTerm != sourceOrder.TradeTerm || o1.PaymentTerm != sourceOrder.PaymentTerm {
			t.Errorf("子票1贸易与付款条款未继承")
		}
		if *o1.ContainerOwnership != *sourceOrder.ContainerOwnership {
			t.Errorf("子票1箱主类型未继承: got %v, want %v", *o1.ContainerOwnership, *sourceOrder.ContainerOwnership)
		}
		if *o1.ShipmentMode != *sourceOrder.ShipmentMode {
			t.Errorf("子票1运输模式未继承: got %v, want %v", *o1.ShipmentMode, *sourceOrder.ShipmentMode)
		}
		if *o1.BookingAgentID != *sourceOrder.BookingAgentID || *o1.ForeignAgentID != *sourceOrder.ForeignAgentID || *o1.ShippingAgentID != *sourceOrder.ShippingAgentID {
			t.Errorf("子票1代理人未继承")
		}
		if o1.CustomerReferenceNo != "PO-MULTI-998" {
			t.Errorf("子票1客户参考号未继承: got %s", o1.CustomerReferenceNo)
		}
		if o1.InternalReferenceNo != "" {
			t.Errorf("子票1未传内部单号应为空, 实际为: %s", o1.InternalReferenceNo)
		}
		o2, _ := data.db.Order.Get(ctx, newOrderID2)
		if o2.InternalReferenceNo != "CHILD-INT-002" {
			t.Errorf("子票2独立传入内部单号未生效: got %s", o2.InternalReferenceNo)
		}
		// 航程平铺投影来自新 TE
		if o1.VesselVoyage != "SHARED VESSEL ONE / SH001W" {
			t.Errorf("子票1航程未平铺新TE: got %s", o1.VesselVoyage)
		}
		// 服务类型与品类
		stCount, _ := o1.QueryServiceTypes().Count(ctx)
		ccCount, _ := o1.QueryCargoCategories().Count(ctx)
		if stCount != 1 || ccCount != 1 {
			t.Errorf("子票1服务类型或品类未复制: st=%d, cc=%d", stCount, ccCount)
		}
		// 人员与提成快照与标签
		pCount, _ := data.db.OrderPersonnel.Query().Where(orderpersonnelent.OrderIDEQ(newOrderID1)).Count(ctx)
		if pCount != 1 {
			t.Errorf("子票1人员复制失败: %d", pCount)
		}
		attrCount, _ := data.db.OrderCommissionAttribution.Query().Where(ordercommissionattributionent.OrderIDEQ(newOrderID1)).Count(ctx)
		if attrCount != 1 {
			t.Errorf("子票1提成归属复制失败: %d", attrCount)
		}
		tagCount, _ := data.db.OrderEnterpriseTag.Query().Where(orderenterprisetagent.OrderIDEQ(newOrderID1)).Count(ctx)
		if tagCount != 1 {
			t.Errorf("子票1组织标签复制失败: %d", tagCount)
		}

		// 6. 断言 Item E: 子票历史 Lineage 反查
		// 从新票 1 查询变更历史列表 (包含出生时的拆票事件和内嵌改配事件)
		hist1, totalHist1, err := uc.ListChangeEvents(ctx, org.ID, newOrderID1, 1, 10)
		if err != nil || totalHist1 != 2 || len(hist1) != 2 {
			t.Fatalf("新票1反查历史失败: total=%d, err=%v", totalHist1, err)
		}
		foundSplit1 := false
		foundReassign1 := false
		for _, h := range hist1 {
			if h.EventType == biz.EventTypeSplit && h.ID == splitEvt.ID {
				foundSplit1 = true
				if h.SplitSummary == nil || len(h.SplitSummary.Results) != 3 {
					t.Fatalf("拆票历史汇总缺少结果明细: %+v", h.SplitSummary)
				}
				for _, result := range h.SplitSummary.Results {
					if result.FinalMasterNo == "" || result.PackageCount <= 0 || !result.GrossWeightKg.IsPositive() || !result.VolumeCbm.IsPositive() {
						t.Fatalf("拆票历史快照不得静默解码为零值: %+v", result)
					}
				}
			}
			if h.EventType == biz.EventTypeReassignment {
				foundReassign1 = true
			}
		}
		if !foundSplit1 {
			t.Errorf("新票1历史列表中未找到拆票事件 %v", splitEvt.ID)
		}
		if !foundReassign1 {
			t.Errorf("新票1历史列表中未找到内嵌改配事件")
		}

		// 从新票 1 获取变更事件详情
		evtDetail1, err := uc.GetChangeEvent(ctx, org.ID, newOrderID1, splitEvt.ID, "SPLIT")
		if err != nil || evtDetail1 == nil || evtDetail1.SplitSummary == nil {
			t.Fatalf("新票1获取拆票事件详情失败: %v", err)
		}
		if evtDetail1.SplitSummary.SourceOrderID != sourceOrder.ID {
			t.Errorf("新票1反查源订单ID不匹配: got %v, want %v", evtDetail1.SplitSummary.SourceOrderID, sourceOrder.ID)
		}
		if len(evtDetail1.SplitSummary.Results) != 3 {
			t.Errorf("新票1反查结果数量应为 3: got %d", len(evtDetail1.SplitSummary.Results))
		}

		unrelatedOrder, err := data.db.Order.Create().
			SetOrganizationID(org.ID).
			SetOrderNo("SE20260903997").
			SetCustomerID(customer.ID).
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
			t.Fatalf("创建无关订单失败: %v", err)
		}
		if _, err := uc.GetChangeEvent(ctx, org.ID, unrelatedOrder.ID, splitEvt.ID, "SPLIT"); err == nil {
			t.Fatal("同组织无关订单不得读取不属于自己的拆票事件")
		}

		// 从新票 2 同样能反查
		hist2, totalHist2, err := uc.ListChangeEvents(ctx, org.ID, newOrderID2, 1, 10)
		if err != nil || totalHist2 != 2 {
			t.Fatalf("新票2反查历史失败: total=%d, err=%v", totalHist2, err)
		}
		foundSplit2 := false
		for _, h := range hist2 {
			if h.EventType == biz.EventTypeSplit && h.ID == splitEvt.ID {
				foundSplit2 = true
			}
		}
		if !foundSplit2 {
			t.Errorf("新票2历史列表中未找到拆票事件 %v", splitEvt.ID)
		}
		evtDetail2, err := uc.GetChangeEvent(ctx, org.ID, newOrderID2, splitEvt.ID, "SPLIT")
		if err != nil || evtDetail2 == nil || evtDetail2.SplitSummary == nil {
			t.Fatalf("新票2获取拆票事件详情失败: %v", err)
		}
		if evtDetail2.SplitSummary.SourceOrderID != sourceOrder.ID {
			t.Errorf("新票2反查源订单ID不匹配: got %v, want %v", evtDetail2.SplitSummary.SourceOrderID, sourceOrder.ID)
		}
	})

	// -------------------------------------------------------------------------
	// T. NEW 改配或拆票目标的 master_no 严格校验与合法小写规范化
	// -------------------------------------------------------------------------
	t.Run("NEW目标MasterNo格式严格校验与小写转大写规范化", func(t *testing.T) {
		fVal := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "020")

		// 1. 拆票目标 master_no 含连字符 -> 精确返回 ErrSeaMasterBillInvalidArgument，全事务回滚无残留
		splitHyphenInput := &biz.SeaOrderSplitInput{
			OrderID:            fVal.order.ID,
			IdempotencyKey:     "split-hyphen-val-020",
			RequestFingerprint: "fp-hyphen-val-020",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{
					ClientTargetKey: "res-new-hyphen",
					TargetType:      biz.SplitTargetTypeNew,
					MasterNo:        "INVALID-HYPHEN-MBL",
					IssuerPartnerID: &carrier.ID,
					VesselName:      "TEST SHIP",
					VoyageNo:        "V001",
				},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fVal.hbl1.ID}, DraftFeeIDs: []uuid.UUID{fVal.fee1.ID}},
				{ClientResultKey: "res-new-hyphen", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new-hyphen", HouseBillIDs: []uuid.UUID{fVal.hbl2.ID}},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fVal),
		}
		_, err := uc.ExecuteSplit(ctx, org.ID, user.ID, splitHyphenInput)
		if err == nil {
			t.Fatal("含连字符的 MasterNo 应返回错误")
		}
		if reason := errors.Reason(err); reason != "SEA_MASTER_BILL_INVALID_ARGUMENT" {
			t.Errorf("期望错误码 SEA_MASTER_BILL_INVALID_ARGUMENT, 实际: %v", reason)
		}
		// 断言无任何残留
		hyphenMblCount, _ := data.db.SeaMasterBill.Query().Where(seamasterbillent.OrganizationIDEQ(org.ID), seamasterbillent.MasterNoEQ("INVALID-HYPHEN-MBL")).Count(ctx)
		if hyphenMblCount != 0 {
			t.Errorf("失败事务不应残留 MBL")
		}
		hyphenTeCount, _ := data.db.SeaTransportExecution.Query().Where(seatransportexecutionent.OrganizationIDEQ(org.ID), seatransportexecutionent.VoyageNoEQ("V001")).Count(ctx)
		if hyphenTeCount != 0 {
			t.Errorf("失败事务不应残留 TE")
		}
		hyphenSplitEvtCount, _ := data.db.SeaOrderSplitEvent.Query().Where(seaorderspliteventent.IdempotencyKeyEQ("split-hyphen-val-020")).Count(ctx)
		if hyphenSplitEvtCount != 0 {
			t.Errorf("失败事务不应残留 SplitEvent")
		}

		// 2. 改配目标 master_no 含首尾空格 -> 精确返回 ErrSeaMasterBillInvalidArgument，全事务回滚无残留
		reasSpaceInput := &biz.SeaOrderReassignmentInput{
			OrderID:            fVal.order.ID,
			IdempotencyKey:     "reas-space-val-020",
			RequestFingerprint: "fp-space-val-020",
			Reason:             "测试空格提单号",
			ResponsibilityType: biz.ResponsibilityTypeCustomer,
			Target: &biz.SeaOrderReassignmentTargetInput{
				TargetType:      biz.SplitTargetTypeNew,
				MasterNo:        "  SPACEPREFIXMBL99  ",
				IssuerPartnerID: &carrier.ID,
				VesselName:      "TEST SHIP",
				VoyageNo:        "V002",
			},
			ExpectedOrderVersion: fVal.order.Version,
			ExpectedLinkVersion:  fVal.link.Version,
		}
		_, err = uc.ExecuteReassignment(ctx, org.ID, user.ID, reasSpaceInput)
		if err == nil {
			t.Fatal("含首尾空格的 MasterNo 应返回错误")
		}
		if reason := errors.Reason(err); reason != "SEA_MASTER_BILL_INVALID_ARGUMENT" {
			t.Errorf("期望错误码 SEA_MASTER_BILL_INVALID_ARGUMENT, 实际: %v", reason)
		}
		spaceMblCount, _ := data.db.SeaMasterBill.Query().Where(seamasterbillent.OrganizationIDEQ(org.ID), seamasterbillent.MasterNoContains("SPACEPREFIXMBL99")).Count(ctx)
		if spaceMblCount != 0 {
			t.Errorf("失败事务不应残留 MBL")
		}
		spaceTeCount, _ := data.db.SeaTransportExecution.Query().Where(seatransportexecutionent.OrganizationIDEQ(org.ID), seatransportexecutionent.VoyageNoEQ("V002")).Count(ctx)
		if spaceTeCount != 0 {
			t.Errorf("失败事务不应残留 TE")
		}
		spaceReasEvtCount, _ := data.db.SeaOrderReassignmentEvent.Query().Where(seaorderreassignmenteventent.IdempotencyKeyEQ("reas-space-val-020")).Count(ctx)
		if spaceReasEvtCount != 0 {
			t.Errorf("失败事务不应残留 ReassignmentEvent")
		}
		linkCheck, _ := data.db.SeaMasterBillOrderLink.Get(ctx, fVal.link.ID)
		if linkCheck.Status != seamasterbillorderlinkent.StatusACTIVE {
			t.Errorf("源Link应保持ACTIVE, 实际: %v", linkCheck.Status)
		}

		// 3. 改配目标 master_no 为合法纯小写字母数字 -> 成功执行并保存为大写原号/规范号
		reasLowerInput := &biz.SeaOrderReassignmentInput{
			OrderID:            fVal.order.ID,
			IdempotencyKey:     "reas-lower-val-020",
			RequestFingerprint: "fp-lower-val-020",
			Reason:             "测试小写字母数字规范化",
			ResponsibilityType: biz.ResponsibilityTypeCustomer,
			Target: &biz.SeaOrderReassignmentTargetInput{
				TargetType:      biz.SplitTargetTypeNew,
				MasterNo:        "cosu123456789",
				IssuerPartnerID: &carrier.ID,
				VesselName:      "COSCO STAR",
				VoyageNo:        "CS001",
			},
			ExpectedOrderVersion: fVal.order.Version,
			ExpectedLinkVersion:  fVal.link.Version,
		}
		lowerEvt, err := uc.ExecuteReassignment(ctx, org.ID, user.ID, reasLowerInput)
		if err != nil {
			t.Fatalf("合法小写改配执行失败: %v", err)
		}
		if lowerEvt == nil {
			t.Fatal("未生成改配事件")
		}

		// 验证保存的 MBL 确实保存为大写原号/规范号 "COSU123456789"
		savedMbl, err := data.db.SeaMasterBill.Get(ctx, lowerEvt.TargetMasterBillID)
		if err != nil {
			t.Fatalf("查询保存的新MBL失败: %v", err)
		}
		if savedMbl.MasterNo != "COSU123456789" {
			t.Errorf("MasterNo 未大写规范化: got %s, want %s", savedMbl.MasterNo, "COSU123456789")
		}
		if savedMbl.NormalizedMasterNo != "COSU123456789" {
			t.Errorf("NormalizedMasterNo 未大写规范化: got %s, want %s", savedMbl.NormalizedMasterNo, "COSU123456789")
		}
	})

	// -------------------------------------------------------------------------
	// U. 未命中Target拦截与NEW目标Preview/Execute校验严格闭环 (P1/P2)
	// -------------------------------------------------------------------------
	t.Run("未命中Target与NEW目标Preview和Execute校验严格闭环", func(t *testing.T) {
		fVal := createTestSplitFixture(t, ctx, data, org.ID, customer.ID, carrier.ID, spec.ID, mbl.ID, user.ID, asset.ID, "022")

		// 1. result.ClientTargetKey 未在 targets 定义：Preview 与 Execute 均直接报 SEA_ORDER_SPLIT_INVALID_ARGUMENT
		//    且 data 层彻底杜绝回退 CURRENT
		unmappedTargetInput := &biz.SeaOrderSplitInput{
			OrderID:            fVal.order.ID,
			IdempotencyKey:     "split-unmapped-target-022",
			RequestFingerprint: "fp-unmapped-target-022",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "target-orig", TargetType: biz.SplitTargetTypeCurrent},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "target-orig", HouseBillIDs: []uuid.UUID{fVal.hbl1.ID}},
				{ClientResultKey: "res-new", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "not-exist-target", HouseBillIDs: []uuid.UUID{fVal.hbl2.ID}},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fVal),
		}
		if _, err := uc.PreviewSplit(ctx, org.ID, unmappedTargetInput); err != biz.ErrSeaOrderSplitInvalidArgument {
			t.Fatalf("Preview 未命中目标应返回 ErrSeaOrderSplitInvalidArgument, 实际: %v", err)
		}
		if _, err := uc.ExecuteSplit(ctx, org.ID, user.ID, unmappedTargetInput); err != biz.ErrSeaOrderSplitInvalidArgument {
			t.Fatalf("Execute 未命中目标应返回 ErrSeaOrderSplitInvalidArgument, 实际: %v", err)
		}

		// 2. NEW 目标 Preview 与 Execute 校验闭环
		// 2a. 传入不存在的港口：Preview 与 Execute 均精确拦截为 ErrSeaMasterBillInvalidArgument
		badPortID := uuid.New()
		badPortInput := &biz.SeaOrderSplitInput{
			OrderID:            fVal.order.ID,
			IdempotencyKey:     "split-bad-port-022",
			RequestFingerprint: "fp-bad-port-022",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{
					ClientTargetKey:  "res-new",
					TargetType:       biz.SplitTargetTypeNew,
					MasterNo:         "VALIDNEWPORT022",
					IssuerPartnerID:  &carrier.ID,
					OriginLocationID: &badPortID,
					VesselName:       "SHIP01",
					VoyageNo:         "V01",
				},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fVal.hbl1.ID}},
				{ClientResultKey: "res-new", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new", HouseBillIDs: []uuid.UUID{fVal.hbl2.ID}},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fVal),
		}
		if _, err := uc.PreviewSplit(ctx, org.ID, badPortInput); err != biz.ErrSeaMasterBillInvalidArgument {
			t.Fatalf("Preview 传入不存在港口应返回 ErrSeaMasterBillInvalidArgument, 实际: %v", err)
		}
		if _, err := uc.ExecuteSplit(ctx, org.ID, user.ID, badPortInput); err != biz.ErrSeaMasterBillInvalidArgument {
			t.Fatalf("Execute 传入不存在港口应返回 ErrSeaMasterBillInvalidArgument, 实际: %v", err)
		}

		// 2b. 传入仅具有 CLIENT 角色而无 CARRIER/SUPPLIER 角色的 Partner：Preview 与 Execute 均拦截
		badRoleInput := &biz.SeaOrderSplitInput{
			OrderID:            fVal.order.ID,
			IdempotencyKey:     "split-bad-role-022",
			RequestFingerprint: "fp-bad-role-022",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{
					ClientTargetKey: "res-new",
					TargetType:      biz.SplitTargetTypeNew,
					MasterNo:        "VALIDNEWROLE022",
					IssuerPartnerID: &customer.ID, // customer 仅有 CLIENT 角色
					VesselName:      "SHIP01",
					VoyageNo:        "V01",
				},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fVal.hbl1.ID}},
				{ClientResultKey: "res-new", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new", HouseBillIDs: []uuid.UUID{fVal.hbl2.ID}},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fVal),
		}
		if _, err := uc.PreviewSplit(ctx, org.ID, badRoleInput); err != biz.ErrSeaMasterBillInvalidArgument {
			t.Fatalf("Preview 传入非签发资质Partner应返回 ErrSeaMasterBillInvalidArgument, 实际: %v", err)
		}
		if _, err := uc.ExecuteSplit(ctx, org.ID, user.ID, badRoleInput); err != biz.ErrSeaMasterBillInvalidArgument {
			t.Fatalf("Execute 传入非签发资质Partner应返回 ErrSeaMasterBillInvalidArgument, 实际: %v", err)
		}

		// 2c. 传入已有同名 MBL（唯一冲突）：Preview 与 Execute 均返回 ErrSeaMasterBillExists
		dupMblInput := &biz.SeaOrderSplitInput{
			OrderID:            fVal.order.ID,
			IdempotencyKey:     "split-dup-mbl-022",
			RequestFingerprint: "fp-dup-mbl-022",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{
					ClientTargetKey: "res-new",
					TargetType:      biz.SplitTargetTypeNew,
					MasterNo:        mbl.MasterNo, // 使用既有 MBL 号
					IssuerPartnerID: &carrier.ID,
					VesselName:      "SHIP01",
					VoyageNo:        "V01",
				},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fVal.hbl1.ID}},
				{ClientResultKey: "res-new", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new", HouseBillIDs: []uuid.UUID{fVal.hbl2.ID}},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fVal),
		}
		if _, err := uc.PreviewSplit(ctx, org.ID, dupMblInput); err != biz.ErrSeaMasterBillExists {
			t.Fatalf("Preview 存在同名MBL应返回 ErrSeaMasterBillExists, 实际: %v", err)
		}
		if _, err := uc.ExecuteSplit(ctx, org.ID, user.ID, dupMblInput); err != biz.ErrSeaMasterBillExists {
			t.Fatalf("Execute 存在同名MBL应返回 ErrSeaMasterBillExists, 实际: %v", err)
		}

		// 2d. 传入非法日期格式：Preview 与 Execute 均拦截为 ErrSeaMasterBillInvalidArgument
		badDateInput := &biz.SeaOrderSplitInput{
			OrderID:            fVal.order.ID,
			IdempotencyKey:     "split-bad-date-022",
			RequestFingerprint: "fp-bad-date-022",
			Targets: []*biz.SeaOrderSplitTargetInput{
				{ClientTargetKey: "res-origin", TargetType: biz.SplitTargetTypeCurrent},
				{
					ClientTargetKey: "res-new",
					TargetType:      biz.SplitTargetTypeNew,
					MasterNo:        "VALIDNEWDATE022",
					IssuerPartnerID: &carrier.ID,
					VesselName:      "SHIP01",
					VoyageNo:        "V01",
					ETD:             "not-a-date",
				},
			},
			Results: []*biz.SeaOrderSplitResultInput{
				{ClientResultKey: "res-origin", ResultRole: biz.ResultRoleOriginal, ClientTargetKey: "res-origin", HouseBillIDs: []uuid.UUID{fVal.hbl1.ID}},
				{ClientResultKey: "res-new", ResultRole: biz.ResultRoleCreated, ClientTargetKey: "res-new", HouseBillIDs: []uuid.UUID{fVal.hbl2.ID}},
			},
			ExpectedVersions: buildFixtureExpectedVersions(fVal),
		}
		if _, err := uc.PreviewSplit(ctx, org.ID, badDateInput); err != biz.ErrSeaMasterBillInvalidArgument {
			t.Fatalf("Preview 传入非法日期应返回 ErrSeaMasterBillInvalidArgument, 实际: %v", err)
		}
		if _, err := uc.ExecuteSplit(ctx, org.ID, user.ID, badDateInput); err != biz.ErrSeaMasterBillInvalidArgument {
			t.Fatalf("Execute 传入非法日期应返回 ErrSeaMasterBillInvalidArgument, 实际: %v", err)
		}
	})
}
