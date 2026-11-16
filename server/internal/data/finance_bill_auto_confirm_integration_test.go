package data

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	auditlogent "github.com/roncin/roncin-go-admin/server/internal/data/ent/auditlog"
	financebillent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financebill"
	financenettingent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenetting"
	nettingallocent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financenettingallocation"
	"github.com/roncin/roncin-go-admin/server/internal/platform/migration"
)

// TestFinanceBillAutoConfirmPostgres 验证账单免人工确认：单张建账直接为已确认且创建人即确认人；
// 已有发票、核销或已确认对冲的账单只能改备注、对账标题与到期日，草稿对冲不构成保护。
func TestFinanceBillAutoConfirmPostgres(t *testing.T) {
	data, cleanup := getIntegrationData(t)
	t.Cleanup(cleanup)
	ctx := context.Background()

	t.Run("单张建账直接为已确认", func(t *testing.T) {
		fixture := newFinanceBillPostgresFixture(t, data)
		feeID := fixture.createUnbilledFee("auto-confirm")
		created, err := fixture.newUsecase(NewFinanceBillRepo(data)).Create(ctx, fixture.organizationID, fixture.actorID, biz.CreateFinanceBillInput{
			FeeIDs: []uuid.UUID{feeID}, BillDate: financeBillIntegrationDate, IdempotencyKey: "bill-auto-confirm-" + fixture.suffix, SettlementAccountID: fixture.accountID,
		})
		if err != nil {
			t.Fatalf("建账失败: %v", err)
		}
		if created.Status != biz.FinanceBillConfirmed || created.ConfirmedBy == nil || *created.ConfirmedBy != fixture.actorID || created.ConfirmedAt == nil || created.Version != 1 {
			t.Fatalf("新账单应直接为已确认且创建人即确认人: %#v", created)
		}
	})

	t.Run("受保护账单只允许改备注、对账标题与到期日", func(t *testing.T) {
		f := newFeeLedgerPostgresFixture(t, data)
		usecase := f.newUsecase(NewFinanceBillRepo(data))
		feeID := f.createLedgerFee("protected-update", "100")
		bill := f.createConfirmedBill("protected-update", "100", financeBillIntegrationDate)
		f.linkBillFee("protected-update", bill, feeID, "100")
		netting, allocation := f.createNetting("protected-update", financenettingent.StatusCONFIRMED, bill, "100", true)

		_, err := usecase.Update(ctx, []uuid.UUID{f.organizationID}, f.actorID, biz.UpdateFinanceBillInput{
			ID: bill.ID, ExpectedVersion: bill.Version, BillDate: "2026-08-31", SettlementAccountID: f.accountID,
		})
		if !errors.Is(err, biz.ErrFinanceBillProtectedFields) {
			t.Fatalf("已确认对冲后改账单日期应拒绝，实际 %v", err)
		}
		unchanged, err := data.db.FinanceBill.Get(ctx, bill.ID)
		if err != nil || unchanged.Version != bill.Version || unchanged.BillDate != bill.BillDate {
			t.Fatalf("拒绝时不得写入: %#v %v", unchanged, err)
		}

		note, title, dueDate := "受保护后补充备注", "受保护对账抬头", "2026-09-30"
		updated, err := usecase.Update(ctx, []uuid.UUID{f.organizationID}, f.actorID, biz.UpdateFinanceBillInput{
			ID: bill.ID, ExpectedVersion: bill.Version, BillDate: financeBillIntegrationDate, SettlementAccountID: f.accountID,
			Note: &note, StatementTitle: &title, DueDate: &dueDate,
		})
		if err != nil {
			t.Fatalf("受保护账单改备注失败: %v", err)
		}
		if updated.Version != bill.Version+1 || updated.Note == nil || *updated.Note != note || updated.StatementTitle == nil || *updated.StatementTitle != title || updated.DueDate == nil || *updated.DueDate != dueDate {
			t.Fatalf("受保护更新字段错误: %#v", updated)
		}
		if updated.TotalAmount.StringFixed(8) != bill.TotalAmount || updated.BaseCurrencyAmount.StringFixed(8) != bill.BaseCurrencyAmount || updated.ExchangeRate.StringFixed(8) != bill.ExchangeRate || updated.BillDate != bill.BillDate {
			t.Fatalf("受保护更新不得改金额或汇率: %#v", updated)
		}

		// 撤销对冲后保护解除，可按账单日期重算（同币种汇率为 1）。
		if _, err := netting.Update().SetStatus(financenettingent.StatusREVERSED).SetReversedAt(time.Now()).Save(ctx); err != nil {
			t.Fatal(err)
		}
		if _, err := data.db.FinanceNettingAllocation.Update().Where(nettingallocent.IDEQ(allocation.ID)).SetActive(false).Save(ctx); err != nil {
			t.Fatal(err)
		}
		released, err := usecase.Update(ctx, []uuid.UUID{f.organizationID}, f.actorID, biz.UpdateFinanceBillInput{
			ID: bill.ID, ExpectedVersion: updated.Version, BillDate: "2026-08-31", SettlementAccountID: f.accountID,
		})
		if err != nil {
			t.Fatalf("撤销对冲后改账单日期应允许: %v", err)
		}
		if released.BillDate != "2026-08-31" || released.Version != updated.Version+1 {
			t.Fatalf("解除保护后更新结果错误: %#v", released)
		}
	})

	t.Run("草稿对冲不阻止改账单日期", func(t *testing.T) {
		f := newFeeLedgerPostgresFixture(t, data)
		usecase := f.newUsecase(NewFinanceBillRepo(data))
		feeID := f.createLedgerFee("draft-netting-update", "100")
		bill := f.createConfirmedBill("draft-netting-update", "100", financeBillIntegrationDate)
		f.linkBillFee("draft-netting-update", bill, feeID, "100")
		f.createNetting("draft-netting-update", financenettingent.StatusDRAFT, bill, "100", false)
		updated, err := usecase.Update(ctx, []uuid.UUID{f.organizationID}, f.actorID, biz.UpdateFinanceBillInput{
			ID: bill.ID, ExpectedVersion: bill.Version, BillDate: "2026-08-31", SettlementAccountID: f.accountID,
		})
		if err != nil || updated.BillDate != "2026-08-31" {
			t.Fatalf("草稿对冲不应阻止改账单日期: %#v %v", updated, err)
		}
	})
}

