package data

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/conf"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	membershipent "github.com/roncin/roncin-go-admin/server/internal/data/ent/membership"
	numberruleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numberrule"
	numbersequenceent "github.com/roncin/roncin-go-admin/server/internal/data/ent/numbersequence"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
	ordercommissionattributionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordercommissionattribution"
	orderlifecycleeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderlifecycleevent"
	orderpersonnelent "github.com/roncin/roncin-go-admin/server/internal/data/ent/orderpersonnel"
	ordershippingdocumentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/ordershippingdocument"
	organizationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/organization"
	partnerent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partner"
	partnerroleent "github.com/roncin/roncin-go-admin/server/internal/data/ent/partnerrole"
	portent "github.com/roncin/roncin-go-admin/server/internal/data/ent/port"
	seahousebill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seamasterbill "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbill"
	seamasterbillorderlink "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
	seatransportexecution "github.com/roncin/roncin-go-admin/server/internal/data/ent/seatransportexecution"
	userent "github.com/roncin/roncin-go-admin/server/internal/data/ent/user"
)

type orderPostgresFixture struct {
	t              *testing.T
	data           *Data
	organizationID uuid.UUID
	partnerID      uuid.UUID
	actorID        uuid.UUID
	suffix         string
}

type orderWriteResult struct {
	order *biz.Order
	err   error
}

