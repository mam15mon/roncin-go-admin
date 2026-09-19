package data

import (
	"github.com/roncin/roncin-go-admin/server/internal/biz"
	"github.com/roncin/roncin-go-admin/server/internal/data/ent"
)

func financeCommissionApplicationToBiz(item *ent.FinanceCommissionApplication) *biz.FinanceCommissionApplication {
	total, _ := decimalOf(item.TotalCommissionAmount)
	totalCNY, _ := decimalOf(item.TotalCnyCommissionAmount)
	return &biz.FinanceCommissionApplication{
		ID:                       item.ID,
		OrganizationID:           item.OrganizationID,
		EmployeeID:               item.EmployeeID,
		ApplicationMonth:         item.ApplicationMonth,
		CoverageTo:               item.CoverageTo,
		Status:                   biz.CommissionApplicationStatus(item.Status),
		Version:                  item.Version,
		CommissionCount:          item.CommissionCount,
		BaseCurrency:             item.BaseCurrency,
		TotalCommissionAmount:    total,
		TotalCNYCommissionAmount: totalCNY,
		SubmittedAt:              item.SubmittedAt,
		SubmittedBy:              item.SubmittedBy,
		DecidedAt:                item.DecidedAt,
		DecidedBy:                item.DecidedBy,
		DecisionReason:           item.DecisionReason,
		CreatedAt:                item.CreatedAt,
		UpdatedAt:                item.UpdatedAt,
	}
}

func financeCommissionApplicationLineToBiz(item *ent.FinanceCommissionApplicationLine) (*biz.FinanceCommissionApplicationLine, error) {
	amount, err := decimalOf(item.CommissionAmount)
	if err != nil {
		return nil, err
	}
	cnyAmount, err := decimalOf(item.CnyCommissionAmount)
	if err != nil {
		return nil, err
	}
	personnelRole := ""
	if item.PersonnelRole != "" {
		personnelRole = item.PersonnelRole
	}
	return &biz.FinanceCommissionApplicationLine{
		ID:                  item.ID,
		OrganizationID:      item.OrganizationID,
		EmployeeID:          item.EmployeeID,
		ApplicationID:       item.ApplicationID,
		CommissionID:        item.CommissionID,
		CommissionDate:      item.CommissionDate,
		VerificationID:      item.VerificationID,
		VerificationNo:      item.VerificationNo,
		NettingID:           item.NettingID,
		NettingNo:           item.NettingNo,
		PersonnelRole:       biz.CommissionPersonnelRole(personnelRole),
		RuleID:              item.RuleID,
		RuleVersion:         item.RuleVersion,
		RuleName:            item.RuleName,
		CalculationBasis:    item.CalculationBasis,
		BaseCurrency:        item.BaseCurrency,
		CommissionAmount:    amount,
		CNYCommissionAmount: cnyAmount,
		SourceFingerprint:   item.SourceFingerprint,
		CreatedAt:           item.CreatedAt,
	}, nil
}
