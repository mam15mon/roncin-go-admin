import { afterEach, describe, expect, it, vi } from 'vitest';
import { serializeCsv } from '@/utils/csv';
import { billExportColumns } from './ledgerExport';

afterEach(() => vi.unstubAllEnvs());

describe('bills 台账导出字段', () => {
  it('完整保留业务列、独立币种、金额精度与中文语义，日期按本地时区输出', () => {
    vi.stubEnv('TZ', 'Asia/Shanghai');
    const row: API.FinanceBill = {
      organizationName: '天津公司',
      settlementPartyName: '客户甲',
      currency: 'USD',
      baseCurrency: 'CNY',
      createdAt: '2026-10-17T14:44:52Z',
      note: '备注,含"引号"\n第二行',
      totalAmount: '-15362.50000000',
      taxAmount: '-62.50000000',
      netAmount: '-15300.00000000',
      exchangeRate: '6.71830000',
      baseCurrencyAmount: '-103210.88000000',
      tags: [
        {
          name: '重点客户',
        },
      ],
      direction: 'RECEIVABLE',
      purpose: 'CORRECTION_REVERSAL',
      status: 2,
      billNo: 'B001',
      batchNo: 'BAT001',
      statementTitle: '对账抬头',
      exchangeRateSource: 'MANUAL',
      verifiedAmount: '-62.50000000',
      unverifiedAmount: '-15300.00000000',
      feeCount: 3,
      billDate: '2026-10-01',
      dueDate: '2026-10-31',
      overdueDays: 0,
    };
    const values = Object.fromEntries(
      billExportColumns.map((column) => [column.header, column.value(row)]),
    );
    expect(values).toEqual({
      所属公司: '天津公司',
      结算单位: '客户甲',
      原币币种: 'USD',
      本位币币种: 'CNY',
      创建时间: '2026-10-17 22:44:52',
      备注: '备注,含"引号"\n第二行',
      标签: '重点客户',
      方向: '应收',
      账单用途: '更正冲回',
      状态: '已确认',
      账单编号: 'B001',
      建单批次: 'BAT001',
      对账抬头: '对账抬头',
      账单金额: '-15362.50',
      税额: '-62.50',
      不含税金额: '-15300.00',
      账单汇率: '6.7183',
      汇率来源: '手工',
      折本币金额: '-103210.88',
      已核销: '-62.50',
      未核销: '-15300.00',
      费用数: '3',
      账单日期: '2026-10-01',
      到期日: '2026-10-31',
      逾期天数: '0',
    });
    expect(new Set(billExportColumns.map((column) => column.header)).size).toBe(
      25,
    );
    const csv = serializeCsv(
      [row, { ...row, currency: 'EUR', baseCurrency: 'HKD' }],
      billExportColumns,
    );
    expect(csv).toContain('EUR');
    expect(csv).toContain('HKD');
    expect(csv).not.toContain('[object Object]');
    expect(csv).not.toContain('T14:44:52Z');
    expect(csv).toContain('"备注,含""引号""\n第二行"');
  });
});