func TestOrderCreateTransactionPostgres(t *testing.T) {
	source := os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE")
	if source == "" {
		t.Skip("未配置临时 PostgreSQL 集成测试数据库")
	}
	data, cleanup, err := NewData(&conf.Data{Database: &conf.Data_Database{
		Driver:             "postgres",
		Source:             source,
		AutoMigrate:        true,
		MaxOpenConnections: 8,
		MaxIdleConnections: 8,
	}}, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatalf("初始化集成测试数据库: %v", err)
	}
	defer cleanup()

	t.Run("创建订单审计包含初始单证结构与HBL数量", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		input := fixture.validInput()
		structure := biz.SeaDocumentStructureHouse
		input.SeaDocumentInput = &biz.SeaOrderDocumentInput{
			DocumentStructure: &structure,
			HouseBills: []*biz.SeaHouseBillInput{
				{HouseNo: "  HBL-AUDIT-001  ", IssuerSource: biz.SeaHouseBillIssuerSourceSelfOrganization},
			},
		}
		created, err := fixture.newUsecase().Create(context.Background(), fixture.organizationID, fixture.actorID, input)
		if err != nil {
			t.Fatalf("创建带初始 HBL 的订单失败: %v", err)
		}
		if created.SeaDocumentSummary == nil || created.SeaDocumentSummary.DocumentStructure != biz.SeaDocumentStructureHouse || created.SeaDocumentSummary.HouseBillCount != 1 {
			t.Fatalf("创建后的单证摘要异常: %#v", created.SeaDocumentSummary)
		}
		audit, err := data.db.AuditLog.Query().Where(
			auditlogent.OrganizationIDEQ(fixture.organizationID),
			auditlogent.ActionEQ("order.create"),
		).Only(context.Background())
		if err != nil {
			t.Fatalf("读取订单创建审计失败: %v", err)
		}
		var details map[string]string
		if err := json.Unmarshal(audit.Details, &details); err != nil {
			t.Fatalf("解析订单创建审计详情失败: %v", err)
		}
		if details["sea_document.initial_structure"] != "HOUSE" || details["sea_house_bills.initial_count"] != "1" {
			t.Fatalf("订单创建审计缺少初始单证摘要: %#v", details)
		}
	})

	t.Run("并发创建订单时发号不重号不丢号", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		operations := make([]func(context.Context) (*biz.Order, error), 0, 4)
		for i := range 4 {
			input := fixture.validInput()
			input.SeaMasterBillInput.MasterNo = strings.ToUpper(fixture.suffix) + string(rune('A'+i))
			operations = append(operations, func(ctx context.Context) (*biz.Order, error) {
				return usecase.Create(ctx, fixture.organizationID, fixture.actorID, input)
			})
		}
		results := runOrderWritesConcurrently(operations...)
		numbers := make(map[string]struct{}, len(results))
		for index, result := range results {
			if result.err != nil {
				t.Fatalf("第 %d 个并发请求创建订单: %v", index+1, result.err)
			}
			if result.order == nil || result.order.OrderNo == "" {
				t.Fatalf("第 %d 个并发请求未返回订单号: %#v", index+1, result.order)
			}
			if result.order.Version != 1 || result.order.FlowStatus != biz.OrderFlowDraft {
				t.Fatalf("第 %d 个并发订单初始状态异常: version=%d flowStatus=%s", index+1, result.order.Version, result.order.FlowStatus)
			}
			numbers[result.order.OrderNo] = struct{}{}
		}
		if len(numbers) != len(results) {
			t.Fatalf("并发创建出现重复订单号: %v", numbers)
		}
		fixture.requireCommittedOrders(4, 4)
	})

	t.Run("引用校验失败回滚订单与单号序列", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		input := fixture.validInput()
		input.CustomerID = uuid.New()

		_, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, input)
		if !errors.Is(err, biz.ErrOrderCustomerInvalid) {
			t.Fatalf("无效客户创建订单的错误 = %v，期望 ErrOrderCustomerInvalid", err)
		}
		fixture.requireRolledBackState()
	})

	t.Run("并发修改草稿时版本冲突只有一个成功", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		created, err := usecase.Create(context.Background(), fixture.organizationID, fixture.actorID, fixture.validInput())
		if err != nil {
			t.Fatalf("创建草稿订单: %v", err)
		}
		first, second := fixture.validInput(), fixture.validInput()
		first.GoodsDescription = "并发修改一"
		second.GoodsDescription = "并发修改二"
		results := runOrderWritesConcurrently(
			func(ctx context.Context) (*biz.Order, error) {
				return usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version, first)
			},
			func(ctx context.Context) (*biz.Order, error) {
				return usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, created.ID, created.Version, second)
			},
		)
		successes, conflicts := 0, 0
		for _, result := range results {
			switch {
			case result.err == nil && result.order != nil:
				successes++
				if result.order.Version != 2 {
					t.Fatalf("并发更新成功方版本 = %d，期望 2", result.order.Version)
				}
			case errors.Is(result.err, biz.ErrOrderStatusConflict):
				conflicts++
			default:
				t.Fatalf("并发更新返回非预期结果: order=%#v error=%v", result.order, result.err)
			}
		}
		if successes != 1 || conflicts != 1 {
			t.Fatalf("并发更新结果 success=%d conflict=%d，期望各 1", successes, conflicts)
		}
		fixture.requireDraftUpdateResult()
	})

	t.Run("海运主单匹配、确认关联与修改门禁全生命周期", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		ctx := context.Background()

		// 1. 创建第 1 票订单，创建全新 MBL
		order1Input := fixture.validInput()
		order1Input.SeaMasterBillInput.MasterNo = "COSCO999901"
		order1, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, order1Input)
		if err != nil {
			t.Fatalf("创建订单1失败: %v", err)
		}
		if order1.SeaMasterBill == nil || order1.SeaMasterBill.MasterNo != "COSCO999901" || order1.SeaMasterBill.MemberCount != 1 {
			t.Fatalf("订单1主单概要异常: %#v", order1.SeaMasterBill)
		}

		// 2. 尝试创建第 2 票订单，使用相同 MBL No + 签发方，未指定 candidateID -> 必须拦截并要求确认
		order2DuplicateInput := fixture.validInput()
		order2DuplicateInput.SeaMasterBillInput.MasterNo = "cosco999901" // 小写测试
		_, err = usecase.Create(ctx, fixture.organizationID, fixture.actorID, order2DuplicateInput)
		if !errors.Is(err, biz.ErrSeaMasterBillConfirmationRequired) {
			t.Fatalf("未确认关联创建同主单应返回 ErrSeaMasterBillConfirmationRequired, 实际: %v", err)
		}

		// 3. 调用候选匹配查询
		matchResult, err := usecase.MatchSeaMasterBillCandidate(ctx, fixture.organizationID, fixture.partnerID, "cosco999901", nil)
		if err != nil {
			t.Fatalf("匹配主单候选失败: %v", err)
		}
		if !matchResult.Matched || matchResult.Candidate == nil || matchResult.Candidate.MemberCount != 1 {
			t.Fatalf("候选匹配结果异常: %#v", matchResult)
		}

		// 4. 显式确认关联候选创建第 2 票订单
		order2Input := fixture.validInput()
		order2Input.SeaMasterBillInput.MasterNo = "COSCO999901"
		order2Input.SeaMasterBillInput.CandidateID = &matchResult.Candidate.ID
		order2Input.SeaMasterBillInput.ExpectedCandidateVersion = &matchResult.Candidate.Version
		order2, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, order2Input)
		if err != nil {
			t.Fatalf("确认关联创建订单2失败: %v", err)
		}
		if order2.SeaMasterBill == nil || order2.SeaMasterBill.MasterBillID != matchResult.Candidate.ID || order2.SeaMasterBill.MemberCount != 2 {
			t.Fatalf("订单2主单关联异常: %#v", order2.SeaMasterBill)
		}

		// 5. 此时主单有 2 个成员，尝试单票修改主单号 -> 必须被拦截 (ErrSeaMasterBillCorrectionBlocked)
		order1UpdateAttempt := fixture.validInput()
		order1UpdateAttempt.SeaMasterBillInput.MasterNo = "COSCO999902"
		order1UpdateAttempt.SeaMasterBillInput.CorrectionReason = "尝试修改共享主单号"
		_, err = usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order1.ID, order1.Version, order1UpdateAttempt)
		if !errors.Is(err, biz.ErrSeaMasterBillCorrectionBlocked) {
			t.Fatalf("多成员主单修改主单号应被拒绝, 实际: %v", err)
		}

		// 6. 创建单票 MBL 的独立订单 3
		order3Input := fixture.validInput()
		order3Input.SeaMasterBillInput.MasterNo = "MSCU888801"
		order3, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, order3Input)
		if err != nil {
			t.Fatalf("创建订单3失败: %v", err)
		}

		// 7. 单票 MBL 修改主单号，未填写更正原因 -> 失败
		order3NoReason := fixture.validInput()
		order3NoReason.SeaMasterBillInput.MasterNo = "MSCU888802"
		order3NoReason.SeaMasterBillInput.ExpectedCandidateVersion = &order3.SeaMasterBill.Version
		order3NoReason.SeaMasterBillInput.CorrectionReason = ""
		_, err = usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order3.ID, order3.Version, order3NoReason)
		if err == nil {
			t.Fatalf("单票修改主单号无原因应失败，实际成功")
		}

		// 8. 单票 MBL 修改主单号，填写更正原因 -> 成功
		order3WithReason := fixture.validInput()
		order3WithReason.SeaMasterBillInput.MasterNo = "MSCU888802"
		order3WithReason.SeaMasterBillInput.ExpectedCandidateVersion = &order3.SeaMasterBill.Version
		order3WithReason.SeaMasterBillInput.CorrectionReason = "输入录入错误更正"
		updatedOrder3, err := usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order3.ID, order3.Version, order3WithReason)
		if err != nil {
			t.Fatalf("单票修改主单号带原因更新失败: %v", err)
		}
		if updatedOrder3.SeaMasterBill == nil || updatedOrder3.SeaMasterBill.MasterNo != "MSCU888802" {
			t.Fatalf("更新后订单3主单号未变更: %#v", updatedOrder3.SeaMasterBill)
		}
	})

	t.Run("同组织同签发方同主单号并发创建只能成功一票", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		first := fixture.validInput()
		second := fixture.validInput()
		first.SeaMasterBillInput.MasterNo = "ONE" + strings.ToUpper(fixture.suffix)
		second.SeaMasterBillInput.MasterNo = first.SeaMasterBillInput.MasterNo

		results := runOrderWritesConcurrently(
			func(ctx context.Context) (*biz.Order, error) {
				return usecase.Create(ctx, fixture.organizationID, fixture.actorID, first)
			},
			func(ctx context.Context) (*biz.Order, error) {
				return usecase.Create(ctx, fixture.organizationID, fixture.actorID, second)
			},
		)
		successes, confirmations := 0, 0
		for _, result := range results {
			switch {
			case result.err == nil:
				successes++
			case errors.Is(result.err, biz.ErrSeaMasterBillConfirmationRequired):
				confirmations++
			default:
				t.Fatalf("并发同主单创建返回非预期结果: order=%#v error=%v", result.order, result.err)
			}
		}
		if successes != 1 || confirmations != 1 {
			t.Fatalf("并发同主单创建结果 success=%d confirmation=%d，期望各 1", successes, confirmations)
		}
		ctx := context.Background()
		for label, count := range map[string]int{
			"订单":   fixture.mustCountOrders(ctx),
			"运输执行": fixture.mustCountTransportExecutions(ctx),
			"共享主单": fixture.mustCountMasterBills(ctx),
			"活动关系": fixture.mustCountActiveMasterBillLinks(ctx),
		} {
			if count != 1 {
				t.Fatalf("并发同主单创建后%s数 = %d，期望 1", label, count)
			}
		}
	})

	t.Run("不同签发方允许使用相同主单号且一票只能有一个活动关系", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		ctx := context.Background()
		issuer := fixture.createSupplier(ctx, "ISSUER-"+fixture.suffix)
		usecase := fixture.newUsecase()
		masterNo := "SAME" + strings.ToUpper(fixture.suffix)

		first := fixture.validInput()
		first.SeaMasterBillInput.MasterNo = masterNo
		created, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, first)
		if err != nil {
			t.Fatalf("创建第一签发方主单: %v", err)
		}
		second := fixture.validInput()
		second.SeaMasterBillInput.MasterNo = masterNo
		second.SeaMasterBillInput.IssuerPartnerID = issuer.ID
		if _, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, second); err != nil {
			t.Fatalf("不同签发方使用相同主单号: %v", err)
		}
		if count := fixture.mustCountMasterBills(ctx); count != 2 {
			t.Fatalf("不同签发方同号主单数 = %d，期望 2", count)
		}

		masterBill, err := data.db.SeaMasterBill.Query().Where(
			seamasterbill.OrganizationIDEQ(fixture.organizationID),
			seamasterbill.IssuerPartnerIDEQ(issuer.ID),
		).Only(ctx)
		if err != nil {
			t.Fatalf("读取第二签发方主单: %v", err)
		}
		_, err = data.db.SeaMasterBillOrderLink.Create().
			SetOrganizationID(fixture.organizationID).
			SetMasterBillID(masterBill.ID).
			SetOrderID(created.ID).
			SetStatus(seamasterbillorderlink.StatusACTIVE).
			SetStartedAt(time.Now().UTC()).
			Save(ctx)
		if !ent.IsConstraintError(err) {
			t.Fatalf("同一订单创建第二个活动主单关系应命中数据库唯一约束，实际: %v", err)
		}
	})

	t.Run("多成员共享MBL修改航程冲突阻断且目的地不参与共享校验", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		ctx := context.Background()
		usecase := fixture.newUsecase()

		destPort1 := fixture.createPort(ctx, "D1"+fixture.suffix[:3], "目的港1")
		destPort2 := fixture.createPort(ctx, "D2"+fixture.suffix[:3], "目的港2")

		masterNo := "VOY" + strings.ToUpper(fixture.suffix)
		first := fixture.validInput()
		first.SeaMasterBillInput.MasterNo = masterNo
		first.VesselVoyage = "EVER GIVEN / 001W"
		first.ETD = "2026-09-10T00:00:00Z"
		first.ETA = "2026-09-25T00:00:00Z"
		first.DestinationLocationID = &destPort1.ID

		order1, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, first)
		if err != nil {
			t.Fatalf("创建订单1: %v", err)
		}

		// 创建共享订单2
		second := fixture.validInput()
		second.SeaMasterBillInput.MasterNo = masterNo
		second.SeaMasterBillInput.CandidateID = &order1.SeaMasterBill.MasterBillID
		second.SeaMasterBillInput.ExpectedCandidateVersion = &order1.SeaMasterBill.Version
		second.VesselVoyage = "EVER GIVEN / 001W"
		second.ETD = "2026-09-10T00:00:00Z"
		second.ETA = "2026-09-25T00:00:00Z"

		_, err = usecase.Create(ctx, fixture.organizationID, fixture.actorID, second)
		if err != nil {
			t.Fatalf("创建共享订单2: %v", err)
		}

		// 尝试修改订单1的船名航次与共享运输执行冲突 -> 必须阻断
		conflictUpdate := fixture.validInput()
		conflictUpdate.SeaMasterBillInput.MasterNo = masterNo
		conflictUpdate.VesselVoyage = "EVER GIVEN / 002E"
		conflictUpdate.ETD = "2026-09-10T00:00:00Z"
		conflictUpdate.ETA = "2026-09-25T00:00:00Z"

		_, err = usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order1.ID, order1.Version, conflictUpdate)
		if !errors.Is(err, biz.ErrSeaMasterBillVoyageConflict) {
			t.Fatalf("修改多成员共享航程冲突应返回 ErrSeaMasterBillVoyageConflict, 实际: %v", err)
		}

		// 修改目的地不参与共享航程冲突校验 -> 必须成功
		destUpdate := fixture.validInput()
		destUpdate.SeaMasterBillInput.MasterNo = masterNo
		destUpdate.VesselVoyage = "EVER GIVEN / 001W"
		destUpdate.ETD = "2026-09-10T00:00:00Z"
		destUpdate.ETA = "2026-09-25T00:00:00Z"
		destUpdate.DestinationLocationID = &destPort2.ID

		updatedOrder1, err := usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order1.ID, order1.Version, destUpdate)
		if err != nil {
			t.Fatalf("修改目的地应成功: %v", err)
		}
		if updatedOrder1.DestinationLocationID == nil || *updatedOrder1.DestinationLocationID != destPort2.ID {
			t.Fatalf("目的地未正确更新: %v", updatedOrder1.DestinationLocationID)
		}
	})

	t.Run("单成员DRAFT状态MBL修改航程更新共享运输执行", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		ctx := context.Background()
		usecase := fixture.newUsecase()

		masterNo := "SINGLE" + strings.ToUpper(fixture.suffix)
		input := fixture.validInput()
		input.SeaMasterBillInput.MasterNo = masterNo
		input.VesselVoyage = "VESSEL A / 001"
		input.ETD = "2026-09-10T00:00:00Z"

		order, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, input)
		if err != nil {
			t.Fatalf("创建单票订单: %v", err)
		}

		// 单成员修改航程
		updateInput := fixture.validInput()
		updateInput.SeaMasterBillInput.MasterNo = masterNo
		updateInput.VesselVoyage = "VESSEL B / 002"
		updateInput.ETD = "2026-09-12T00:00:00Z"

		_, err = usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, updateInput)
		if !errors.Is(err, biz.ErrSeaMasterBillStatusConflict) {
			t.Fatalf("单成员修改航程缺少主单预期版本应冲突, 实际: %v", err)
		}
		storedBeforeRetry, err := fixture.data.db.Order.Query().Where(orderent.IDEQ(order.ID)).Only(ctx)
		if err != nil || storedBeforeRetry.Version != order.Version || storedBeforeRetry.VesselVoyage != "VESSEL A / 001" {
			t.Fatalf("版本冲突后订单写入未完整回滚: order=%#v error=%v", storedBeforeRetry, err)
		}

		updateInput.SeaMasterBillInput.ExpectedCandidateVersion = &order.SeaMasterBill.Version
		updated, err := usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, updateInput)
		if err != nil {
			t.Fatalf("单成员更新草稿航程: %v", err)
		}
		if updated.VesselVoyage != "VESSEL B / 002" {
			t.Fatalf("订单船名航次未更新: %q", updated.VesselVoyage)
		}

		// 检查运输执行已同步更新
		te, err := fixture.data.db.SeaTransportExecution.Query().Where(
			seatransportexecution.OrganizationIDEQ(fixture.organizationID),
		).Only(ctx)
		if err != nil {
			t.Fatalf("读取运输执行: %v", err)
		}
		if te.VesselName != "VESSEL B" || te.VoyageNo != "002" {
			t.Fatalf("运输执行船名航次未更新: vessel=%q voyage=%q", te.VesselName, te.VoyageNo)
		}
		if updated.SeaMasterBill == nil || updated.SeaMasterBill.Version != order.SeaMasterBill.Version+1 {
			t.Fatalf("航程更新后主单候选版本未递增: before=%#v after=%#v", order.SeaMasterBill, updated.SeaMasterBill)
		}
	})

	t.Run("单成员草稿主单存在已确认分单时禁止更正身份或航程", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		ctx := context.Background()
		usecase := fixture.newUsecase()
		input := fixture.validInput()
		input.SeaMasterBillInput.MasterNo = "DOWN" + strings.ToUpper(fixture.suffix)
		input.VesselVoyage = "VESSEL A / 001"
		order, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, input)
		if err != nil {
			t.Fatalf("创建下游门禁测试订单: %v", err)
		}
		if _, err := fixture.data.db.OrderShippingDocument.Create().
			SetOrderID(order.ID).
			SetHouseNo("HBL-" + fixture.suffix).
			SetStatus(ordershippingdocumentent.StatusCONFIRMED).
			Save(ctx); err != nil {
			t.Fatalf("创建已确认分单事实: %v", err)
		}

		identityUpdate := fixture.validInput()
		identityUpdate.SeaMasterBillInput.MasterNo = "NEWDOWN" + strings.ToUpper(fixture.suffix)
		identityUpdate.SeaMasterBillInput.ExpectedCandidateVersion = &order.SeaMasterBill.Version
		identityUpdate.SeaMasterBillInput.CorrectionReason = "录入错误"
		identityUpdate.VesselVoyage = input.VesselVoyage
		_, err = usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, identityUpdate)
		if !errors.Is(err, biz.ErrSeaMasterBillCorrectionBlocked) {
			t.Fatalf("存在已确认分单时更正主单身份应被阻断, 实际: %v", err)
		}

		voyageUpdate := fixture.validInput()
		voyageUpdate.SeaMasterBillInput.MasterNo = input.SeaMasterBillInput.MasterNo
		voyageUpdate.SeaMasterBillInput.ExpectedCandidateVersion = &order.SeaMasterBill.Version
		voyageUpdate.VesselVoyage = "VESSEL B / 002"
		_, err = usecase.UpdateDraft(ctx, fixture.organizationID, fixture.actorID, order.ID, order.Version, voyageUpdate)
		if !errors.Is(err, biz.ErrSeaMasterBillCorrectionBlocked) {
			t.Fatalf("存在已确认分单时修改共享航程应被阻断, 实际: %v", err)
		}
		stored, err := fixture.data.db.Order.Query().Where(orderent.IDEQ(order.ID)).Only(ctx)
		if err != nil || stored.Version != order.Version || stored.VesselVoyage != input.VesselVoyage {
			t.Fatalf("下游门禁失败后订单写入未完整回滚: order=%#v error=%v", stored, err)
		}
	})
}

