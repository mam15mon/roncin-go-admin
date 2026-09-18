package data

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/shopspring/decimal"
)

// CommissionSnapshotBackfillVersion 标识存量提成行复算快照回填的算法版本：
// 截止时点判定、证据范围或复算口径变化时必须递增，旧 READY 行不回刷。
const CommissionSnapshotBackfillVersion = "COMMISSION_SNAPSHOT_BACKFILL_V1"

// 存量行标记 UNAVAILABLE 时的稳定原因码：迁移报告与补录审批阻断提示共同消费。
const (
	// snapshotUnavailableCutoffTime 提成行缺少可用的首次写入时间，无法确定唯一截止时点。
	snapshotUnavailableCutoffTime = "CUTOFF_TIME_UNAVAILABLE"
	// snapshotUnsupportedCalculationVersion 原提成 calculation_version/口径未被登记，
	// 不存在可路由的历史纯计算器，失败关闭。
	snapshotUnsupportedCalculationVersion = "CALCULATION_VERSION_UNSUPPORTED"
	// snapshotFeeSnapshotInvalid 提成行费用快照不可解析或快照金额非法。
	snapshotFeeSnapshotInvalid = "FEE_SNAPSHOT_INVALID"
	// snapshotBillLineNotTraceable 账单行缺少可判定的停用时间等时间证据，
	// 无法确定其在截止时点是否活动。
	snapshotBillLineNotTraceable = "BILL_LINE_NOT_TRACEABLE"
	// snapshotRecomputeMismatch 按历史纯计算器复算的 allocated_cost/realized_profit/
	// commission_amount 与已存值 8 位精度不一致（含历史事实曾被原地修改）。
	snapshotRecomputeMismatch = "RECOMPUTE_MISMATCH"
)

// CommissionSnapshotBackfillReport 是一次回填执行的汇总报告，不含敏感报文。
type CommissionSnapshotBackfillReport struct {
	BackfillVersion     string
	TotalLines          int
	ReadyCount          int
	UnavailableCount    int
	UnavailableByReason map[string]int
}

// snapshotBackfillLine 是一条待回填的存量提成行输入。
type snapshotBackfillLine struct {
	id                 uuid.UUID
	calculationVersion string
	createdAt          time.Time
	realizedRevenue    string
	allocatedCost      string
	realizedProfit     string
	commissionAmount   string
	ratePercent        string
	calculationBasis   string
	feeSnapshot        []byte
}

// snapshotBillLineEvidence 是参与分母重建的账单行时间证据。
type snapshotBillLineEvidence struct {
	id                 uuid.UUID
	orderFeeID         uuid.UUID
	baseCurrencyAmount string
	active             bool
	createdAt          time.Time
	billCancelledAt    sql.NullTime
}

// BackfillCommissionLineSnapshots 对存量提成行确定性回填历史分母复算快照
// （design 第 8 节）。以原提成行首次计算并写入 commission_amount 的
// created_at 为唯一截止时点；费用成员与金额取自提成行不可变费用快照，
// 账单行本位币取自截止时点前已生效且仍可追溯的账单行事实；复算
// allocated_cost/realized_profit/commission_amount 与已存值按 8 位精度核对，
// 一致才写 READY + MIGRATED + 回填算法版本 + 证据集合哈希，否则标记
// UNAVAILABLE + 稳定原因码。禁止读取迁移时点当前订单费用汇总或当前规则配置。
// 写入带 snapshot_status IS NULL 守卫，可安全重入；报告不含敏感报文。
func BackfillCommissionLineSnapshots(ctx context.Context, db *sql.DB) (*CommissionSnapshotBackfillReport, error) {
	report := &CommissionSnapshotBackfillReport{
		BackfillVersion:     CommissionSnapshotBackfillVersion,
		UnavailableByReason: map[string]int{},
	}
	lastID := uuid.Nil
	for {
		lines, err := loadSnapshotBackfillBatch(ctx, db, lastID, 200)
		if err != nil {
			return nil, err
		}
		if len(lines) == 0 {
			break
		}
		evidence, err := loadSnapshotBillLineEvidence(ctx, db, lines)
		if err != nil {
			return nil, err
		}
		for i := range lines {
			if err := backfillSnapshotLine(ctx, db, &lines[i], evidence, report); err != nil {
				return nil, err
			}
		}
		lastID = lines[len(lines)-1].id
	}
	return report, nil
}