const financeBillAutoConfirmMigration = "20261114100000"

// TestFinanceBillAutoConfirmMigrationPostgres 在正式迁移前态构造草稿账单，验证迁移把草稿转为已确认，
// 确认人按批次创建人或最早成功建账审计回填、确认时间取创建时间；非草稿账单保持不变，重复执行不再写入。
func TestFinanceBillAutoConfirmMigrationPostgres(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, financeBillAutoConfirmMigration)
	t.Cleanup(cleanup)
	ctx := context.Background()
	client := data.db
	org, partner, batchCreator, auditCreator, laterActor := seedAutoConfirmMigrationBase(t, data, "OK")

	// 迁移前态的批次头仍含后续迁移删除的汇总列，当前 Ent Schema 已不声明这些列，故以正式前态列直接写入。
	batchID := uuid.New()
	if _, err := data.sqlDB.ExecContext(ctx, `INSERT INTO "finance_bill_batches"("id","created_at","updated_at","organization_id","batch_no","idempotency_key","request_hash","fee_count","bill_count","total_base_amount","base_currency","created_by") VALUES ($1,now(),now(),$2,'BG-AUTO',$3,$4,1,1,100,'CNY',$5)`,
		batchID, org, uuid.NewString(), strings.Repeat("a", 64), batchCreator); err != nil {
		t.Fatal(err)
	}
	batchBill := createAutoConfirmMigrationBill(t, data, org, partner, "BATCH", financebillent.StatusDRAFT, &batchID)
	auditBill := createAutoConfirmMigrationBill(t, data, org, partner, "AUDIT", financebillent.StatusDRAFT, nil)
	for _, item := range []struct {
		actor  uuid.UUID
		result auditlogent.Result
		at     time.Time
	}{{laterActor, auditlogent.ResultSuccess, time.Now().Add(-time.Minute)}, {auditCreator, auditlogent.ResultSuccess, time.Now().Add(-time.Hour)}, {laterActor, auditlogent.ResultFailure, time.Now().Add(-2 * time.Hour)}} {
		if _, err := client.AuditLog.Create().SetOrganizationID(org).SetUserID(item.actor).SetAction("finance.bill.create").SetResult(item.result).
			SetResourceType("finance_bill").SetResourceID(auditBill.ID.String()).SetCreatedAt(item.at).Save(ctx); err != nil {
			t.Fatal(err)
		}
	}
	confirmedBill := createAutoConfirmMigrationBill(t, data, org, partner, "CONFIRMED", financebillent.StatusCONFIRMED, nil)

	migrationDir := filepath.Join("..", "..", "migrations")
	for attempt := 0; attempt < 2; attempt++ {
		if err := migration.Apply(ctx, data.sqlDB, migrationDir); err != nil {
			t.Fatalf("第 %d 次执行迁移失败: %v", attempt+1, err)
		}
		for _, expected := range []struct {
			name  string
			bill  uuid.UUID
			actor uuid.UUID
		}{{"批次账单", batchBill.ID, batchCreator}, {"单张账单", auditBill.ID, auditCreator}} {
			stored, err := client.FinanceBill.Get(ctx, expected.bill)
			if err != nil {
				t.Fatal(err)
			}
			if stored.Status != financebillent.StatusCONFIRMED || stored.ConfirmedBy == nil || *stored.ConfirmedBy != expected.actor || stored.ConfirmedAt == nil || !stored.ConfirmedAt.Equal(stored.CreatedAt) || stored.Version != 2 {
				t.Fatalf("%s 回填错误（第 %d 次）: %#v", expected.name, attempt+1, stored)
			}
		}
		stored, err := client.FinanceBill.Get(ctx, confirmedBill.ID)
		if err != nil {
			t.Fatal(err)
		}
		if stored.Version != confirmedBill.Version || stored.ConfirmedBy != nil || !stored.UpdatedAt.Equal(confirmedBill.UpdatedAt) {
			t.Fatalf("非草稿账单不得改写: %#v", stored)
		}
	}
}

