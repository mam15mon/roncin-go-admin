package data

// 海运共享主单批次规则集成测试（共享主单自动关联、直单禁拼与批次内分单排重）：
// T1 命中全 HOUSE 批次创建成功入批；候选版本并发变化仍 409。
// T2 命中含直单成员批次 → ErrSeaMasterBillBatchDirectBlocked。
// T3 批次成员 HOUSE→DIRECT：还有其他活动成员 → 阻断；仅剩自己 → 放行。
// T4 批次排重：跨签发主体同号（含与作废行同号）→ 含冲突号的友好错误；
//    跨批次同号成功；唯一索引并发兜底。
// T5 迁移：存量同批次重号 → DO 块预检中文报错终止；干净库从零迁移成功。
// T6 既有共享主单/单证用例回归由 order_transaction_integration_test.go 与
//    sea_document_change_integration_test.go 覆盖。

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	kratoserrors "github.com/go-kratos/kratos/v3/errors"
	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	seadocumentmodechangeeventent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seadocumentmodechangeevent"
	seahousebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebill"
	seahousebillversionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seahousebillversion"
	seamasterbillorderlinkent "github.com/roncin/roncin-go-admin/server/internal/data/ent/seamasterbillorderlink"
)

const seaHouseBillBatchNoUniqueIndex = `idx_sea_house_bills_batch_no_unique`
const seaHouseBillBatchNoUniqueMigration = `20260916100000_sea_house_bill_batch_no_unique.sql`

