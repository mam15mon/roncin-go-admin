import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import { serializeCsv } from '@/utils/csv';
import {
  agingExportColumns,
  makeReportQuery,
  profitExportColumns,
  reportPage,
  validateAgingResponse,
  validateAgingRow,
  validateCashflowRow,
  validateProfitResponse,
  validateProfitRow,
} from './reportModel';

const row: API.ProfitReportRow = {
  id: 'row-1',
  orderId: 'order-1',
  orderNo: '=ORDER',
  baseCurrency: 'CNY',
  receivableAmount: '10600.00000000',
  payableAmount: '6360.00000000',
  profitAmount: '4240.00000000',
  netReceivableAmount: '10000.00000000',
  netPayableAmount: '6000.00000000',
  netProfitAmount: '4000.00000000',
};
const response: API.ListProfitReportResponse = {
  success: true,
  generatedAt: '2026-11-14T00:00:00Z',
  total: '1',
  data: [row],
};

describe('报表wire、筛选与导出映射', () => {
  it('ProtoJSON省略零计数/空数组时是真实空报表，显式零与非零均可读取', () => {
    expect(
      reportPage(
        { success: true, generatedAt: response.generatedAt },
        validateProfitRow,
      ),
    ).toEqual({ data: [], total: 0, success: true });
    expect(
      reportPage({ ...response, total: '0', data: [] }, validateProfitRow)
        .total,
    ).toBe(0);
    expect(
      reportPage(response, validateProfitRow).data[0].netProfitAmount,
    ).toBe('4000.00000000');
    validateProfitRow({
      ...row,
      profitAmount: '0',
      netProfitAmount: '0.00000000',
      businessDate: undefined,
    });
    expect(() =>
      reportPage(
        { ...response, data: [{ ...row, orderId: undefined }] },
        validateProfitRow,
      ),
    ).toThrow('订单标识');
  });
  it('成功信封缺失、false或缺真实金额字段不能被当作成功零值', () => {
    for (const success of [undefined, false])
      expect(() =>
        reportPage({ ...response, success }, validateProfitRow),
      ).toThrow();
    expect(() =>
      reportPage(
        { ...response, data: [{ ...row, profitAmount: undefined }] },
        validateProfitRow,
      ),
    ).toThrow('profitAmount');
    expect(() => reportPage(undefined, validateProfitRow)).toThrow();
    expect(() =>
      validateCashflowRow({
        id: 'c',
        direction: 'RECEIVABLE',
        currency: 'CNY',
        baseCurrency: 'CNY',
        amount: '0',
      }),
    ).toThrow('本币金额');
  });
  it('完整合计缺失明确失败，不能从本页计算替代', () => {
    expect(() => validateProfitResponse(response)).toThrow('完整筛选合计');
    expect(() =>
      validateAgingResponse({
        success: true,
        total: '1',
        asOfDate: '2026-11-14',
      }),
    ).toThrow('原币账龄合计');
    validateProfitResponse({ success: true, total: '0' });
  });
  it('optional未设到期日/逾期天数保留存在性，不强行填0', () => {
    const aging: API.AgingReportRow = {
      id: 'b',
      currency: 'CNY',
      baseCurrency: 'CNY',
      totalAmount: '100',
      settledAmount: '0',
      balance: '100',
      baseBalance: '100',
      bucket: 'UNDATED',
    };
    validateAgingRow(aging);
    expect(aging.dueDate).toBeUndefined();
    expect(aging.overdueDays).toBeUndefined();
    const csv = serializeCsv([aging], agingExportColumns);
    expect(csv).toContain('未设置到期日');
    expect(csv).toContain('100.00');
  });
  it('双/单边日期与公司归属规范化，账龄不发送历史截止日期', () => {
    expect(
      makeReportQuery(
        'profit',
        {
          dateRange: [dayjs('2026-02-01'), null],
          keyword: '  客户  ',
          currency: 'USD',
        },
        'company-a',
      ),
    ).toEqual({
      organizationId: 'company-a',
      keyword: '客户',
      dateFrom: '2026-02-01',
      dateTo: undefined,
    });
    expect(
      makeReportQuery(
        'cashflow',
        { dateRange: [null, dayjs('2026-02-28')], currency: ' usd ' },
        'company-a',
      ),
    ).toEqual({
      organizationId: 'company-a',
      currency: 'USD',
      dateFrom: undefined,
      dateTo: '2026-02-28',
    });
    expect(
      makeReportQuery(
        'aging',
        {
          dateRange: [dayjs('2026-01-01'), dayjs('2026-12-31')],
          bucket: 'UNDATED',
        },
        'company-a',
      ),
    ).toEqual({ organizationId: 'company-a', bucket: 'UNDATED' });
  });
  it('导出金额两位无千分位，业务文本做公式防护、保留负利润', () => {
    const csv = serializeCsv(
      [{ ...row, profitAmount: '-15362.50000000' }],
      profitExportColumns,
    );
    expect(csv?.startsWith('\uFEFF')).toBe(true);
    expect(csv).toContain("'=ORDER");
    expect(csv).toContain('-15362.50');
    expect(csv).not.toContain('15,362.50');
    expect(csv).toContain('\r\n');
  });
});