func (f *orderPostgresFixture) createPort(ctx context.Context, unlocode, name string) *ent.Port {
	port, err := f.data.db.Port.Create().
		SetOrganizationID(f.organizationID).
		SetUnLocode(unlocode).
		SetNameZh(name).
		SetNameEn(name).
		SetCountryCode("CN").
		SetTransportModes([]string{"SEA"}).
		SetSource("system").
		SetSortOrder(1).
		SetEnabled(true).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试港口: %v", err)
	}
	return port
}

func (f *orderPostgresFixture) mustCountOrders(ctx context.Context) int {
	count, err := f.data.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil {
		f.t.Fatalf("统计订单: %v", err)
	}
	return count
}

func (f *orderPostgresFixture) mustCountTransportExecutions(ctx context.Context) int {
	count, err := f.data.db.SeaTransportExecution.Query().Where(seatransportexecution.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil {
		f.t.Fatalf("统计运输执行: %v", err)
	}
	return count
}

func (f *orderPostgresFixture) mustCountMasterBills(ctx context.Context) int {
	count, err := f.data.db.SeaMasterBill.Query().Where(seamasterbill.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil {
		f.t.Fatalf("统计共享主单: %v", err)
	}
	return count
}

func (f *orderPostgresFixture) mustCountActiveMasterBillLinks(ctx context.Context) int {
	count, err := f.data.db.SeaMasterBillOrderLink.Query().Where(
		seamasterbillorderlink.OrganizationIDEQ(f.organizationID),
		seamasterbillorderlink.StatusEQ(seamasterbillorderlink.StatusACTIVE),
	).Count(ctx)
	if err != nil {
		f.t.Fatalf("统计活动主单关系: %v", err)
	}
	return count
}

func (f *orderPostgresFixture) createSupplier(ctx context.Context, code string) *ent.Partner {
	partner, err := f.data.db.Partner.Create().
		SetOrganizationID(f.organizationID).
		SetCode(code).
		SetLegalName("订单事务测试签发方-" + f.suffix).
		SetNormalizedName("订单事务测试签发方-" + f.suffix).
		Save(ctx)
	if err != nil {
		f.t.Fatalf("创建测试签发方: %v", err)
	}
	if _, err := f.data.db.PartnerRole.Create().
		SetPartnerID(partner.ID).
		SetRoleType(partnerroleent.RoleTypeSupplier).
		SetEnabled(true).
		Save(ctx); err != nil {
		f.t.Fatalf("创建测试签发方角色: %v", err)
	}
	return partner
}

func newOrderPostgresFixture(t *testing.T, data *Data) *orderPostgresFixture {
	t.Helper()
	ctx := context.Background()
	suffix := strings.ReplaceAll(uuid.NewString(), "-", "")[:12]
	organization, err := data.db.Organization.Create().
		SetCode("ORDER-TX-" + suffix).
		SetName("订单事务集成测试组织-" + suffix).
		SetKind("company").
		SetBaseCurrency("CNY").
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试组织: %v", err)
	}
	fixture := &orderPostgresFixture{
		t: t, data: data, organizationID: organization.ID, suffix: suffix,
	}
	t.Cleanup(fixture.cleanup)

	partner, err := data.db.Partner.Create().
		SetOrganizationID(organization.ID).
		SetCode("CUSTOMER-" + suffix).
		SetLegalName("订单事务测试客户-" + suffix).
		SetNormalizedName("订单事务测试客户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试往来单位: %v", err)
	}
	fixture.partnerID = partner.ID
	if _, err = data.db.PartnerRole.Create().
		SetPartnerID(partner.ID).
		SetRoleType(partnerroleent.RoleTypeCustomer).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试客户角色: %v", err)
	}
	if _, err = data.db.PartnerRole.Create().
		SetPartnerID(partner.ID).
		SetRoleType(partnerroleent.RoleTypeSupplier).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试供应商角色: %v", err)
	}

	actor, err := data.db.User.Create().
		SetDisplayName("订单事务测试用户-" + suffix).
		Save(ctx)
	if err != nil {
		t.Fatalf("创建测试用户: %v", err)
	}
	fixture.actorID = actor.ID
	if _, err = data.db.Membership.Create().
		SetUserID(actor.ID).
		SetOrganizationID(organization.ID).
		SetPrimary(true).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试成员关系: %v", err)
	}

	if _, err = data.db.NumberRule.Create().
		SetOrganizationID(organization.ID).
		SetDocumentType(numberruleent.DocumentTypeOrder).
		SetPrefix("SE-").
		SetDateFormat(numberruleent.DateFormatYyyyMMdd).
		SetSequenceLength(5).
		SetResetPolicy(numberruleent.ResetPolicyDaily).
		SetEnabled(true).
		Save(ctx); err != nil {
		t.Fatalf("创建测试订单编号规则: %v", err)
	}
	return fixture
}

