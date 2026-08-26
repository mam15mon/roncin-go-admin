import Decimal from 'decimal.js';

export type VerificationBalanceCandidate = {
  id: string;
  balance: string;
};

export type VerificationAllocationDraft = {
  cashflowId: string;
  billId: string;
  amount: string;
};

const money = (value?: string) => {
  try {
    return new Decimal(value || 0);
  } catch {
    return new Decimal(0);
  }
};

export function buildVerificationAllocations(
  cashflows: VerificationBalanceCandidate[],
  bills: VerificationBalanceCandidate[],
): VerificationAllocationDraft[] {
  const allocations: VerificationAllocationDraft[] = [];
  const billBalances = new Map(
    bills.map((bill) => [bill.id, money(bill.balance)]),
  );

  for (const cashflow of cashflows) {
    let cashBalance = money(cashflow.balance);
    if (!cashBalance.isPositive()) continue;
    for (const bill of bills) {
      const billBalance = billBalances.get(bill.id) || new Decimal(0);
      if (!billBalance.isPositive()) continue;
      const amount = Decimal.min(cashBalance, billBalance);
      allocations.push({
        cashflowId: cashflow.id,
        billId: bill.id,
        amount: amount.toFixed(8),
      });
      cashBalance = cashBalance.minus(amount);
      billBalances.set(bill.id, billBalance.minus(amount));
      if (!cashBalance.isPositive()) break;
    }
  }
  return allocations;
}

export function sumVerificationAmounts(values: string[]) {
  return values.reduce((total, value) => total.plus(money(value)), new Decimal(0));
}

export function isPositiveVerificationAmount(value?: string) {
  return money(value).isPositive();
}
