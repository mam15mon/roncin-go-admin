package data

import (
	"encoding/json"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
	commissionline "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
)

func commissionWithLinesToBiz(x *ent.FinanceCommission) (*biz.FinanceCommission, error) {
	result, err := commissionToBiz(x)
	if err != nil {
		return nil, err
	}
	result.Lines = make([]*biz.FinanceCommissionLine, 0, len(x.Edges.Lines))
	for _, line := range x.Edges.Lines {
		converted, convertErr := commissionLineToBiz(line)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Lines = append(result.Lines, converted)
	}
	result.Adjustments = make([]*biz.FinanceCommissionAdjustment, 0, len(x.Edges.Adjustments))
	for _, item := range x.Edges.Adjustments {
		converted, convertErr := commissionAdjustmentToBiz(item)
		if convertErr != nil {
			return nil, convertErr
		}
		result.Adjustments = append(result.Adjustments, converted)
		if converted.Status == biz.CommissionConfirmed || converted.Status == biz.CommissionPaid {
			if converted.Direction == biz.CommissionAdjustmentDecrease {
				result.AdjustmentAmount = result.AdjustmentAmount.Sub(converted.Amount)
			} else {
				result.AdjustmentAmount = result.AdjustmentAmount.Add(converted.Amount)
			}
		}
	}
	result.AdjustmentAmount = result.AdjustmentAmount.Round(8)
	result.EffectiveCommissionAmount = result.CommissionAmount.Add(result.AdjustmentAmount).Round(8)
	// CNY 调整金额动态继承主单汇率快照折算，草稿与已取消调整不计入。
	result.CNYAdjustmentAmount = result.AdjustmentAmount.Mul(result.CNYExchangeRate).Round(8)
	result.CNYEffectiveCommissionAmount = result.CNYCommissionAmount.Add(result.CNYAdjustmentAmount).Round(8)
	return result, nil
}

func commissionToBiz(x *ent.FinanceCommission) (*biz.FinanceCommission, error) {
	revenue, err := decimalOf(x.RealizedRevenue)
	if err != nil {
		return nil, err
	}
	cost, err := decimalOf(x.AllocatedCost)
	if err != nil {
		return nil, err
	}
	profit, err := decimalOf(x.RealizedProfit)
	if err != nil {
		return nil, err
	}
	rate, err := decimalOf(x.RatePercent)
	if err != nil {
		return nil, err
	}
	amount, err := decimalOf(x.CommissionAmount)
	if err != nil {
		return nil, err
	}
	commissionBase, err := decimalOf(x.CommissionBaseAmount)
	if err != nil {
		return nil, err
	}
	cnyRate, err := decimalOf(x.CnyExchangeRate)
	if err != nil {
		return nil, err
	}
	cnyAmount, err := decimalOf(x.CnyCommissionAmount)
	if err != nil {
		return nil, err
	}
	result := &biz.FinanceCommission{ID: x.ID, OrganizationID: x.OrganizationID, CommissionNo: x.CommissionNo, IdempotencyKey: x.IdempotencyKey, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName, CustomerCount: x.CustomerCount, OrderCount: x.OrderCount, FeeCount: x.FeeCount, Status: biz.CommissionStatus(x.Status), BaseCurrency: x.BaseCurrency, RealizedRevenue: revenue, AllocatedCost: cost, RealizedProfit: profit, CommissionBaseAmount: commissionBase, RatePercent: rate, CommissionAmount: amount, EffectiveCommissionAmount: amount, CommissionDate: x.CommissionDate, CNYExchangeRate: cnyRate, CNYExchangeRateSource: string(x.CnyExchangeRateSource), CNYExchangeRateDate: x.CnyExchangeRateDate, CNYExchangeRateSettingID: x.CnyExchangeRateSettingID, CNYCommissionAmount: cnyAmount, Note: x.Note, Version: x.Version, RuleVersion: x.RuleVersion, CalculationVersion: x.CalculationVersion, SourceFingerprint: x.SourceFingerprint, ConfirmedAt: x.ConfirmedAt, ConfirmedBy: x.ConfirmedBy, PaidAt: x.PaidAt, PaidBy: x.PaidBy, CancelledAt: x.CancelledAt, CancelledBy: x.CancelledBy, CancellationReason: x.CancellationReason, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt}
	if x.VerificationID != nil {
		result.VerificationID = *x.VerificationID
		result.VerificationNo = optionalStringValue(x.VerificationNo)
	}
	if x.NettingID != nil {
		result.NettingID = *x.NettingID
		result.NettingNo = optionalStringValue(x.NettingNo)
	}
	if x.Edges.Organization != nil {
		result.OrganizationName = x.Edges.Organization.Name
	}
	if x.RuleID != nil {
		result.RuleID = *x.RuleID
	}
	if x.RuleName != nil {
		result.RuleName = *x.RuleName
	}
	if x.PersonnelRole != nil {
		result.PersonnelRole = biz.CommissionPersonnelRole(*x.PersonnelRole)
	}
	if x.CalculationBasis != nil {
		result.CalculationBasis = biz.CommissionCalculationBasis(*x.CalculationBasis)
	}
	return result, nil
}