func TestSeaMasterBillBatchRulesPostgres(t *testing.T) {
	if os.Getenv("RONCIN_INTEGRATION_DATABASE_SOURCE") == "" {
		t.Skip("未配置专用 RONCIN_INTEGRATION_DATABASE_SOURCE")
	}
	ctx := context.Background()
	data, cleanup := getIntegrationData(t)
	defer cleanup()

	t.Run("T1_命中全HOUSE批次自动入批与候选并发冲突", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		masterNo := "COSCOBAT" + strings.ToUpper(fixture.suffix[:8])

		input1 := fixture.validInput()
		input1.SeaMasterBillInput.MasterNo = masterNo
		if _, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, input1); err != nil {
			t.Fatalf("创建批次首票订单失败: %v", err)
		}

		match, err := usecase.MatchSeaMasterBillCandidate(ctx, fixture.organizationID, fixture.shippingLineID, strings.ToLower(masterNo), nil)
		if err != nil {
			t.Fatalf("匹配共享主单候选失败: %v", err)
		}
		if !match.Matched || match.Candidate == nil || match.Candidate.MemberCount != 1 {
			t.Fatalf("候选匹配结果异常: %#v", match)
		}
		if len(match.Candidate.Members) != 1 || match.Candidate.Members[0].DocumentStructure != biz.SeaDocumentStructureHouse {
			t.Fatalf("候选成员单证结构应为 HOUSE: %#v", match.Candidate.Members)
		}
		wantHouseNo := input1.SeaDocumentInput.HouseBill.HouseNo
		if len(match.Candidate.BatchNormalizedHouseNos) != 1 || match.Candidate.BatchNormalizedHouseNos[0] != wantHouseNo {
			t.Fatalf("候选批次分单号清单异常: %#v，期望包含 %s", match.Candidate.BatchNormalizedHouseNos, wantHouseNo)
		}

		order2, err := createOrderConfirmCandidate(ctx, fixture, usecase, masterNo, match.Candidate.ID, match.Candidate.Version, nil)
		if err != nil {
			t.Fatalf("确认候选创建订单2失败: %v", err)
		}
		if order2.SeaMasterBill == nil || order2.SeaMasterBill.MasterBillID != match.Candidate.ID || order2.SeaMasterBill.MemberCount != 2 {
			t.Fatalf("订单2应自动入批且成员数为 2: %#v", order2.SeaMasterBill)
		}

		// 候选在保存前被并发变更（版本过期）：保留 409 兜底。
		staleVersion := match.Candidate.Version + 1
		_, err = createOrderConfirmCandidate(ctx, fixture, usecase, masterNo, match.Candidate.ID, staleVersion, nil)
		if !errors.Is(err, biz.ErrSeaMasterBillStatusConflict) {
			t.Fatalf("候选版本过期应返回状态冲突，实际: %v", err)
		}
	})

	t.Run("T2_命中含直单成员批次阻断加拼", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		masterNo := "COSCODIR" + strings.ToUpper(fixture.suffix[:8])

		// 批次首票显式直单（无其他成员，允许创建并独占主单）。
		input1 := fixture.validInput()
		direct := biz.SeaDocumentStructureDirect
		input1.SeaDocumentInput = &biz.SeaOrderDocumentInput{DocumentStructure: &direct}
		input1.SeaMasterBillInput.MasterNo = masterNo
		order1, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, input1)
		if err != nil {
			t.Fatalf("创建直单首票失败: %v", err)
		}
		if order1.SeaDocumentSummary == nil || order1.SeaDocumentSummary.DocumentStructure != biz.SeaDocumentStructureDirect {
			t.Fatalf("订单1应为直单: %#v", order1.SeaDocumentSummary)
		}

		match, err := usecase.MatchSeaMasterBillCandidate(ctx, fixture.organizationID, fixture.shippingLineID, masterNo, nil)
		if err != nil || !match.Matched {
			t.Fatalf("匹配直单批次候选失败: %v", err)
		}
		if len(match.Candidate.Members) != 1 || match.Candidate.Members[0].DocumentStructure != biz.SeaDocumentStructureDirect {
			t.Fatalf("候选成员单证结构应为 DIRECT: %#v", match.Candidate.Members)
		}

		// 含直单成员的批次禁止任何加拼（即使来票为 HOUSE）。
		_, err = createOrderConfirmCandidate(ctx, fixture, usecase, masterNo, match.Candidate.ID, match.Candidate.Version, nil)
		if kratoserrors.FromError(err).Reason != biz.ErrSeaMasterBillBatchDirectBlocked.Reason {
			t.Fatalf("含直单成员批次加拼应返回直单占用阻断，实际: %v", err)
		}
	})

	t.Run("T3_批次成员转直单受活动成员阻断", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		masterNo := "COSCOEXIT" + strings.ToUpper(fixture.suffix[:8])

		input1 := fixture.validInput()
		input1.SeaMasterBillInput.MasterNo = masterNo
		order1, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, input1)
		if err != nil {
			t.Fatalf("创建订单1失败: %v", err)
		}
		match, err := usecase.MatchSeaMasterBillCandidate(ctx, fixture.organizationID, fixture.shippingLineID, masterNo, nil)
		if err != nil || !match.Matched {
			t.Fatalf("匹配候选失败: %v", err)
		}
		order2, err := createOrderConfirmCandidate(ctx, fixture, usecase, masterNo, match.Candidate.ID, match.Candidate.Version, nil)
		if err != nil {
			t.Fatalf("确认候选创建订单2失败: %v", err)
		}

		hbl1, err := fixture.data.db.SeaHouseBill.Query().
			Where(seahousebillent.OrderIDEQ(order1.ID), seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED)).
			Only(ctx)
		if err != nil {
			t.Fatalf("读取订单1当前 HBL 失败: %v", err)
		}
		// 订单创建路径不生成版本快照；模式切换要求 current_version_id 已就绪。
		currentVersionID := ensureHouseBillCurrentVersion(t, fixture.data, fixture.actorID, hbl1.ID)
		// 本用例产生的不可变版本与模式切换事件必须先于夹具清理删除，
		// 否则 HBL 外键（NO ACTION）会阻断夹具行级清理。
		t.Cleanup(func() {
			cleanupCtx := context.Background()
			_, _ = fixture.data.db.SeaDocumentModeChangeEvent.Delete().
				Where(seadocumentmodechangeeventent.OrganizationIDEQ(fixture.organizationID)).
				Exec(cleanupCtx)
			_, _ = fixture.data.db.SeaHouseBillVersion.Delete().
				Where(seahousebillversionent.OrganizationIDEQ(fixture.organizationID)).
				Exec(cleanupCtx)
		})
		link1, err := fixture.data.db.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(fixture.organizationID),
				seamasterbillorderlinkent.OrderIDEQ(order1.ID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Only(ctx)
		if err != nil {
			t.Fatalf("读取订单1活动关联失败: %v", err)
		}

		blockedCmd := &biz.SeaDocumentModeChangeCommand{
			OrderID:                  order1.ID,
			ExpectedOrderVersion:     order1.Version,
			ExpectedLinkVersion:      link1.Version,
			ExpectedHouseBillVersion: &hbl1.Version,
			ExpectedCurrentVersionID: &currentVersionID,
			TargetMode:               biz.SeaDocumentStructureDirect,
			Reason:                   "测试批次成员转直单",
			IdempotencyKey:           "mode-exit-block-" + uuid.NewString(),
			Confirmation:             seaBatchTestConfirmation(),
		}
		audit := &biz.AuditEvent{OrganizationID: &fixture.organizationID, UserID: &fixture.actorID, Result: "success"}
		changeUC := biz.NewSeaDocumentChangeUsecase(NewSeaDocumentChangeRepo(fixture.data))
		err = changeUC.ExecuteModeChange(ctx, fixture.organizationID, fixture.actorID, blockedCmd, audit)
		if kratoserrors.FromError(err).Reason != biz.ErrSeaDocumentBatchMemberExitBlocked.Reason {
			t.Fatalf("批次还有其他活动成员时转直单应被阻断，实际: %v", err)
		}

		// 订单2 退出批次（模拟拆票/改派后的 ENDED 关系）后，仅剩自己允许转直单。
		link2, err := fixture.data.db.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(fixture.organizationID),
				seamasterbillorderlinkent.OrderIDEQ(order2.ID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Only(ctx)
		if err != nil {
			t.Fatalf("读取订单2活动关联失败: %v", err)
		}
		if _, err := fixture.data.db.SeaMasterBillOrderLink.UpdateOneID(link2.ID).
			SetStatus(seamasterbillorderlinkent.StatusENDED).
			SetEndedAt(time.Now().UTC()).
			SetEndedReason("集成测试退出批次").
			Save(ctx); err != nil {
			t.Fatalf("结束订单2活动关联失败: %v", err)
		}

		allowedCmd := *blockedCmd
		allowedCmd.IdempotencyKey = "mode-exit-allow-" + uuid.NewString()
		if err := changeUC.ExecuteModeChange(ctx, fixture.organizationID, fixture.actorID, &allowedCmd, audit); err != nil {
			t.Fatalf("批次仅剩自己时转直单应放行: %v", err)
		}
		voided, err := fixture.data.db.SeaHouseBill.Get(ctx, hbl1.ID)
		if err != nil || voided.Status != seahousebillent.StatusVOIDED {
			t.Fatalf("转直单后 HBL 应为 VOIDED: status=%s err=%v", voided.Status, err)
		}
	})

	t.Run("T4_批次内分单排重与跨批次放行", func(t *testing.T) {
		fixture := newOrderPostgresFixture(t, data)
		usecase := fixture.newUsecase()
		masterNo := "COSCODUP" + strings.ToUpper(fixture.suffix[:8])
		dupNo := "HBLDUP" + strings.ToUpper(fixture.suffix[:6])

		// 批次首票与第二票（不同分单号）。
		input1 := fixture.validInput()
		input1.SeaMasterBillInput.MasterNo = masterNo
		input1.SeaDocumentInput.HouseBill.HouseNo = dupNo
		order1, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, input1)
		if err != nil {
			t.Fatalf("创建批次首票失败: %v", err)
		}
		match, err := usecase.MatchSeaMasterBillCandidate(ctx, fixture.organizationID, fixture.shippingLineID, masterNo, nil)
		if err != nil || !match.Matched {
			t.Fatalf("匹配候选失败: %v", err)
		}
		order2, err := createOrderConfirmCandidate(ctx, fixture, usecase, masterNo, match.Candidate.ID, match.Candidate.Version, nil)
		if err != nil {
			t.Fatalf("创建批次第二票失败: %v", err)
		}

		// 同批次跨签发主体重号（小写变体 + 委托单位签发）：预查给出含冲突号的友好错误。
		duplicateInput := fixture.validInput()
		duplicateInput.SeaDocumentInput.HouseBill.HouseNo = strings.ToLower(dupNo)
		duplicateInput.SeaDocumentInput.HouseBill.IssuerSource = biz.SeaHouseBillIssuerSourceCustomerPartner
		_, err = createOrderConfirmCandidate(ctx, fixture, usecase, masterNo, match.Candidate.ID, match.Candidate.Version, duplicateInput.SeaDocumentInput)
		if kratoserrors.FromError(err).Reason != biz.ErrSeaHouseBillBatchNoDuplicate.Reason {
			t.Fatalf("同批次跨主体重号应返回批次排重错误，实际: %v", err)
		}
		if !strings.Contains(kratoserrors.FromError(err).Message, dupNo) {
			t.Fatalf("批次排重错误应包含冲突分单号: %s", kratoserrors.FromError(err).Message)
		}

		// 与作废分单同号同样禁止复用（一号一案，排重不过滤状态）。
		voidedNo := "HBLVOID" + strings.ToUpper(fixture.suffix[:6])
		if _, err := fixture.data.db.SeaHouseBill.Create().
			SetOrganizationID(fixture.organizationID).
			SetOrderID(order2.ID).
			SetMasterBillID(order1.SeaMasterBill.MasterBillID).
			SetHouseNo(voidedNo).
			SetNormalizedHouseNo(voidedNo).
			SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
			SetIssuerOrganizationID(fixture.organizationID).
			SetStatus(seahousebillent.StatusVOIDED).
			SetVersion(1).
			Save(ctx); err != nil {
			t.Fatalf("植入作废 HBL 失败: %v", err)
		}
		voidedInput := fixture.validInput()
		voidedInput.SeaDocumentInput.HouseBill.HouseNo = voidedNo
		_, err = createOrderConfirmCandidate(ctx, fixture, usecase, masterNo, match.Candidate.ID, match.Candidate.Version, voidedInput.SeaDocumentInput)
		if kratoserrors.FromError(err).Reason != biz.ErrSeaHouseBillBatchNoDuplicate.Reason {
			t.Fatalf("与作废分单同号应被批次排重阻断，实际: %v", err)
		}

		// 跨批次同号合法复用：全新 MBL + 首票同号保存成功。
		otherBatchInput := fixture.validInput()
		otherBatchInput.SeaMasterBillInput.MasterNo = "COSCOOTHER" + strings.ToUpper(fixture.suffix[:8])
		otherBatchInput.SeaDocumentInput.HouseBill.HouseNo = dupNo
		if _, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, otherBatchInput); err != nil {
			t.Fatalf("跨批次同号应保存成功: %v", err)
		}

		// 唯一索引并发兜底：绕过应用层预查直接写入同批次同号必须被数据库拒绝。
		link2, err := fixture.data.db.SeaMasterBillOrderLink.Query().
			Where(
				seamasterbillorderlinkent.OrganizationIDEQ(fixture.organizationID),
				seamasterbillorderlinkent.OrderIDEQ(order2.ID),
				seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
			).
			Only(ctx)
		if err != nil {
			t.Fatalf("读取订单2活动关联失败: %v", err)
		}
		order2HBL, err := fixture.data.db.SeaHouseBill.Query().
			Where(seahousebillent.OrderIDEQ(order2.ID), seahousebillent.StatusNotIn(seahousebillent.StatusVOIDED)).
			Only(ctx)
		if err != nil {
			t.Fatalf("读取订单2 HBL 失败: %v", err)
		}
		if _, err := fixture.data.db.SeaHouseBill.Delete().Where(seahousebillent.IDEQ(order2HBL.ID)).Exec(ctx); err != nil {
			t.Fatalf("清理订单2 HBL 失败: %v", err)
		}
		_, err = fixture.data.db.SeaHouseBill.Create().
			SetOrganizationID(fixture.organizationID).
			SetOrderID(order2.ID).
			SetMasterBillID(link2.MasterBillID).
			SetHouseNo(dupNo).
			SetNormalizedHouseNo(dupNo).
			SetIssuerSource(seahousebillent.IssuerSourceSELF_ORGANIZATION).
			SetStatus(seahousebillent.StatusDRAFT).
			SetVersion(1).
			Save(ctx)
		if !ent.IsConstraintError(err) {
			t.Fatalf("同批次同号直插应被唯一索引拒绝，实际: %v", err)
		}
	})

	t.Run("T5_存量重号迁移预检终止与干净库迁移成功", func(t *testing.T) {
		// 独立隔离 Schema：本测试自身的 getIntegrationData 已经从零执行完整迁移链
		// （含新唯一索引迁移），即干净库迁移成功路径。数据清理必须在夹具行级
		// 清理之后执行，因此以 t.Cleanup 注册而非 defer。
		data5, cleanup5 := getIntegrationData(t)
		t.Cleanup(cleanup5)
		fixture := newOrderPostgresFixture(t, data5)
		usecase := fixture.newUsecase()
		masterNo := "COSCOMIG" + strings.ToUpper(fixture.suffix[:8])
		dupNo := "HBLMIG" + strings.ToUpper(fixture.suffix[:6])

		input1 := fixture.validInput()
		input1.SeaMasterBillInput.MasterNo = masterNo
		input1.SeaDocumentInput.HouseBill.HouseNo = dupNo
		if _, err := usecase.Create(ctx, fixture.organizationID, fixture.actorID, input1); err != nil {
			t.Fatalf("创建迁移首票失败: %v", err)
		}
		match, err := usecase.MatchSeaMasterBillCandidate(ctx, fixture.organizationID, fixture.shippingLineID, masterNo, nil)
		if err != nil || !match.Matched {
			t.Fatalf("匹配候选失败: %v", err)
		}
		order2, err := createOrderConfirmCandidate(ctx, fixture, usecase, masterNo, match.Candidate.ID, match.Candidate.Version, nil)
		if err != nil {
			t.Fatalf("创建迁移第二票失败: %v", err)
		}

		// 模拟迁移前存量：删除唯一索引并把第二票分单号改写为与首票相同
		// （同批次跨签发主体重号）。
		if _, err := data5.sqlDB.ExecContext(ctx, `DROP INDEX "`+seaHouseBillBatchNoUniqueIndex+`"`); err != nil {
			t.Fatalf("删除批次唯一索引失败: %v", err)
		}
		if _, err := data5.sqlDB.ExecContext(
			ctx,
			`UPDATE "sea_house_bills" SET "house_no" = $1, "normalized_house_no" = $1 WHERE "order_id" = $2`,
			dupNo, order2.ID,
		); err != nil {
			t.Fatalf("改写存量分单号失败: %v", err)
		}

		migrationSQL, err := os.ReadFile(filepath.Join("..", "..", "migrations", seaHouseBillBatchNoUniqueMigration))
		if err != nil {
			t.Fatalf("读取迁移文件失败: %v", err)
		}
		_, err = data5.sqlDB.ExecContext(ctx, string(migrationSQL))
		if err == nil {
			t.Fatalf("存量同批次重号必须使迁移失败终止")
		}
		if !strings.Contains(err.Error(), "批次排重迁移已停止") || !strings.Contains(err.Error(), dupNo) {
			t.Fatalf("迁移失败应输出中文重号明细: %v", err)
		}
	})
}

