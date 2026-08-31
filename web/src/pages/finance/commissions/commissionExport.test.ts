import dayjs from 'dayjs';
import { describe, expect, it } from 'vitest';
import { FinanceCommissionStatus } from '@/enums.generated';
import {
  buildCommissionExportFileName,
  normalizeCommissionFilters,
  serializeCommissionCsv,
} from './commissionExport';

describe('提成月份筛选', () => {
  it('将同月范围转换为包含边界的日期', () => {
    expect(
      normalizeCommissionFilters({
        keyword: '  COM-001  ',
        status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
        commissionMonth: [dayjs('2026-07-01'), dayjs('2026-07-01')],
      }),
    ).toEqual({
      keyword: 'COM-001',
      status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
      commissionDateFrom: '2026-07-01',
      commissionDateTo: '2026-07-31',
    });
  });

  it.each([
    {
      name: '闰年二月',
      range: [dayjs('2028-02-01'), dayjs('2028-02-01')],
      from: '2028-02-01',
      to: '2028-02-29',
    },
    {
      name: '跨年范围',
      range: [dayjs('2026-12-01'), dayjs('2027-01-01')],
      from: '2026-12-01',
      to: '2027-01-31',
    },
    {
      name: '只有开始月份',
      range: [dayjs('2026-03-01'), null],
      from: '2026-03-01',
      to: undefined,
    },
    {
      name: '只有结束月份',
      range: [null, dayjs('2026-04-01')],
      from: undefined,
      to: '2026-04-30',
    },
  ])('支持$name', ({ range, from, to }) => {
    const filters = normalizeCommissionFilters({
      commissionMonth: range as [dayjs.Dayjs | null, dayjs.Dayjs | null],
    });
    expect(filters.commissionDateFrom).toBe(from);
    expect(filters.commissionDateTo).toBe(to);
  });
});

describe('提成 CSV', () => {
  it('空结果不生成 CSV', () => {
    expect(serializeCommissionCsv([])).toBeNull();
  });

  it('输出 BOM、CRLF、固定表头并转义特殊字符', () => {
    const content = serializeCommissionCsv([
      {
        commissionNo: '=危险编号',
        status: FinanceCommissionStatus.FINANCE_COMMISSION_STATUS_CONFIRMED,
        verificationNo: '+危险核销',
        commissionDate: '2026-08-31',
        employeeName: '-危险员工',
        personnelRole: 'SALES',
        ruleName: '规则,"甲"\r\n第二行',
        calculationBasis: 'REALIZED_PROFIT',
        ratePercent: '-2.5000',
        baseCurrency: '@危险币种',
        createdAt: '2026-08-31T09:30:00Z',
        commissionAmount: '-10.00000000',
        cnyCommissionAmount: '-20.00000000',
        adjustmentAmount: '1.00000000',
        cnyAdjustmentAmount: '2.00000000',
        effectiveCommissionAmount: '-9.00000000',
        cnyEffectiveCommissionAmount: '-18.00000000',
      },
    ]);

    expect(content).not.toBeNull();
    expect(content?.startsWith('\uFEFF提成编号,状态,核销编号,归属日期')).toBe(
      true,
    );
    expect(content).toContain('\r\n');
    expect(content).toContain("'=危险编号");
    expect(content).toContain("'+危险核销");
    expect(content).toContain("'-危险员工");
    expect(content).toContain("'@危险币种");
    expect(content).toContain('"规则,""甲""\r\n第二行"');
    expect(content).toContain(',-2.5000,');
    expect(content).toContain(',-10.00000000,-20.00000000,');
  });

  it('保护带前导空白的业务文本公式', () => {
    const content = serializeCommissionCsv([{ commissionNo: '  =SUM(A1:A2)' }]);
    expect(content).toContain("'  =SUM(A1:A2)");
  });
});

describe('提成导出文件名', () => {
  it('使用单月、范围和单边月份', () => {
    expect(
      buildCommissionExportFileName({
        commissionDateFrom: '2026-07-01',
        commissionDateTo: '2026-07-31',
      }),
    ).toBe('提成导出_2026-07.csv');
    expect(
      buildCommissionExportFileName({
        commissionDateFrom: '2026-07-01',
        commissionDateTo: '2026-08-31',
      }),
    ).toBe('提成导出_2026-07至2026-08.csv');
    expect(
      buildCommissionExportFileName({ commissionDateFrom: '2026-07-01' }),
    ).toBe('提成导出_2026-07起.csv');
    expect(
      buildCommissionExportFileName({ commissionDateTo: '2026-08-31' }),
    ).toBe('提成导出_截至2026-08.csv');
  });

  it('没有月份时使用导出日期', () => {
    expect(buildCommissionExportFileName({}, dayjs('2026-08-31'))).toBe(
      '提成导出_2026-08-31.csv',
    );
  });
});