func (f *orderPostgresFixture) newUsecase() *biz.OrderUsecase {
	return biz.NewOrderUsecase(NewOrderRepo(f.data), NewBusinessTagRepo(f.data), NewSeaMasterBillRepo(f.data), NewSeaDocumentRepo(f.data))
}

func (f *orderPostgresFixture) validInput() *biz.Order {
	return &biz.Order{
		CustomerID:     f.partnerID,
		BusinessType:   biz.OrderBusinessSE,
		TradeDirection: biz.OrderTradeExport,
		TradeTerm:      biz.OrderTradeFOB,
		PaymentTerm:    biz.OrderPaymentPrepaid,
		SeaMasterBillInput: &biz.SeaMasterBillInput{
			MasterNo:        "COSCO0001",
			IssuerPartnerID: f.partnerID,
		},
	}
}

func runOrderWritesConcurrently(operations ...func(context.Context) (*biz.Order, error)) []orderWriteResult {
	start := make(chan struct{})
	results := make(chan orderWriteResult, len(operations))
	for _, operation := range operations {
		operation := operation
		go func() {
			<-start
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			order, err := operation(ctx)
			results <- orderWriteResult{order: order, err: err}
		}()
	}
	close(start)
	collected := make([]orderWriteResult, 0, len(operations))
	for range operations {
		collected = append(collected, <-results)
	}
	return collected
}

