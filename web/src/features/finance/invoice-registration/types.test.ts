import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  type InvoiceRegistrationBillSource,
  type InvoiceRegistrationDraft,
  invoiceBillSource,
  invoiceRegistrationPayload,
} from './types';

const source: InvoiceRegistrationBillSource = {
  id: 'bill',
  version: '3',
  currency: 'CNY',
  fees: [{ orderFeeId: 'fee', totalAmount: '100', invoiceableAmount: '80' }],
};
const record = (): InvoiceRegistrationDraft => ({
  invoiceProfileId: 'profile',
  invoiceType: 'NORMAL',
  taxInvoiceNo: ' NO-1 ',
  invoiceDate: dayjs('2026-11-14'),
  totalAmount: '30',
  note: '  ',
  allocations: [
    {
      billId: 'bill',
      amount: '30.00',
      fees: [{ orderFeeId: 'fee', amount: '30.000', taxAmount: '1.00000001' }],
    },
  ],
});
describe('发票登记载荷', () => {
  it('规范化同一意图，保留手填税额与已有账单版本', () => {
    const input = record();
    expect(invoiceRegistrationPayload([input], [source])).toEqual([
      {
        invoiceProfileId: 'profile',
        invoiceType: 'NORMAL',
        taxInvoiceNo: 'NO-1',
        invoiceDate: '2026-11-14',
        totalAmount: '30.00000000',
        note: undefined,
        allocations: [
          {
            billId: 'bill',
            expectedVersion: '3',
            amount: '30.00000000',
            fees: [
              {
                orderFeeId: 'fee',
                amount: '30.00000000',
                taxAmount: '1.00000001',
              },
            ],
          },
        ],
      },
    ]);
    expect(input.allocations[0].fees[0].taxAmount).toBe('1.00000001');
  });
  it.each(['0', '', 'abc', '1.000000001', '1e2', '-1'])(
    '拒绝非法正金额%s，不静默舍入',
    (value) => {
      const input = record();
      input.totalAmount = value;
      expect(() => invoiceRegistrationPayload([input], [source])).toThrow();
    },
  );
  it('税额0合法，超含税额及精度越界拒绝', () => {
    const input = record();
    input.allocations[0].fees[0].taxAmount = '0';
    expect(
      invoiceRegistrationPayload([input], [source])[0].allocations?.[0]
        .fees?.[0].taxAmount,
    ).toBe('0.00000000');
    for (const tax of ['30.00000001', '-1', '0.000000001', '']) {
      input.allocations[0].fees[0].taxAmount = tax;
      expect(() => invoiceRegistrationPayload([input], [source])).toThrow();
    }
  });
  it('逐层合计、重复费与跨多票累计余额均验证', () => {
    const input = record();
    input.allocations[0].amount = '29';
    expect(() => invoiceRegistrationPayload([input], [source])).toThrow(
      '费用合计',
    );
    input.allocations[0].amount = '30';
    input.totalAmount = '29';
    expect(() => invoiceRegistrationPayload([input], [source])).toThrow(
      '账单分配合计',
    );
    input.totalAmount = '30';
    input.allocations[0].fees.push({ ...input.allocations[0].fees[0] });
    expect(() => invoiceRegistrationPayload([input], [source])).toThrow(
      '来源缺失或重复',
    );
    const second = record();
    second.taxInvoiceNo = 'NO-2';
    expect(() =>
      invoiceRegistrationPayload(
        [record(), second],
        [{ ...source, fees: [{ ...source.fees[0], invoiceableAmount: '50' }] }],
      ),
    ).toThrow('累计分配');
  });
  it('wire来源税率未设与显式0区分，非活动费用和缺业务ID显式处理', () => {
    const base: API.FinanceBill = {
      id: 'bill',
      version: '1',
      lines: [
        { active: true, orderFeeId: 'unknown', totalAmount: '10' },
        { active: true, orderFeeId: 'zero', totalAmount: '5', taxRate: '0' },
        { orderFeeId: 'inactive', totalAmount: '3' },
      ],
    };
    const mapped = invoiceBillSource(base);
    expect(mapped.fees).toHaveLength(2);
    expect(mapped.fees[0].sourceTaxRate).toBeUndefined();
    expect(mapped.fees[1].sourceTaxRate).toBe('0');
    expect(() => invoiceBillSource({ ...base, id: undefined })).toThrow('身份');
    expect(() =>
      invoiceBillSource({ ...base, lines: [{ active: true }] }),
    ).toThrow('业务标识');
  });
});