// loadSnapshotBackfillBatch 按 id 升序读取一批 snapshot_status 为空的存量行
// 及其父单 calculation_version。
func loadSnapshotBackfillBatch(ctx context.Context, db *sql.DB, afterID uuid.UUID, limit int) ([]snapshotBackfillLine, error) {
	rows, err := db.QueryContext(ctx, `
SELECT l.id, c.calculation_version, l.created_at,
       l.realized_revenue, l.allocated_cost, l.realized_profit, l.commission_amount,
       l.rate_percent, l.calculation_basis, l.fee_snapshot
FROM finance_commission_lines l
JOIN finance_commissions c ON c.id = l.commission_id
WHERE l.snapshot_status IS NULL AND l.id > $1
ORDER BY l.id
LIMIT $2`, afterID, limit)
	if err != nil {
		return nil, fmt.Errorf("读取存量提成行回填批次: %w", err)
	}
	defer rows.Close()
	result := make([]snapshotBackfillLine, 0, limit)
	for rows.Next() {
		var line snapshotBackfillLine
		if err := rows.Scan(&line.id, &line.calculationVersion, &line.createdAt,
			&line.realizedRevenue, &line.allocatedCost, &line.realizedProfit, &line.commissionAmount,
			&line.ratePercent, &line.calculationBasis, &line.feeSnapshot); err != nil {
			return nil, fmt.Errorf("扫描存量提成行回填批次: %w", err)
		}
		result = append(result, line)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历存量提成行回填批次: %w", err)
	}
	return result, nil
}

