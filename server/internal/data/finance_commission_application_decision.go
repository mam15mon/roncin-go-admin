package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	applicationent "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplication"
	applicationline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionapplicationline"
)

// lockApplicationHeaderForDecision 是财务批准/驳回共用的申请头锁定与门禁：
// FOR UPDATE 串行化并发决策，expected_version 与 PENDING_REVIEW 双重校验，
// 不匹配返回「该申请已被处理」稳定冲突。
func lockApplicationHeaderForDecision(ctx context.Context, client *ent.Client, organizationIDs []uuid.UUID, id uuid.UUID, expectedVersion uint64) (*ent.FinanceCommissionApplication, error) {
	header, err := client.FinanceCommissionApplication.Query().Where(
		applicationent.IDEQ(id),
		applicationent.OrganizationIDIn(organizationIDs...),
	).ForUpdate().Only(ctx)
	if err != nil {
		return nil, mapEntError(err, biz.ErrCommissionApplicationNotFound, nil)
	}
	if header.Version != expectedVersion || header.Status != applicationent.StatusPENDING_REVIEW {
		return nil, biz.ErrCommissionApplicationStatusConflict
	}
	return header, nil
}

// Approve 整单批准月度申请（design §5.2）。单个共享事务内顺序：
//  1. 锁定申请头并校验 expected_version 与 PENDING_REVIEW——并发批准/驳回/重提
//     只有一个事务成功，其余稳定冲突；
//  2. 按 commission_id 升序锁定申请明细，交由共享事务内的整批确认辅助
//     （confirmCommissionsForApplicationApproval）按「来源订单 → 提成父单」
//     固定锁序逐笔重算指纹、阻断未建账费用并把 DRAFT 提成整批转为 CONFIRMED；
//  3. 更新申请头为 APPROVED、记录决策人/时间、递增版本并写决策审计。
//     任一步失败整体回滚，不产生部分批准。
func (r *financeCommissionApplicationRepo) Approve(ctx context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64) (*biz.FinanceCommissionApplication, error) {
	var result *biz.FinanceCommissionApplication
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, clientErr := r.data.client(txCtx)
		if clientErr != nil {
			return clientErr
		}
		header, lockErr := lockApplicationHeaderForDecision(txCtx, client, organizationIDs, id, expectedVersion)
		if lockErr != nil {
			return lockErr
		}
		lines, lineErr := client.FinanceCommissionApplicationLine.Query().
			Where(applicationline.ApplicationIDEQ(header.ID)).
			Order(applicationline.ByCommissionID()).
			ForUpdate().All(txCtx)
		if lineErr != nil {
			return lineErr
		}
		if len(lines) == 0 {
			return biz.ErrCommissionApplicationSourceConflict
		}
		commissionIDs := make([]uuid.UUID, 0, len(lines))
		for _, line := range lines {
			commissionIDs = append(commissionIDs, line.CommissionID)
		}
		tx, txOK := txFromContext(txCtx)
		if !txOK {
			return biz.ErrCommissionApplicationInvalid
		}
		if confirmErr := confirmCommissionsForApplicationApproval(txCtx, tx, header.OrganizationID, decisionMaker, commissionIDs); confirmErr != nil {
			return confirmErr
		}
		total, amountErr := decimalOf(header.TotalCommissionAmount)
		if amountErr != nil {
			return amountErr
		}
		now := time.Now()
		updated, updateErr := client.FinanceCommissionApplication.UpdateOneID(header.ID).
			SetStatus(applicationent.StatusAPPROVED).
			SetVersion(header.Version + 1).
			SetDecidedAt(now).
			SetDecidedBy(decisionMaker).
			SetUpdatedAt(now).
			Save(txCtx)
		if updateErr != nil {
			return mapEntError(updateErr, biz.ErrCommissionApplicationNotFound, biz.ErrCommissionApplicationStatusConflict)
		}
		audit := biz.CommissionApplicationDecisionAudit(header.OrganizationID, decisionMaker, header.ID, header.EmployeeID,
			biz.CommissionApplicationDecisionAction(true), header.ApplicationMonth, header.CoverageTo, header.CommissionCount,
			total, header.Version+1, "")
		if auditErr := writeAudit(txCtx, client.AuditLog, audit); auditErr != nil {
			return auditErr
		}
		result = financeCommissionApplicationToBiz(updated)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// Reject 整单驳回月度申请（design §5.3）：与批准共用申请头锁定与版本门禁；
// 只把申请转为 REJECTED、记录决策审计并递增版本，明细与提成单保持不变
// （提成仍为 DRAFT，员工重提路径承接）；驳回原因必填由领域入口校验。
func (r *financeCommissionApplicationRepo) Reject(ctx context.Context, organizationIDs []uuid.UUID, decisionMaker, id uuid.UUID, expectedVersion uint64, reason string) (*biz.FinanceCommissionApplication, error) {
	var result *biz.FinanceCommissionApplication
	err := r.data.WithinTransaction(ctx, func(txCtx context.Context) error {
		client, clientErr := r.data.client(txCtx)
		if clientErr != nil {
			return clientErr
		}
		header, lockErr := lockApplicationHeaderForDecision(txCtx, client, organizationIDs, id, expectedVersion)
		if lockErr != nil {
			return lockErr
		}
		total, amountErr := decimalOf(header.TotalCommissionAmount)
		if amountErr != nil {
			return amountErr
		}
		now := time.Now()
		updated, updateErr := client.FinanceCommissionApplication.UpdateOneID(header.ID).
			SetStatus(applicationent.StatusREJECTED).
			SetVersion(header.Version + 1).
			SetDecidedAt(now).
			SetDecidedBy(decisionMaker).
			SetDecisionReason(reason).
			SetUpdatedAt(now).
			Save(txCtx)
		if updateErr != nil {
			return mapEntError(updateErr, biz.ErrCommissionApplicationNotFound, biz.ErrCommissionApplicationStatusConflict)
		}
		audit := biz.CommissionApplicationDecisionAudit(header.OrganizationID, decisionMaker, header.ID, header.EmployeeID,
			biz.CommissionApplicationDecisionAction(false), header.ApplicationMonth, header.CoverageTo, header.CommissionCount,
			total, header.Version+1, reason)
		if auditErr := writeAudit(txCtx, client.AuditLog, audit); auditErr != nil {
			return auditErr
		}
		result = financeCommissionApplicationToBiz(updated)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}
