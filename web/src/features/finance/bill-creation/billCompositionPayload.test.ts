import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import {
  billBatchItemLabel,
  billGroupOperationPayload,
  type GroupFormValue,
  normalizeBillBatchResponse,
  requestMessage,
} from './billWorkbenchHelpers';

const group: API.BillBatchPreviewGroup = {
  groupKey: 'group-ar',
  totalAmount: '100',
  fees: [{ id: 'fee-1', totalAmount: '100' }],
};
function values(): GroupFormValue {
  return {
    statementTitle: '客户',
    invoices: [40, 60].map((amount, index) => ({
      invoiceProfileId: 'profile',
      invoiceType: 'DIGITAL_NORMAL',
      taxInvoiceNo: `TAX-${index}`,
      invoiceDate: dayjs('2026-11-14'),
      totalAmount: String(amount),
      allocations: [
        {
          billId: 'group-ar',
          amount: String(amount),
          fees: [
            { orderFeeId: 'fee-1', amount: String(amount), taxAmount: '0' },
          ],
        },
      ],
    })),
    verifications: [
      {
        verificationDate: dayjs('2026-11-14'),
        allocations: [{ cashflowId: 'cash-old', amount: '60' }],
      },
      {
        verificationDate: dayjs('2026-11-14'),
        shortfallEnabled: true,
        shortfall: {
          amount: '40',
          transactionDate: dayjs('2026-11-14'),
          ourAccount: ' 收款账户 ',
          paymentMethod: 'BANK_TRANSFER',
        },
      },
    ],
  };
}

describe('建账组合载荷与逐项失败回显', () => {
  it('多票多次核销和差额共用本组费用，提交不包含临时账单ID或版本', () => {
    const input = values();
    const payload = billGroupOperationPayload(group, input);
    expect(payload.invoices).toHaveLength(2);
    expect(payload.invoices?.[0]).toEqual({
      invoiceProfileId: 'profile',
      invoiceType: 'DIGITAL_NORMAL',
      taxInvoiceNo: 'TAX-0',
      invoiceDate: '2026-11-14',
      totalAmount: '40.00000000',
      fees: [
        { orderFeeId: 'fee-1', amount: '40.00000000', taxAmount: '0.00000000' },
      ],
    });
    expect(payload.verifications?.[1].allocations).toEqual([
      { cashflowId: '', amount: '40.00000000' },
    ]);
    expect(payload.verifications?.[1].shortfallCashflow?.ourAccount).toBe(
      '收款账户',
    );
    expect(JSON.stringify(payload)).not.toContain('billId');
    expect(input.invoices?.[0].allocations?.[0].billId).toBe('group-ar');
  });
  it('禁止外组费用、票额不一致与核销超额，失败保留原表单输入', () => {
    const input = values();
    if (!input.invoices?.[0].allocations?.[0].fees?.[0])
      throw new Error('缺少测试行');
    input.invoices[0].allocations[0].fees[0].orderFeeId = 'foreign-fee';
    expect(() => billGroupOperationPayload(group, input)).toThrow(
      '不属于当前账单组',
    );
    expect(input.invoices[0].taxInvoiceNo).toBe('TAX-0');
    expect(() =>
      billGroupOperationPayload({ ...group, totalAmount: '99' }, values()),
    ).toThrow('同次核销合计');
  });
  it('未登记发票不能附带核销，部分发票允许核销超过开票额且不改变草稿', () => {
    const input = values();
    input.invoices = [];
    expect(() => billGroupOperationPayload(group, input)).toThrow(
      '账单尚未开票',
    );
    expect(input.verifications).toHaveLength(2);
    const partialInvoice = values().invoices?.[0];
    if (!partialInvoice) throw new Error('缺少发票夹具');
    input.invoices = [partialInvoice];
    expect(billGroupOperationPayload(group, input).verifications).toHaveLength(
      2,
    );
  });
  it('逐条错误与NETTING整体失败位置使用中文记录序号', () => {
    const result = normalizeBillBatchResponse(
      {
        data: { id: 'batch' },
        results: [
          {
            groupKey: 'ar',
            error: {
              reason: 'BALANCE',
              message: '超额',
              items: [
                {
                  step: 'VERIFICATION',
                  itemIndex: 2,
                  reason: 'BALANCE',
                  message: '超额',
                },
              ],
            },
          },
        ],
      },
      ['ar'],
    );
    expect(billBatchItemLabel(result.groups[0].itemErrors?.[0] ?? {})).toBe(
      '第2条核销',
    );
    expect(
      requestMessage(
        Object.assign(new Error(), {
          data: {
            message: '超额',
            metadata: { step: 'VERIFICATION', item_index: '2' },
          },
        }),
        '失败',
      ),
    ).toBe('第2条核销：超额');
    expect(
      requestMessage(
        Object.assign(new Error(), {
          data: {
            message: '超额',
            metadata: {
              step: 'VERIFICATION',
              item_index: '2',
              group_key: 'ar',
            },
          },
        }),
        '失败',
        {
          ar: {
            settlementPartyName: '华东客户',
            direction: 'RECEIVABLE',
            currency: 'USD',
          },
        },
      ),
    ).toBe('华东客户 · 应收 · USD：第2条核销：超额');
  });
});
