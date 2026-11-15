import { describe, expect, it } from 'vitest';
import {
  applyAllocationChange,
  applyBillSelection,
  applyFeeChange,
  applyTotalChange,
  autoAllocate,
  availableFor,
  hasManualAllocation,
  newInvoiceRecord,
  rebuildUntouchedRecords,
  recordGaps,
  suggestTax,
} from './autofill';
import type {
  InvoiceRegistrationBillSource,
  InvoiceRegistrationDraft,
} from './types';
import { invoiceRegistrationPayload } from './types';

const single: InvoiceRegistrationBillSource = {
  id: 'bill',
  version: '3',
  billNo: 'BI-1',
  fees: [{ orderFeeId: 'fee', invoiceableAmount: '2000', sourceTaxRate: '6' }],
};
const mixed: InvoiceRegistrationBillSource[] = [
  {
    id: 'a',
    version: '1',
    fees: [
      { orderFeeId: 'a1', invoiceableAmount: '100', sourceTaxRate: '6' },
      { orderFeeId: 'a0', invoiceableAmount: '0', sourceTaxRate: '6' },
      { orderFeeId: 'a2', invoiceableAmount: '50.5', sourceTaxRate: '9' },
    ],
  },
  {
    id: 'b',
    version: '1',
    fees: [{ orderFeeId: 'b1', invoiceableAmount: '30' }],
  },
];

describe('建议税额', () => {
  it('含税口径按税率计算并四舍五入两位，未设置不建议、零税率建议0', () => {
    expect(suggestTax('1060', '6')).toBe('60.00');
    expect(suggestTax('100', '6')).toBe('5.66');
    expect(suggestTax('100', '0')).toBe('0.00');
    expect(suggestTax('100', undefined)).toBeUndefined();
    expect(suggestTax('', '6')).toBeUndefined();
    expect(suggestTax('1e3', '6')).toBeUndefined();
  });
});

describe('一账单一费用自动带入', () => {
  it('新票直接选好账单与费用，输入总额后带入分配与建议税额并可提交', () => {
    const record = newInvoiceRecord([single]);
    expect(record.allocations).toMatchObject([
      { billId: 'bill', fees: [{ orderFeeId: 'fee' }] },
    ]);
    const next = applyTotalChange(
      {
        ...record,
        totalAmount: '1060',
        invoiceProfileId: 'p',
        taxInvoiceNo: 'NO',
      },
      [single],
    );
    expect(next.allocations[0]).toMatchObject({
      amount: '1060',
      fees: [{ amount: '1060', taxAmount: '60.00', taxSource: 'auto' }],
    });
    const [payload] = invoiceRegistrationPayload([next], [single]);
    expect(payload.allocations?.[0]).toMatchObject({
      amount: '1060.00000000',
      expectedVersion: '3',
      fees: [{ amount: '1060.00000000', taxAmount: '60.00000000' }],
    });
    expect(JSON.stringify(payload)).not.toContain('Source');
  });
  it('无可登记余额时总额留空：单账单只预选账单，多账单不预选', () => {
    const empty = {
      ...mixed[0],
      fees: mixed[0].fees.map((fee) => ({ ...fee, invoiceableAmount: '0' })),
    };
    const record = newInvoiceRecord([empty]);
    expect(record.totalAmount).toBeUndefined();
    expect(record.allocations).toMatchObject([{ billId: 'a', fees: [] }]);
    expect(
      newInvoiceRecord([empty, { ...mixed[1], fees: [] }]).allocations,
    ).toEqual([]);
  });
  it('税率未设置时税额留空待财务填写', () => {
    const bill = {
      ...single,
      fees: [{ orderFeeId: 'fee', invoiceableAmount: '10' }],
    };
    const next = applyTotalChange(
      { ...newInvoiceRecord([bill]), totalAmount: '10' },
      [bill],
    );
    expect(next.allocations[0].fees[0]).toMatchObject({
      amount: '10',
      taxAmount: '',
    });
  });
});

describe('手改保护', () => {
  const base = () =>
    applyTotalChange({ ...newInvoiceRecord([single]), totalAmount: '1060' }, [
      single,
    ]);
  it('手改税额后改总额，税额不被覆盖，金额继续跟随', () => {
    const record = base();
    record.allocations[0].fees[0] = {
      ...record.allocations[0].fees[0],
      taxAmount: '59.99',
      taxSource: 'manual',
    };
    const next = applyTotalChange({ ...record, totalAmount: '530' }, [single]);
    expect(next.allocations[0].fees[0]).toMatchObject({
      amount: '530',
      taxAmount: '59.99',
    });
    expect(hasManualAllocation(next)).toBe(true);
  });
  it('手改费用金额后改总额，金额不变且差额可见', () => {
    let record = base();
    record.allocations[0].fees[0] = {
      ...record.allocations[0].fees[0],
      amount: '1000',
      amountSource: 'manual',
    };
    record = applyFeeChange(record, 0, 0, [single]);
    expect(record.allocations[0]).toMatchObject({
      amount: '1000',
      fees: [{ taxAmount: '56.60' }],
    });
    const next = applyTotalChange({ ...record, totalAmount: '1100' }, [single]);
    expect(next.allocations[0]).toMatchObject({
      amount: '1100',
      fees: [{ amount: '1000' }],
    });
    const gaps = recordGaps(next);
    expect(gaps.invoiceGap?.toFixed()).toBe('0');
    expect(gaps.billGaps[0]?.toFixed()).toBe('100');
    expect(() =>
      invoiceRegistrationPayload([{ ...next, taxInvoiceNo: 'NO' }], [single]),
    ).toThrow('费用合计须等于账单分配金额');
  });
  it('手改账单分配后改总额，分配保持并显示发票差额', () => {
    const record = base();
    record.allocations[0] = {
      ...record.allocations[0],
      amount: '800',
      amountSource: 'manual',
    };
    const changed = applyAllocationChange(record, 0, [single]);
    expect(changed.allocations[0].fees[0].amount).toBe('800');
    const next = applyTotalChange({ ...changed, totalAmount: '900' }, [single]);
    expect(next.allocations[0].amount).toBe('800');
    expect(recordGaps(next).invoiceGap?.toFixed()).toBe('100');
  });
});