// loadSnapshotBillLineEvidence 一次性读取本批次涉及费用的全部账单行事实，
// 只取参与判定所需的时间与金额列。
func loadSnapshotBillLineEvidence(ctx context.Context, db *sql.DB, lines []snapshotBackfillLine) (map[uuid.UUID][]snapshotBillLineEvidence, error) {
	feeIDs := make(map[uuid.UUID]struct{})
	for i := range lines {
		fees, err := decodeSnapshotFees(lines[i].feeSnapshot)
		if err != nil {
			// 快照不可解析的行不需要账单行证据，按行级原因码处理。
			continue
		}
		for _, fee := range fees {
			feeIDs[fee.FeeID] = struct{}{}
		}
	}
	if len(feeIDs) == 0 {
		return map[uuid.UUID][]snapshotBillLineEvidence{}, nil
	}
	ids := make([]uuid.UUID, 0, len(feeIDs))
	for id := range feeIDs {
		ids = append(ids, id)
	}
	rows, err := db.QueryContext(ctx, `
SELECT bl.id, bl.order_fee_id, bl.base_currency_amount, bl.active, bl.created_at, b.cancelled_at
FROM finance_bill_lines bl
JOIN finance_bills b ON b.id = bl.bill_id
WHERE bl.order_fee_id = ANY($1::uuid[])`, ids)
	if err != nil {
		return nil, fmt.Errorf("读取提成行账单行证据: %w", err)
	}
	defer rows.Close()
	result := make(map[uuid.UUID][]snapshotBillLineEvidence)
	for rows.Next() {
		var item snapshotBillLineEvidence
		var amount string
		if err := rows.Scan(&item.id, &item.orderFeeID, &amount, &item.active, &item.createdAt, &item.billCancelledAt); err != nil {
			return nil, fmt.Errorf("扫描提成行账单行证据: %w", err)
		}
		item.baseCurrencyAmount = amount
		result[item.orderFeeID] = append(result[item.orderFeeID], item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历提成行账单行证据: %w", err)
	}
	return result, nil
}

// decodeSnapshotFees 解析提成行不可变费用快照（与生成路径 biz.CommissionFeeDetail
// 同构）。
func decodeSnapshotFees(raw []byte) ([]*biz.CommissionFeeDetail, error) {
	fees := make([]*biz.CommissionFeeDetail, 0, 8)
	if err := json.Unmarshal(raw, &fees); err != nil {
		return nil, err
	}
	return fees, nil
}

// backfillSnapshotLine 处理单条存量行：重建截止时点分母并复核已存结果，写入
// READY+MIGRATED 或 UNAVAILABLE。行级失败按原因码落库，不中断整批。
func backfillSnapshotLine(ctx context.Context, db *sql.DB, line *snapshotBackfillLine, evidence map[uuid.UUID][]snapshotBillLineEvidence, report *CommissionSnapshotBackfillReport) error {
	report.TotalLines++
	receivable, payable, hash, reason, err := rebuildSnapshotDenominators(line, evidence)
	if err != nil {
		return err
	}
	if reason != "" {
		if err := markSnapshotUnavailable(ctx, db, line.id, reason); err != nil {
			return err
		}
		report.UnavailableCount++
		report.UnavailableByReason[reason]++
		return nil
	}
	if err := markSnapshotReady(ctx, db, line.id, receivable, payable, hash); err != nil {
		return err
	}
	report.ReadyCount++
	return nil
}

// rebuildSnapshotDenominators 按截止时点重建历史总应收/总应付分母并复核已存
// 计算结果。reason 非空表示该行不可还原，receivable/payable/hash 无效。
func rebuildSnapshotDenominators(line *snapshotBackfillLine, evidence map[uuid.UUID][]snapshotBillLineEvidence) (decimal.Decimal, decimal.Decimal, string, string, error) {
	zero := decimal.Zero
	// 截止时点唯一性：原提成行首次计算并写入 commission_amount 的 created_at。
	if line.id == uuid.Nil || line.createdAt.IsZero() {
		return zero, zero, "", snapshotUnavailableCutoffTime, nil
	}
	// 历史版本纯计算器路由：未知 calculation_version 或口径失败关闭，不猜测。
	if line.calculationVersion != biz.CommissionCalculationVersion ||
		(line.calculationBasis != string(biz.CommissionBasisRealizedProfit) && line.calculationBasis != string(biz.CommissionBasisRealizedRevenue)) {
		return zero, zero, "", snapshotUnsupportedCalculationVersion, nil
	}
	fees, err := decodeSnapshotFees(line.feeSnapshot)
	if err != nil {
		return zero, zero, "", snapshotFeeSnapshotInvalid, nil
	}
	cutoff := line.createdAt.UTC()
	hashParts := []string{
		"commission-snapshot-backfill/" + strings.ToLower(strings.TrimPrefix(CommissionSnapshotBackfillVersion, "COMMISSION_SNAPSHOT_")),
		"calculation_version|" + line.calculationVersion,
		"cutoff|" + cutoff.Format(time.RFC3339Nano),
	}
	totalReceivable := decimal.Zero
	totalPayable := decimal.Zero
	for _, fee := range fees {
		// 费用快照与 biz.CommissionFeeDetail 同构，本位币金额已是 decimal。
		feeBase := fee.BaseCurrencyAmount
		hashParts = append(hashParts, fmt.Sprintf("FEE|%s|%s|%s", fee.FeeID, fee.Direction, feeBase.StringFixed(8)))
		// 与生成口径一致：费用已建账时分母取账单行本位币快照，未建账回落到
		// 费用自身快照。账单行必须在截止时点前已存在且活动状态可判定。
		aggregate := decimal.Zero
		hasBillBase := false
		for _, billLine := range evidence[fee.FeeID] {
			if billLine.createdAt.After(cutoff) {
				// 截止时点后才建账/改账产生的账单行，不进入历史分母。
				continue
			}
			if billLine.active {
				// 当前活动且早于截止时点：截止时点必然活动。
			} else if !billLine.billCancelledAt.Valid {
				// 已停用但无停用时间证据：无法判定停用是否发生在截止时点前。
				return zero, zero, "", snapshotBillLineNotTraceable, nil
			} else if billLine.billCancelledAt.Time.UTC().After(cutoff) {
				// 截止时点后停用：截止时点仍活动。
			} else {
				// 截止时点前已停用：当时未建账。
				continue
			}
			base, parseErr := decimalOf(billLine.baseCurrencyAmount)
			if parseErr != nil {
				return zero, zero, "", snapshotBillLineNotTraceable, nil
			}
			aggregate = aggregate.Add(base)
			hasBillBase = true
			hashParts = append(hashParts, fmt.Sprintf("BILL_LINE|%s|%s", billLine.id, base.StringFixed(8)))
		}
		if !hasBillBase {
			aggregate = feeBase
		}
		if fee.Direction == "RECEIVABLE" {
			totalReceivable = totalReceivable.Add(aggregate)
		} else {
			totalPayable = totalPayable.Add(aggregate)
		}
	}
	totalReceivable = totalReceivable.Round(8)
	totalPayable = totalPayable.Round(8)
	// 证据集合哈希：参与重建的截止时点与事实集合的规范编码，按字典序排序后哈希。
	sort.Strings(hashParts)
	digest := sha256.Sum256([]byte(strings.Join(hashParts, "\n")))
	// 复核：按历史纯计算器复算并与已存值 8 位精度比对，任何漂移都判不可还原，
	// 不覆盖原提成金额、不填写猜测分母。
	realized, parseErr := decimalOf(line.realizedRevenue)
	if parseErr != nil {
		return zero, zero, "", snapshotRecomputeMismatch, nil
	}
	rate, parseErr := decimalOf(line.ratePercent)
	if parseErr != nil {
		return zero, zero, "", snapshotRecomputeMismatch, nil
	}
	storedCost, parseErr := decimalOf(line.allocatedCost)
	if parseErr != nil {
		return zero, zero, "", snapshotRecomputeMismatch, nil
	}
	storedProfit, parseErr := decimalOf(line.realizedProfit)
	if parseErr != nil {
		return zero, zero, "", snapshotRecomputeMismatch, nil
	}
	storedAmount, parseErr := decimalOf(line.commissionAmount)
	if parseErr != nil {
		return zero, zero, "", snapshotRecomputeMismatch, nil
	}
	cost, profit, _, amount, calculateErr := biz.CalculateCommissionLine(realized, totalReceivable, totalPayable, rate, biz.CommissionCalculationBasis(line.calculationBasis))
	if calculateErr != nil ||
		cost.StringFixed(8) != storedCost.StringFixed(8) ||
		profit.StringFixed(8) != storedProfit.StringFixed(8) ||
		amount.StringFixed(8) != storedAmount.StringFixed(8) {
		return zero, zero, "", snapshotRecomputeMismatch, nil
	}
	return totalReceivable, totalPayable, hex.EncodeToString(digest[:]), "", nil
}

// markSnapshotReady 写入 READY + MIGRATED 快照：完整分母、回填算法版本与证据
// 集合哈希，满足数据库 CHECK 的一致性要求；带 snapshot_status IS NULL 守卫保证重入幂等。
func markSnapshotReady(ctx context.Context, db *sql.DB, id uuid.UUID, receivable, payable decimal.Decimal, hash string) error {
	_, err := db.ExecContext(ctx, `
UPDATE finance_commission_lines
SET total_receivable_snapshot = $2, total_payable_snapshot = $3,
    snapshot_status = 'READY', snapshot_source = 'MIGRATED',
    snapshot_backfill_version = $4, snapshot_evidence_hash = $5
WHERE id = $1 AND snapshot_status IS NULL`, id, receivable.StringFixed(8), payable.StringFixed(8), CommissionSnapshotBackfillVersion, hash)
	if err != nil {
		return fmt.Errorf("写入提成行 READY 快照: %w", err)
	}
	return nil
}

// markSnapshotUnavailable 标记不可还原行：不伪填分母，只记录稳定原因码。
func markSnapshotUnavailable(ctx context.Context, db *sql.DB, id uuid.UUID, reason string) error {
	_, err := db.ExecContext(ctx, `
UPDATE finance_commission_lines
SET snapshot_status = 'UNAVAILABLE', snapshot_unavailable_reason_code = $2
WHERE id = $1 AND snapshot_status IS NULL`, id, reason)
	if err != nil {
		return fmt.Errorf("标记提成行 UNAVAILABLE 快照: %w", err)
	}
	return nil
}
