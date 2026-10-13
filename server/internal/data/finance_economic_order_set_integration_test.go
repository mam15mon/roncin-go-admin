package data

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"

	"github.com/roncin/roncin-go-admin/server/internal/biz"
	line "github.com/roncin/roncin-go-admin/server/internal/data/ent/financecommissionline"
)

func TestFinanceEconomicOrderSetExpansionPostgres(t *testing.T) {
	f := newFeeSupplementFixture(t)
	a, b, c := f.createOrder("ORDER-SET-A-"+f.suffix), f.createOrder("ORDER-SET-B-"+f.suffix), f.createOrder("ORDER-SET-C-"+f.suffix)
	f.approveInitialFees(a)
	f.approveInitialFees(b)
	f.approveInitialFees(c)
	parent := f.createConfirmedCommissionWithSnapshot(a, "60", "10", "1000", "400")
	billIDs, err := f.data.db.FinanceBillLine.Query().All(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	ids := make([]uuid.UUID, 0, len(billIDs))
	for _, l := range billIDs {
		if l.OrderID == a.ID {
			ids = append(ids, l.BillID)
		}
	}
	pre, err := commissionEconomicAffectedOrderIDs(f.ctx, f.data.db, f.organizationID, ids)
	if err != nil || len(pre) != 1 || pre[0] != a.ID {
		t.Fatalf("预读应仅A：%v,%v", pre, err)
	}
	original, err := f.data.db.FinanceCommissionLine.Query().Where(line.CommissionIDEQ(parent.ID)).Only(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	// 表达预读之后才可见的跨票父提成：B行的经济快照不参与本测试，
	// 只核验资金写入必须持有这个父单全部订单锁。
	_, err = f.data.db.FinanceCommissionLine.Create().SetOrganizationID(f.organizationID).SetCommissionID(parent.ID).SetOrderID(b.ID).SetOrderNo(b.OrderNo).SetOrderDate(original.OrderDate).SetCustomerID(original.CustomerID).SetCustomerCode(original.CustomerCode).SetCustomerName(original.CustomerName).SetPersonnelAssignmentID(original.PersonnelAssignmentID).SetPersonnelOrganizationID(original.PersonnelOrganizationID).SetPersonnelAssignedAt(original.PersonnelAssignedAt).SetFeeCount(0).SetEmployeeID(original.EmployeeID).SetEmployeeName(original.EmployeeName).SetPersonnelRole(original.PersonnelRole).SetCalculationBasis(original.CalculationBasis).SetBaseCurrency(original.BaseCurrency).SetRealizedRevenue("0").SetAllocatedCost("0").SetRealizedProfit("0").SetCommissionBaseAmount("0").SetRatePercent(original.RatePercent).SetCommissionAmount("0").Save(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	// B同时属于BC父单，A的完整关联集合必须继续展开到C。
	bc := f.createConfirmedCommissionWithSnapshot(b, "60", "10", "1000", "400")
	_, err = f.data.db.FinanceCommissionLine.Create().SetOrganizationID(f.organizationID).SetCommissionID(bc.ID).SetOrderID(c.ID).SetOrderNo(c.OrderNo).SetOrderDate(original.OrderDate).SetCustomerID(original.CustomerID).SetCustomerCode(original.CustomerCode).SetCustomerName(original.CustomerName).SetPersonnelAssignmentID(original.PersonnelAssignmentID).SetPersonnelOrganizationID(original.PersonnelOrganizationID).SetPersonnelAssignedAt(original.PersonnelAssignedAt).SetFeeCount(0).SetEmployeeID(original.EmployeeID).SetEmployeeName(original.EmployeeName).SetPersonnelRole(original.PersonnelRole).SetCalculationBasis(original.CalculationBasis).SetBaseCurrency(original.BaseCurrency).SetRealizedRevenue("0").SetAllocatedCost("0").SetRealizedProfit("0").SetCommissionBaseAmount("0").SetRatePercent(original.RatePercent).SetCommissionAmount("0").Save(f.ctx)
	if err != nil {
		t.Fatal(err)
	}
	err = f.data.WithinTransaction(f.ctx, func(ctx context.Context) error {
		client, err := f.data.client(ctx)
		if err != nil {
			return err
		}
		if err = lockFinanceEconomicOrders(ctx, client, pre); err != nil {
			return err
		}
		return validateFinanceEconomicOrderSet(ctx, client, f.organizationID, ids, pre)
	})
	if !errors.Is(err, biz.ErrOrderStatusConflict) {
		t.Fatalf("预读A而父单已扩为A/B/C时应409整笔重试，得到%v", err)
	}
	fresh, err := commissionEconomicAffectedOrderIDs(f.ctx, f.data.db, f.organizationID, ids)
	if err != nil || len(fresh) != 3 {
		t.Fatalf("刷新应包含A/B/C：%v,%v", fresh, err)
	}
	err = f.data.WithinTransaction(f.ctx, func(ctx context.Context) error {
		client, err := f.data.client(ctx)
		if err != nil {
			return err
		}
		if err = lockFinanceEconomicOrders(ctx, client, fresh); err != nil {
			return err
		}
		return validateFinanceEconomicOrderSet(ctx, client, f.organizationID, ids, fresh)
	})
	if err != nil {
		t.Fatal("刷新完整集合后应允许继续", err)
	}
}