func (f *orderPostgresFixture) requireCommittedOrders(wantOrders int, wantSequence int64) {
	f.t.Helper()
	ctx := context.Background()
	orders, err := f.data.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).All(ctx)
	if err != nil || len(orders) != wantOrders {
		f.t.Fatalf("已提交订单数 = %d，期望 %d，error=%v", len(orders), wantOrders, err)
	}
	lifecycleCount, err := f.data.db.OrderLifecycleEvent.Query().Where(orderlifecycleeventent.HasOrderWith(orderent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || lifecycleCount != wantOrders {
		f.t.Fatalf("已提交生命周期事件数 = %d，期望 %d，error=%v", lifecycleCount, wantOrders, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("order.create")).Count(ctx)
	if err != nil || auditCount != wantOrders {
		f.t.Fatalf("已提交订单审计数 = %d，期望 %d，error=%v", auditCount, wantOrders, err)
	}
	personnelCount, err := f.data.db.OrderPersonnel.Query().Where(orderpersonnelent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || personnelCount != wantOrders {
		f.t.Fatalf("已提交订单人员数 = %d，期望 %d（仅创建人），error=%v", personnelCount, wantOrders, err)
	}
	sequences, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).All(ctx)
	if err != nil || len(sequences) != 1 || sequences[0].CurrentValue != wantSequence {
		f.t.Fatalf("已提交订单序列 = %#v，期望当前值 %d，error=%v", sequences, wantSequence, err)
	}
}

func (f *orderPostgresFixture) requireRolledBackState() {
	f.t.Helper()
	ctx := context.Background()
	orderCount, err := f.data.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || orderCount != 0 {
		f.t.Fatalf("回滚后订单数 = %d，期望 0，error=%v", orderCount, err)
	}
	lifecycleCount, err := f.data.db.OrderLifecycleEvent.Query().Where(orderlifecycleeventent.HasOrderWith(orderent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || lifecycleCount != 0 {
		f.t.Fatalf("回滚后生命周期事件数 = %d，期望 0，error=%v", lifecycleCount, err)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Count(ctx)
	if err != nil || auditCount != 0 {
		f.t.Fatalf("回滚后审计数 = %d，期望 0，error=%v", auditCount, err)
	}
	sequenceCount, err := f.data.db.NumberSequence.Query().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).Count(ctx)
	if err != nil || sequenceCount != 0 {
		f.t.Fatalf("回滚后订单序列数 = %d，期望 0，error=%v", sequenceCount, err)
	}
}

func (f *orderPostgresFixture) requireDraftUpdateResult() {
	f.t.Helper()
	ctx := context.Background()
	stored, err := f.data.db.Order.Query().Where(orderent.OrganizationIDEQ(f.organizationID)).Only(ctx)
	if err != nil {
		f.t.Fatalf("查询并发更新后的订单: %v", err)
	}
	if stored.Version != 2 {
		f.t.Fatalf("并发更新后订单版本 = %d，期望 2", stored.Version)
	}
	if stored.GoodsDescription != "并发修改一" && stored.GoodsDescription != "并发修改二" {
		f.t.Fatalf("并发更新后货物描述 = %q，期望为其中一个并发值", stored.GoodsDescription)
	}
	auditCount, err := f.data.db.AuditLog.Query().Where(auditlogent.OrganizationIDEQ(f.organizationID), auditlogent.ActionEQ("order.update")).Count(ctx)
	if err != nil || auditCount != 1 {
		f.t.Fatalf("并发更新后 order.update 审计数 = %d，期望 1，error=%v", auditCount, err)
	}
}

func (f *orderPostgresFixture) cleanup() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	steps := []struct {
		name string
		run  func() error
	}{
		{name: "订单人员", run: func() error {
			_, err := f.data.db.OrderPersonnel.Delete().Where(orderpersonnelent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "订单生命周期事件", run: func() error {
			_, err := f.data.db.OrderLifecycleEvent.Delete().Where(orderlifecycleeventent.HasOrderWith(orderent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "订单分单", run: func() error {
			_, err := f.data.db.OrderShippingDocument.Delete().Where(ordershippingdocumentent.HasOrderWith(orderent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "订单提成归属快照", run: func() error {
			_, err := f.data.db.OrderCommissionAttribution.Delete().Where(ordercommissionattributionent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "海运分单", run: func() error {
			_, err := f.data.db.SeaHouseBill.Delete().Where(seahousebill.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "海运主单关联", run: func() error {
			_, err := f.data.db.SeaMasterBillOrderLink.Delete().Where(seamasterbillorderlink.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "订单", run: func() error {
			_, err := f.data.db.Order.Delete().Where(orderent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "海运主单", run: func() error {
			_, err := f.data.db.SeaMasterBill.Delete().Where(seamasterbill.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "海运运输执行", run: func() error {
			_, err := f.data.db.SeaTransportExecution.Delete().Where(seatransportexecution.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "订单审计", run: func() error {
			_, err := f.data.db.AuditLog.Delete().Where(auditlogent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "订单序列", run: func() error {
			_, err := f.data.db.NumberSequence.Delete().Where(numbersequenceent.HasRuleWith(numberruleent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "订单编号规则", run: func() error {
			_, err := f.data.db.NumberRule.Delete().Where(numberruleent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "成员关系", run: func() error {
			_, err := f.data.db.Membership.Delete().Where(membershipent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "用户", run: func() error {
			_, err := f.data.db.User.Delete().Where(userent.IDEQ(f.actorID)).Exec(ctx)
			return err
		}},
		{name: "往来单位角色", run: func() error {
			_, err := f.data.db.PartnerRole.Delete().Where(partnerroleent.HasPartnerWith(partnerent.OrganizationIDEQ(f.organizationID))).Exec(ctx)
			return err
		}},
		{name: "往来单位", run: func() error {
			_, err := f.data.db.Partner.Delete().Where(partnerent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "港口", run: func() error {
			_, err := f.data.db.Port.Delete().Where(portent.OrganizationIDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
		{name: "组织", run: func() error {
			_, err := f.data.db.Organization.Delete().Where(organizationent.IDEQ(f.organizationID)).Exec(ctx)
			return err
		}},
	}
	for _, step := range steps {
		if err := step.run(); err != nil {
			f.t.Errorf("清理%s失败: %v", step.name, err)
			return
		}
	}
}
