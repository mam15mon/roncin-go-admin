package data

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"

	"github.com/roncin/roncin-go-admin/server/internal/access"
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commissionent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommission"
	commissionadjustmentent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionadjustment"
	commissionlineent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
	orderent "github.com/roncin/roncin-go-admin/server/internal/data/ent/order"
)

// financialLockEvidenceVersion 标识财务锁证据规范编码算法版本；编码语义变化时
// 必须递增，旧申请按其固化版本与哈希复核，避免跨版本误判。
const financialLockEvidenceVersion = "FIN_LOCK_EVIDENCE_V1"

// financialLockEvidenceComponents 是财务锁证据的参与事实集合：与财务锁净额
// 口径（financeCommissionLockNetAmount / financeLockedOrderPredicate）完全同源，
// 仅统计参与净额的 CONFIRMED/PAID 提成行与调整行。
type financialLockEvidenceComponents struct {
	lines       []*ent.FinanceCommissionLine
	adjustments []*ent.FinanceCommissionAdjustment
}

// hash 按版本化规范编码计算证据哈希：组件类型 + 主键 + 方向 + 符号化 8 位金额，
// 排序后编码；CONFIRMED 与 PAID 统一编码为 ACTIVE（状态不进入编码），同一提成
// 或调整在 CONFIRMED 与 PAID 之间流转不改变证据。
func (c *financialLockEvidenceComponents) hash() (string, error) {
	parts := []string{"fin-lock-evidence/" + strings.ToLower(strings.TrimPrefix(financialLockEvidenceVersion, "FIN_LOCK_EVIDENCE_"))}
	for _, line := range c.lines {
		amount, err := decimalOf(line.CommissionAmount)
		if err != nil {
			return "", err
		}
		parts = append(parts, fmt.Sprintf("COMMISSION_LINE|%s|ACTIVE|%s", line.ID, amount.StringFixed(8)))
	}
	for _, adjustment := range c.adjustments {
		amount, err := decimalOf(adjustment.Amount)
		if err != nil {
			return "", err
		}
		sign := "+"
		if adjustment.Direction == commissionadjustmentent.DirectionDECREASE {
			sign = "-"
		}
		parts = append(parts, fmt.Sprintf("COMMISSION_ADJUSTMENT|%s|%s|%s%s", adjustment.ID, adjustment.Direction, sign, amount.StringFixed(8)))
	}
	sort.Strings(parts)
	digest := sha256.Sum256([]byte(strings.Join(parts, "\n")))
	return hex.EncodeToString(digest[:]), nil
}

// netAmount 复用台账财务锁净额的 Go 口径，禁止另写一套锁定公式。
func (c *financialLockEvidenceComponents) netAmount() (decimal.Decimal, error) {
	return financeCommissionLockNetAmount(c.lines, c.adjustments)
}

// loadFinancialLockEvidence 在当前事务客户端内读取参与净额的提成行与调整行。
func loadFinancialLockEvidence(ctx context.Context, client *ent.Client, organizationID, orderID uuid.UUID) (*financialLockEvidenceComponents, error) {
	lines, err := client.FinanceCommissionLine.Query().Where(
		commissionlineent.OrganizationIDEQ(organizationID),
		commissionlineent.OrderIDEQ(orderID),
		commissionlineent.HasCommissionWith(commissionent.StatusIn(commissionent.StatusCONFIRMED, commissionent.StatusPAID)),
	).WithCommission().All(ctx)
	if err != nil {
		return nil, err
	}
	adjustments, err := client.FinanceCommissionAdjustment.Query().Where(
		commissionadjustmentent.OrganizationIDEQ(organizationID),
		commissionadjustmentent.OrderIDEQ(orderID),
		commissionadjustmentent.StatusIn(commissionadjustmentent.StatusCONFIRMED, commissionadjustmentent.StatusPAID),
	).All(ctx)
	if err != nil {
		return nil, err
	}
	return &financialLockEvidenceComponents{lines: lines, adjustments: adjustments}, nil
}

