import { afterEach, describe, expect, it, vi } from 'vitest';
import { serializeCsv } from '@/utils/csv';
import { nettingExportColumns } from './ledgerExport';

afterEach(() => vi.unstubAllEnvs());

describe('nettings 台账导出字段', () => {
  it('完整保留业务列、独立币种、金额精度与中文语义，日期按本地时区输出', () => {
    vi.stubEnv('TZ', 'Asia/Shanghai');
    const row: API.FinanceNetting = {
      organizationName: '天津公司',
      settlementPartyName: '客户甲',
      currency: 'USD',
      baseCurrency: 'CNY',
      createdAt: '2026-10-17T14:44:52Z',
      note: '备注,含"引号"\n第二行',
      nettingNo: 'N001',
      status: 4,
      amount: '-3.50000000',
      baseCurrencyAmount: '-23.41000000',
      payableBaseAmount: '-23.51000000',
      exchangeGainLoss: '-0.10000000',
      allocations: [
        {
          billNo: 'B001',
        },
        {
          billNo: 'B002',
        },
      ],
      batchNo: 'BAT001',
      confirmedAt: '2026-10-17T14:44:52Z',
      cancelledAt: '2026-10-17T14:44:52Z',
      reversedAt: '2026-10-17T14:44:52Z',
    };
    const values = Object.fromEntries(
      nettingExportColumns.map((column) => [column.header, column.value(row)]),
    );
    expect(values).toEqual({
      所属公司: '天津公司',
      结算单位: '客户甲',
      原币币种: 'USD',
      本位币币种: 'CNY',
      创建时间: '2026-10-17 22:44:52',
      备注: '备注,含"引号"\n第二行',
      对冲单号: 'N001',
      状态: '已反转',
      抵销金额: '-3.50',
      应收本币抵销额: '-23.41',
      应付本币抵销额: '-23.51',
      对冲汇差: '-0.10',
      分摊数: 2,
      关联批次: 'BAT001',
      关联账单: 'B001；B002',
      确认时间: '2026-10-17 22:44:52',
      取消时间: '2026-10-17 22:44:52',
      反转时间: '2026-10-17 22:44:52',
    });
    expect(
      new Set(nettingExportColumns.map((column) => column.header)).size,
    ).toBe(18);
    const csv = serializeCsv(
      [row, { ...row, currency: 'EUR', baseCurrency: 'HKD' }],
      nettingExportColumns,
    );
    expect(csv).toContain('EUR');
    expect(csv).toContain('HKD');
    expect(csv).not.toContain('[object Object]');
    expect(csv).not.toContain('T14:44:52Z');
    expect(csv).toContain('"备注,含""引号""\n第二行"');
  });
});
