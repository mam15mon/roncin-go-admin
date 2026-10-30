import { afterEach, describe, expect, it, vi } from 'vitest';
import { serializeCsv } from '@/utils/csv';
import { feeExportColumns } from './ledgerExport';

afterEach(() => vi.unstubAllEnvs());

describe('fees 台账导出字段', () => {
  it('完整保留业务列、独立币种、金额精度与中文语义，日期按本地时区输出', () => {
    vi.stubEnv('TZ', 'Asia/Shanghai');
    const row: API.FeeLedgerItem = {
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
      direction: 'PAYABLE',
      customerName: '委托甲',
      businessType: 'SE',
      feeCode: 'F001',
      feeName: '海运费',
      economicEffect: 'DECREASE',
      billingUnit: '票',
      quantity: '3.50000000',
      unitPrice: '0.12345678',
      taxRate: '0.60000000',
      orderLocked: true,
      hasPendingFeeSupplement: true,
      hasPendingFeeCorrection: false,
      canBill: false,
      blockedReasons: ['待审核', '已建账'],
      financialProgress: 1,
      billNo: 'B001',
      orderNo: 'O001',
      updatedAt: '2026-10-17T14:44:52Z',
    };
    const values = Object.fromEntries(
      feeExportColumns.map((column) => [column.header, column.value(row)]),
    );
    expect(values).toEqual({
      所属公司: '天津公司',
      结算单位: '客户甲',
      原币币种: 'USD',
      本位币币种: 'CNY',
      创建时间: '2026-10-17 22:44:52',
      备注: '备注,含"引号"\n第二行',
      标签: '重点客户',
      属性: '应付',
      委托单位: '委托甲',
      业务类型: '海运出口',
      费用编码: 'F001',
      费用名称: '海运费',
      费用性质: '更正冲回',
      计费单位: '票',
      数量: '3.5',
      单价: '0.12345678',
      金额: '-15362.50',
      '税率(%)': '0.6',
      税金: '-62.50',
      不含税总价: '-15300.00',
      汇率: '6.7183',
      折本币总价: '-103210.88',
      订单审核: '已审核',
      待审补录: '是',
      待审更正: '否',
      建账限制: '待审核；已建账',
      财务进度: '账单未建立',
      账单编号: 'B001',
      订单编号: 'O001',
      更新时间: '2026-10-17 22:44:52',
    });
    expect(new Set(feeExportColumns.map((column) => column.header)).size).toBe(
      30,
    );
    const csv = serializeCsv(
      [row, { ...row, currency: 'EUR', baseCurrency: 'HKD' }],
      feeExportColumns,
    );
    expect(csv).toContain('EUR');
    expect(csv).toContain('HKD');
    expect(csv).not.toContain('[object Object]');
    expect(csv).not.toContain('T14:44:52Z');
    expect(csv).toContain('"备注,含""引号""\n第二行"');
  });
});