// TestFinanceBillAutoConfirmMigrationRejectsUnknownCreatorPostgres 验证无法确定创建人时迁移显式失败并整体回滚。
func TestFinanceBillAutoConfirmMigrationRejectsUnknownCreatorPostgres(t *testing.T) {
	data, cleanup := getIntegrationDataBeforeMigration(t, financeBillAutoConfirmMigration)
	t.Cleanup(cleanup)
	ctx := context.Background()
	org, partner, _, _, failedActor := seedAutoConfirmMigrationBase(t, data, "MISSING")
	orphan := createAutoConfirmMigrationBill(t, data, org, partner, "ORPHAN", financebillent.StatusDRAFT, nil)
	// 失败审计不能作为创建人依据。
	if _, err := data.db.AuditLog.Create().SetOrganizationID(org).SetUserID(failedActor).SetAction("finance.bill.create").SetResult(auditlogent.ResultFailure).
		SetResourceType("finance_bill").SetResourceID(orphan.ID.String()).Save(ctx); err != nil {
		t.Fatal(err)
	}
	err := migration.Apply(ctx, data.sqlDB, filepath.Join("..", "..", "migrations"))
	if err == nil || !strings.Contains(err.Error(), "无法确定创建人") {
		t.Fatalf("无法回填创建人时迁移必须失败，实际 %v", err)
	}
	var status string
	var confirmedBy *uuid.UUID
	var version uint64
	getErr := data.sqlDB.QueryRowContext(ctx, `SELECT status,confirmed_by,version FROM finance_bills WHERE id=$1`, orphan.ID).Scan(&status, &confirmedBy, &version)
	if getErr != nil || status != string(financebillent.StatusDRAFT) || confirmedBy != nil || version != orphan.Version {
		t.Fatalf("迁移失败后草稿账单必须保持原样: status=%s confirmedBy=%v version=%d err=%v", status, confirmedBy, version, getErr)
	}
}

func seedAutoConfirmMigrationBase(t *testing.T, data *Data, code string) (org, partner, batchCreator, auditCreator, otherActor uuid.UUID) {
	t.Helper()
	ctx := context.Background()
	organization, err := data.db.Organization.Create().SetCode("AUTO-" + code).SetName("账单免确认迁移-" + code).SetKind("company").SetBaseCurrency("CNY").Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	counterparty, err := data.db.Partner.Create().SetOrganizationID(organization.ID).SetCode("AUTO-" + code).SetLegalName("免确认客户-" + code).SetNormalizedName("免确认客户-" + code).Save(ctx)
	if err != nil {
		t.Fatal(err)
	}
	users := make([]uuid.UUID, 0, 3)
	for _, name := range []string{"批次创建人", "单张创建人", "其他操作人"} {
		user, err := data.db.User.Create().SetDisplayName(name + "-" + code).Save(ctx)
		if err != nil {
			t.Fatal(err)
		}
		users = append(users, user.ID)
	}
	return organization.ID, counterparty.ID, users[0], users[1], users[2]
}

func createAutoConfirmMigrationBill(t *testing.T, data *Data, org, partner uuid.UUID, key string, status financebillent.Status, batchID *uuid.UUID) *entFinanceBillSnapshot {
	t.Helper()
	ctx := context.Background()
	create := withTestFinanceBillSettlementAccountSnapshot(data.db.FinanceBill.Create(), uuid.New(), "CNY").
		SetOrganizationID(org).SetBillNo("AUTO-" + key).SetIdempotencyKey(uuid.NewString()).SetDirection(financebillent.DirectionRECEIVABLE).
		SetStatus(status).SetSettlementPartyID(partner).SetSettlementPartyName("免确认客户").SetCurrency("CNY").SetBaseCurrency("CNY").
		SetExchangeRate("1").SetExchangeRateSource(financebillent.ExchangeRateSourceSYSTEM).SetExchangeRateDate(financeBillIntegrationDate).
		SetTotalAmount("100").SetNetAmount("100").SetTaxAmount("0").SetBaseCurrencyAmount("100").SetFeeCount(1).SetBillDate(financeBillIntegrationDate).
		SetNillableBatchID(batchID).SetVersion(1).SetCreatedAt(time.Now().Add(-24 * time.Hour).Truncate(time.Microsecond))
	bill, err := insertMigrationFixture[ent.FinanceBill](ctx, data, "finance_bills", create.Mutation())
	if err != nil {
		t.Fatalf("创建迁移前态账单 %s: %v", key, err)
	}
	// 重读数据库值作为快照，避免内存时间精度与数据库微秒精度不一致。
	stored := bill
	return &entFinanceBillSnapshot{ID: stored.ID, Version: stored.Version, UpdatedAt: stored.UpdatedAt}
}

type entFinanceBillSnapshot struct {
	ID        uuid.UUID
	Version   uint64
	UpdatedAt time.Time
}