describe('按可登记余额分配', () => {
  it('按顺序填至余额，跳过零余额，混合税率逐费建议，余额不足保留差额', () => {
    const record = autoAllocate(
      { ...newInvoiceRecord(mixed), totalAmount: '200' },
      mixed,
      undefined,
      0,
    );
    expect(record.allocations).toMatchObject([
      {
        billId: 'a',
        amount: '150.5',
        fees: [
          { orderFeeId: 'a1', amount: '100', taxAmount: '5.66' },
          { orderFeeId: 'a2', amount: '50.5', taxAmount: '4.17' },
        ],
      },
      {
        billId: 'b',
        amount: '30',
        fees: [{ orderFeeId: 'b1', taxAmount: '' }],
      },
    ]);
    expect(recordGaps(record).invoiceGap?.toFixed()).toBe('19.5');
  });
  it('扣除本会话其他发票占用，不超可登记余额', () => {
    const first = autoAllocate(
      { ...newInvoiceRecord(mixed), totalAmount: '120' },
      mixed,
      undefined,
      0,
    );
    const second = autoAllocate(
      { ...newInvoiceRecord(mixed), totalAmount: '100' },
      mixed,
      [first, newInvoiceRecord(mixed)],
      1,
    );
    expect(second.allocations[0].fees).toMatchObject([
      { orderFeeId: 'a2', amount: '30.5' },
    ]);
    expect(second.allocations[1].fees[0].amount).toBe('30');
    expect(
      availableFor(mixed[0].fees[2], 'a', [first, second], 1).toFixed(),
    ).toBe('30.5');
  });
  it('总额小于首项余额时只分配到首项', () => {
    const record = autoAllocate(
      { ...newInvoiceRecord(mixed), totalAmount: '40' },
      mixed,
      undefined,
      0,
    );
    expect(record.allocations).toMatchObject([
      { billId: 'a', amount: '40', fees: [{ orderFeeId: 'a1', amount: '40' }] },
    ]);
  });
});

describe('默认总额', () => {
  it('批量账单默认带出余额合计并分配平，逐费建议税额', () => {
    const record = newInvoiceRecord(mixed);
    expect(record).toMatchObject({ totalAmount: '180.5', totalSource: 'auto' });
    expect(recordGaps(record).invoiceGap?.toFixed()).toBe('0');
    expect(record.allocations.map((item) => item.amount)).toEqual([
      '150.5',
      '30',
    ]);
  });
  it('继续添加发票默认剩余余额，余额用尽时留空', () => {
    const first = applyTotalChange(
      {
        ...newInvoiceRecord([single]),
        totalAmount: '1500',
        totalSource: 'manual',
      },
      [single],
    );
    const second = newInvoiceRecord([single], [first]);
    expect(second.totalAmount).toBe('500');
    expect(second.allocations[0].fees[0]).toMatchObject({
      amount: '500',
      taxAmount: '28.30',
    });
    expect(
      newInvoiceRecord([single], [first, second]).totalAmount,
    ).toBeUndefined();
  });
  it('增减账单只重建未手改发票，保留表头；手改与修订原票不动', () => {
    const untouched = {
      ...newInvoiceRecord([mixed[0]]),
      taxInvoiceNo: 'NO-1',
      invoiceProfileId: 'p',
    };
    const manual = {
      ...newInvoiceRecord([mixed[0]]),
      totalAmount: '10',
      totalSource: 'manual' as const,
      allocations: [],
    };
    const original = {
      ...newInvoiceRecord([mixed[0]]),
      sourceInvoiceId: 'inv',
      allocations: [],
    };
    const [a, b, c] = rebuildUntouchedRecords(
      [untouched, manual, original],
      mixed,
    );
    expect(a).toMatchObject({ taxInvoiceNo: 'NO-1', invoiceProfileId: 'p' });
    expect(a.allocations.map((item) => item.billId)).toEqual(['a', 'b']);
    expect(b).toBe(manual);
    expect(c).toBe(original);
  });
  it('多张未手改发票按顺序重建，后序旧占用不挤占前序', () => {
    const one = newInvoiceRecord([mixed[1]]);
    const two = {
      ...newInvoiceRecord([mixed[1]]),
      totalAmount: '30',
      totalSource: 'auto' as const,
    };
    const [first, second] = rebuildUntouchedRecords([one, two], [mixed[1]]);
    expect(first.totalAmount).toBe('30');
    expect(second.totalAmount).toBeUndefined();
  });
});

describe('选择账单', () => {
  it('单可登记费用账单带入发票未分配金额且不超余额', () => {
    const bills = [mixed[1]];
    const record: InvoiceRegistrationDraft = {
      ...newInvoiceRecord(mixed),
      totalAmount: '50',
      allocations: [{ billId: 'b', amount: '', fees: [] }],
    };
    const next = applyBillSelection(record, 0, bills, [record], 0);
    expect(next.allocations[0]).toMatchObject({
      amount: '30',
      fees: [{ orderFeeId: 'b1', amount: '30' }],
    });
  });
});
