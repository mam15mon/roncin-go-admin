import { describe, expect, it } from 'vitest';
import { hasBothBillDirections, hasMixedBillDirections } from './index';

function fee(direction?: string) {
  return { direction } as API.FeeLedgerItem;
}

// 普通账单与对冲账单的模式准入互斥：
// - 单方向：普通可用，对冲禁用并提示至少各一笔；
// - 双方向：普通禁用并提示分别建账，对冲可用。
describe('费用台账建账模式准入', () => {
  it('仅应收：普通可用、对冲不可用', () => {
    const rows = [fee('RECEIVABLE'), fee('RECEIVABLE')];
    expect(hasMixedBillDirections(rows)).toBe(false);
    expect(hasBothBillDirections(rows)).toBe(false);
  });

  it('仅应付：普通可用、对冲不可用', () => {
    const rows = [fee('PAYABLE'), fee('PAYABLE')];
    expect(hasMixedBillDirections(rows)).toBe(false);
    expect(hasBothBillDirections(rows)).toBe(false);
  });

  it('应收 + 应付：普通禁用、对冲可用', () => {
    const rows = [fee('RECEIVABLE'), fee('PAYABLE')];
    expect(hasMixedBillDirections(rows)).toBe(true);
    expect(hasBothBillDirections(rows)).toBe(true);
  });

  it('缺少方向的费用不进入任何模式判断', () => {
    expect(hasMixedBillDirections([fee(), fee('RECEIVABLE')])).toBe(false);
    expect(hasBothBillDirections([fee(), fee('RECEIVABLE')])).toBe(false);
  });
});
