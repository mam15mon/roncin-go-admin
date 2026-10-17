import { afterEach, describe, expect, it, vi } from 'vitest';
import { serializeCsv } from '@/utils/csv';
import { cashflowExportColumns } from './ledgerExport';

afterEach(() => vi.unstubAllEnvs());

describe('cashflows 台账导出字段', () => {
  it('完整保留业务列、独立币种、金额精度与中文语义，日期按本地时区输出', () => {
    vi.stubEnv('TZ', 'Asia/Shanghai');
    const row: API.FinanceCashflow = {
      organizationName: '天津公司',
      settlementPartyName: '客户甲',
      currency: 'USD',
      baseCurrency: 'CNY',
      createdAt: '2026-10-17T14:44:52Z',
      note: '备注,含"引号"\n第二行',
      direction: 'PAYABLE',
      status: 2,
      flowNo: 'CF001',
      amount: '-15362.50000000',
      exchangeRate: '6.71830000',
      baseAmount: '-103210.88000000',
      verifiedAmount: '-62.50000000',
      unverifiedAmount: '-15300.00000000',
      transactionDate: '2026-10-01',
      ourAccount: '000123',
      counterpartyAccount: '000456',
      paymentMethod: 'BANK_TRANSFER',
      bankReferenceNo: 'BANK001',
    };
    const values = Object.fromEntries(
      cashflowExportColumns.map((column) => [column.header, column.value(row)]),
    );
    expect(values).toEqual({
      所属公司: '天津公司',
      结算单位: '客户甲',
      原币币种: 'USD',
      本位币币种: 'CNY',
      创建时间: '2026-10-17 22:44:52',
      备注: '备注,含"引号"\n第二行',
      方向: '付款',
      状态: '已确认',
      流水编号: 'CF001',
      金额: '-15362.50',
      结算汇率: '6.7183',
      折本币: '-103210.88',
      已核销: '-62.50',
      未核销: '-15300.00',
      交易日期: '2026-10-01',
      我方账户: '000123',
      对方账户: '000456',
      结算方式: '银行转账',
      银行流水号: 'BANK001',
    });
    expect(
      new Set(cashflowExportColumns.map((column) => column.header)).size,
    ).toBe(19);
    const csv = serializeCsv(
      [row, { ...row, currency: 'EUR', baseCurrency: 'HKD' }],
      cashflowExportColumns,
    );
    expect(csv).toContain('EUR');
    expect(csv).toContain('HKD');
    expect(csv).not.toContain('[object Object]');
    expect(csv).not.toContain('T14:44:52Z');
    expect(csv).toContain('"备注,含""引号""\n第二行"');
  });

  it.each([
    ['BANK_TRANSFER', '银行转账'],
    ['CHECK', '支票'],
    ['CASH', '现金'],
    ['THIRD_PARTY', '第三方支付'],
  ])('真实支付方式代码 %s 导出为 %s', (paymentMethod, label) => {
    const csv = serializeCsv([{ paymentMethod }], cashflowExportColumns);
    expect(csv).toContain(label);
    expect(csv).not.toContain(paymentMethod);
  });
});