// lockOrderForSupplementInternal 在事务内锁定订单行并计算当前锁事实。补录是
// 业务锁与提成净额财务锁的受控例外：这里不执行普通费用写入口的内容门禁与
// 财务锁拒绝，但财务证据计算必须复用净额口径并在 Order 行锁内读取。
func lockOrderForSupplementInternal(ctx context.Context, tx *ent.Tx, organizationID, orderID uuid.UUID) (*biz.OrderFeeSupplementLockEvidence, error) {
	order, err := tx.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).ForUpdate().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrOrderNotFound, nil)
	}
	businessType, err := orderAccessBusinessType(order.BusinessType)
	if err != nil {
		return nil, err
	}
	components, err := loadFinancialLockEvidence(ctx, tx.Client(), organizationID, orderID)
	if err != nil {
		return nil, err
	}
	hash, err := components.hash()
	if err != nil {
		return nil, err
	}
	net, err := components.netAmount()
	if err != nil {
		return nil, err
	}
	return &biz.OrderFeeSupplementLockEvidence{
		OrderNo:                  order.OrderNo,
		BusinessType:             string(businessType),
		BusinessLocked:           order.LockedAt != nil,
		BusinessLockGeneration:   order.LockGeneration,
		FinancialLocked:          net.Sign() > 0,
		FinancialEvidenceVersion: financialLockEvidenceVersion,
		FinancialEvidenceHash:    hash,
		FinancialNetAmount:       net,
	}, nil
}

func (r *orderFeeSupplementRepo) LockOrderForSupplement(ctx context.Context, organizationID, orderID uuid.UUID) (*biz.OrderFeeSupplementLockEvidence, error) {
	var evidence *biz.OrderFeeSupplementLockEvidence
	err := r.data.WithTx(ctx, func(tx *ent.Tx) error {
		result, lockErr := lockOrderForSupplementInternal(ctx, tx, organizationID, orderID)
		if lockErr != nil {
			return lockErr
		}
		evidence = result
		return nil
	})
	if err != nil {
		return nil, err
	}
	return evidence, nil
}

func (r *orderFeeSupplementRepo) ListLockGrantApprovers(ctx context.Context, organizationID, orderID uuid.UUID) ([]biz.OrderFeeSupplementApprover, error) {
	ref, err := r.GetOrderRef(ctx, organizationID, orderID)
	if err != nil {
		return nil, err
	}
	businessType, err := orderAccessBusinessType(orderent.BusinessType(ref.BusinessType))
	if err != nil {
		return nil, err
	}
	var result []biz.OrderFeeSupplementApprover
	err = r.data.WithTx(ctx, func(tx *ent.Tx) error {
		candidates, candidateErr := queryQualifiedBusinessLockCandidates(ctx, tx.Client(), organizationID, businessType)
		if candidateErr != nil {
			return candidateErr
		}
		result = make([]biz.OrderFeeSupplementApprover, 0, len(candidates))
		for _, candidate := range candidates {
			result = append(result, biz.OrderFeeSupplementApprover{UserID: candidate.UserID, DisplayName: candidate.DisplayName, DingTalkUserID: candidate.DingTalkUserIDSnapshot})
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// hasRealtimeLockGrantWithClient 复用订单直接解锁资格的唯一契约：bootstrap
// admin 显式具备资格；普通用户按统一 lock grant 谓词判定。
func hasRealtimeLockGrantWithClient(ctx context.Context, client *ent.Client, organizationID, userID uuid.UUID, businessType access.OrderBusinessType, isBootstrapAdmin bool) (bool, error) {
	if isBootstrapAdmin {
		return true, nil
	}
	return isUserQualifiedBusinessLockRole(ctx, client, organizationID, userID, businessType)
}

func (r *orderFeeSupplementRepo) HasRealtimeLockGrant(ctx context.Context, organizationID, orderID, userID uuid.UUID, isBootstrapAdmin bool) (bool, error) {
	client, err := r.data.client(ctx)
	if err != nil {
		return false, err
	}
	order, err := client.Order.Query().Where(orderent.IDEQ(orderID), orderent.OrganizationIDEQ(organizationID)).Only(ctx)
	if err != nil {
		return false, mapEntError(err, biz.ErrFeeSupplementNotFound, nil)
	}
	businessType, err := orderAccessBusinessType(order.BusinessType)
	if err != nil {
		return false, err
	}
	return hasRealtimeLockGrantWithClient(ctx, client, organizationID, userID, businessType, isBootstrapAdmin)
}

// ---------------------------------------------------------------------------
// 申请持久化
// ---------------------------------------------------------------------------