func commissionLineToBiz(x *ent.FinanceCommissionLine) (*biz.FinanceCommissionLine, error) {
	revenue, err := decimalOf(x.RealizedRevenue)
	if err != nil {
		return nil, err
	}
	cost, err := decimalOf(x.AllocatedCost)
	if err != nil {
		return nil, err
	}
	profit, err := decimalOf(x.RealizedProfit)
	if err != nil {
		return nil, err
	}
	rate, err := decimalOf(x.RatePercent)
	if err != nil {
		return nil, err
	}
	amount, err := decimalOf(x.CommissionAmount)
	if err != nil {
		return nil, err
	}
	commissionBase, err := decimalOf(x.CommissionBaseAmount)
	if err != nil {
		return nil, err
	}
	fees := make([]*biz.CommissionFeeDetail, 0, x.FeeCount)
	if err = json.Unmarshal([]byte(x.FeeSnapshot), &fees); err != nil {
		return nil, err
	}
	result := &biz.FinanceCommissionLine{ID: x.ID, OrganizationID: x.OrganizationID, CommissionID: x.CommissionID, OrderID: x.OrderID, OrderNo: x.OrderNo, OrderDate: x.OrderDate, CustomerID: x.CustomerID, CustomerCode: x.CustomerCode, CustomerName: x.CustomerName, CustomerAssignmentID: x.PersonnelAssignmentID, CustomerAssignmentOrganizationID: x.PersonnelOrganizationID, CustomerAssignedAt: x.PersonnelAssignedAt, EmployeeID: x.EmployeeID, EmployeeName: x.EmployeeName, PersonnelRole: biz.CommissionPersonnelRole(x.PersonnelRole), CalculationBasis: biz.CommissionCalculationBasis(x.CalculationBasis), BaseCurrency: x.BaseCurrency, RealizedRevenue: revenue, AllocatedCost: cost, RealizedProfit: profit, CommissionBaseAmount: commissionBase, RatePercent: rate, CommissionAmount: amount, FeeCount: x.FeeCount, Fees: fees, CreatedAt: x.CreatedAt, UpdatedAt: x.UpdatedAt, SnapshotStatus: snapshotStatusOrEmpty(x.SnapshotStatus), SnapshotSource: snapshotSourceOrEmpty(x.SnapshotSource)}
	if x.TotalReceivableSnapshot != nil {
		receivable, parseErr := decimalOf(*x.TotalReceivableSnapshot)
		if parseErr != nil {
			return nil, parseErr
		}
		result.TotalReceivableSnapshot = receivable
	}
	if x.TotalPayableSnapshot != nil {
		payable, parseErr := decimalOf(*x.TotalPayableSnapshot)
		if parseErr != nil {
			return nil, parseErr
		}
		result.TotalPayableSnapshot = payable
	}
	return result, nil
}

// snapshotStatusOrEmpty / snapshotSourceOrEmpty 把可空快照枚举转换为领域字符串；
// 全空存量行为空串，调用方必须按「正向判定 == READY」处理，不得放行全空行。
func snapshotStatusOrEmpty(value *commissionline.SnapshotStatus) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func snapshotSourceOrEmpty(value *commissionline.SnapshotSource) string {
	if value == nil {
		return ""
	}
	return string(*value)
}