// createOrderConfirmCandidate 以候选确认参数创建 HOUSE 订单。
// seaDocumentInput 为 nil 时使用夹具默认输入；非 nil 时仅覆盖分单输入。
func createOrderConfirmCandidate(
	ctx context.Context,
	fixture *orderPostgresFixture,
	usecase *biz.OrderUsecase,
	masterNo string,
	candidateID uuid.UUID,
	candidateVersion uint64,
	seaDocumentInput *biz.SeaOrderDocumentInput,
) (*biz.Order, error) {
	link, err := fixture.data.db.SeaMasterBillOrderLink.Query().
		Where(
			seamasterbillorderlinkent.OrganizationIDEQ(fixture.organizationID),
			seamasterbillorderlinkent.MasterBillIDEQ(candidateID),
			seamasterbillorderlinkent.StatusEQ(seamasterbillorderlinkent.StatusACTIVE),
		).
		Order(seamasterbillorderlinkent.ByID()).
		First(ctx)
	if err != nil {
		return nil, err
	}
	te, err := fixture.data.db.SeaTransportExecution.Get(ctx, link.TransportExecutionID)
	if err != nil {
		return nil, err
	}
	input := fixture.validInput()
	input.SeaMasterBillInput.MasterNo = masterNo
	input.SeaMasterBillInput.CandidateID = &candidateID
	input.SeaMasterBillInput.ExpectedCandidateVersion = &candidateVersion
	input.SeaMasterBillInput.CandidateTEID = &te.ID
	input.SeaMasterBillInput.ExpectedCandidateTEVersion = &te.Version
	if seaDocumentInput != nil {
		input.SeaDocumentInput = seaDocumentInput
	}
	return usecase.Create(ctx, fixture.organizationID, fixture.actorID, input)
}

// ensureHouseBillCurrentVersion 为 HBL 补当前不可变版本并返回版本 ID。
func ensureHouseBillCurrentVersion(t *testing.T, data *Data, actorID, hblID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := context.Background()
	var versionID uuid.UUID
	err := data.WithTx(ctx, func(tx *ent.Tx) error {
		hbl, err := tx.SeaHouseBill.Get(ctx, hblID)
		if err != nil {
			return err
		}
		version, err := createHouseVersion(ctx, tx, hbl, actorID, biz.VersionSourceOrderLock, nil, nil, nil, nil)
		if err != nil {
			return err
		}
		if _, err := hbl.Update().SetCurrentVersionID(version.ID).Save(ctx); err != nil {
			return err
		}
		versionID = version.ID
		return nil
	})
	if err != nil {
		t.Fatalf("补齐 HBL 当前不可变版本失败: %v", err)
	}
	return versionID
}

func seaBatchTestConfirmation() *biz.SeaExternalConfirmation {
	return &biz.SeaExternalConfirmation{
		ConfirmedByParty: "集成测试确认窗口",
		ConfirmedAt:      time.Now().UTC().Truncate(time.Second),
		ConfirmationNote: "集成测试外部确认",
	}
}
