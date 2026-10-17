import { afterEach, describe, expect, it, vi } from 'vitest';
import { serializeCsv } from '@/utils/csv';
import { invoiceExportColumns } from './ledgerExport';

afterEach(() => vi.unstubAllEnvs());

describe('invoices 台账导出字段', () => {
  it('完整保留业务列、独立币种、金额精度与中文语义，日期按本地时区输出', () => {
    vi.stubEnv('TZ', 'Asia/Shanghai');
    const row: API.FinanceInvoice = {
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
      recordNo: 'INV001',
      direction: 'PAYABLE',
      status: 2,
      invoiceType: 'SPECIAL',
      taxInvoiceNo: '000123',
      invoiceDate: '2026-10-01',
      billCount: 2,
      billLinks: [
        {
          billNo: 'B001',
        },
        {
          billNo: 'B002',
        },
      ],
      invoiceTitle: '开票抬头',
      redInvoiceNo: 'RED001',
      redInvoiceDate: '2026-10-02',
    };
    const values = Object.fromEntries(
      invoiceExportColumns.map((column) => [column.header, column.value(row)]),
    );
    expect(values).toEqual({
      所属公司: '天津公司',
      结算单位: '客户甲',
      原币币种: 'USD',
      本位币币种: 'CNY',
      创建时间: '2026-10-17 22:44:52',
      备注: '备注,含"引号"\n第二行',
      记录编号: 'INV001',
      方向: '进项',
      状态: '已收票',
      发票类型: '专用发票',
      金额: '-15362.50',
      税额: '-62.50',
      不含税金额: '-15300.00',
      开票汇率: '6.7183',
      折本币金额: '-103210.88',
      税务发票号: '000123',
      开票日期: '2026-10-01',
      关联账单数: '2',
      关联账单: 'B001；B002',
      开票抬头: '开票抬头',
      红字发票号: 'RED001',
      红冲日期: '2026-10-02',
    });
    expect(
      new Set(invoiceExportColumns.map((column) => column.header)).size,
    ).toBe(22);
    const csv = serializeCsv(
      [row, { ...row, currency: 'EUR', baseCurrency: 'HKD' }],
      invoiceExportColumns,
    );
    expect(csv).toContain('EUR');
    expect(csv).toContain('HKD');
    expect(csv).not.toContain('[object Object]');
    expect(csv).not.toContain('T14:44:52Z');
    expect(csv).toContain('"备注,含""引号""\n第二行"');
  });
  it('销项与进项使用各自的已开具或已收票文案', () => {
    const column = invoiceExportColumns.find((item) => item.header === '状态');
    expect(column?.value({ status: 2, direction: 'RECEIVABLE' })).toBe(
      '已开具',
    );
    expect(column?.value({ status: 3, direction: 'PAYABLE' })).toBe('已作废');
  });
});
