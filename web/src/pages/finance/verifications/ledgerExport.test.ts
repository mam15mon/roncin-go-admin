import { afterEach, describe, expect, it, vi } from 'vitest';
import { serializeCsv } from '@/utils/csv';
import { verificationExportColumns } from './ledgerExport';

afterEach(() => vi.unstubAllEnvs());

describe('verifications 台账导出字段', () => {
  it('完整保留业务列、独立币种、金额精度与中文语义，日期按本地时区输出', () => {
    vi.stubEnv('TZ', 'Asia/Shanghai');
    const row: API.FinanceVerification = {
      organizationName: '天津公司',
      settlementPartyName: '客户甲',
      currency: 'USD',
      baseCurrency: 'CNY',
      createdAt: '2026-10-17T14:44:52Z',
      note: '备注,含"引号"\n第二行',
      verificationNo: 'V001',
      status: 1,
      direction: 'RECEIVABLE',
      amount: '-3.50000000',
      baseAmount: '-23.51000000',
      billBaseAmount: '-23.41000000',
      cashflowBaseAmount: '-23.51000000',
      exchangeGainLoss: '-0.10000000',
      allocations: [
        {
          cashflowNo: 'CF001',
          billNo: 'B001',
          amount: '-3.50000000',
        },
      ],
      verificationDate: '2026-10-01',
      reversedAt: '2026-10-17T14:44:52Z',
    };
    const values = Object.fromEntries(
      verificationExportColumns.map((column) => [
        column.header,
        column.value(row),
      ]),
    );
    expect(values).toEqual({
      所属公司: '天津公司',
      结算单位: '客户甲',
      原币币种: 'USD',
      本位币币种: 'CNY',
      创建时间: '2026-10-17 22:44:52',
      备注: '备注,含"引号"\n第二行',
      核销编号: 'V001',
      状态: '有效',
      方向: '应收核销',
      核销金额: '-3.50',
      流水本位币合计: '-23.51',
      账单本位币合计: '-23.41',
      资金本位币合计: '-23.51',
      汇兑损益: '-0.10',
      分配数: 1,
      核销日期: '2026-10-01',
      关联明细: 'CF001 → B001: -3.50 USD',
      反核销时间: '2026-10-17 22:44:52',
    });
    expect(
      new Set(verificationExportColumns.map((column) => column.header)).size,
    ).toBe(18);
    const csv = serializeCsv(
      [row, { ...row, currency: 'EUR', baseCurrency: 'HKD' }],
      verificationExportColumns,
    );
    expect(csv).toContain('EUR');
    expect(csv).toContain('HKD');
    expect(csv).not.toContain('[object Object]');
    expect(csv).not.toContain('T14:44:52Z');
    expect(csv).toContain('"备注,含""引号""\n第二行"');
  });
});
