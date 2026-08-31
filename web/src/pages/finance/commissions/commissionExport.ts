import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import {
  calculationBasisText,
  commissionStatusMeta,
  personnelRoleText,
} from './types';

export type CommissionMonthRange = [Dayjs | null, Dayjs | null] | null;

export type CommissionSearchValues = {
  keyword?: string;
  status?: number;
  commissionMonth?: CommissionMonthRange;
};

export type CommissionQueryFilters = Pick<
  API.SettlementServiceExportCommissionsParams,
  'keyword' | 'status' | 'commissionDateFrom' | 'commissionDateTo'
>;

export function normalizeCommissionFilters(
  values: CommissionSearchValues,
): CommissionQueryFilters {
  const [startMonth, endMonth] = values.commissionMonth ?? [null, null];
  const keyword = values.keyword?.trim();

  return {
    keyword: keyword || undefined,
    status: values.status,
    commissionDateFrom: startMonth
      ? startMonth.startOf('month').format('YYYY-MM-DD')
      : undefined,
    commissionDateTo: endMonth
      ? endMonth.endOf('month').format('YYYY-MM-DD')
      : undefined,
  };
}

type CsvCellKind = 'text' | 'controlled';

type CsvColumn = {
  header: string;
  kind: CsvCellKind;
  value: (item: API.CommissionExportItem) => string;
};

const csvColumns: CsvColumn[] = [
  {
    header: '提成编号',
    kind: 'text',
    value: (item) => item.commissionNo ?? '',
  },
  {
    header: '状态',
    kind: 'text',
    value: (item) =>
      item.status === undefined
        ? ''
        : (commissionStatusMeta[item.status]?.text ?? ''),
  },
  {
    header: '核销编号',
    kind: 'text',
    value: (item) => item.verificationNo ?? '',
  },
  {
    header: '归属日期',
    kind: 'controlled',
    value: (item) => item.commissionDate ?? '',
  },
  {
    header: '提成员工',
    kind: 'text',
    value: (item) => item.employeeName ?? '',
  },
  {
    header: '考核角色',
    kind: 'text',
    value: (item) => personnelRoleText(item.personnelRole),
  },
  { header: '规则名称', kind: 'text', value: (item) => item.ruleName ?? '' },
  {
    header: '计提口径',
    kind: 'text',
    value: (item) => calculationBasisText(item.calculationBasis),
  },
  {
    header: '比例(%)',
    kind: 'controlled',
    value: (item) => item.ratePercent ?? '',
  },
  {
    header: '本位币币种',
    kind: 'text',
    value: (item) => item.baseCurrency ?? '',
  },
  {
    header: '生成时间',
    kind: 'controlled',
    value: (item) => item.createdAt ?? '',
  },
  {
    header: '原始提成(本位币)',
    kind: 'controlled',
    value: (item) => item.commissionAmount ?? '',
  },
  {
    header: '原始提成(CNY)',
    kind: 'controlled',
    value: (item) => item.cnyCommissionAmount ?? '',
  },
  {
    header: '调整金额(本位币)',
    kind: 'controlled',
    value: (item) => item.adjustmentAmount ?? '',
  },
  {
    header: '调整金额(CNY)',
    kind: 'controlled',
    value: (item) => item.cnyAdjustmentAmount ?? '',
  },
  {
    header: '有效提成(本位币)',
    kind: 'controlled',
    value: (item) => item.effectiveCommissionAmount ?? '',
  },
  {
    header: '有效提成(CNY)',
    kind: 'controlled',
    value: (item) => item.cnyEffectiveCommissionAmount ?? '',
  },
];

function protectFormula(value: string): string {
  return /^[\t\r\n ]*[=+\-@]/.test(value) ? `'${value}` : value;
}

function escapeCsvCell(value: string): string {
  return /[",\r\n]/.test(value) ? `"${value.replace(/"/g, '""')}"` : value;
}

export function serializeCommissionCsv(
  items: API.CommissionExportItem[],
): string | null {
  if (items.length === 0) return null;

  const header = csvColumns.map((column) => escapeCsvCell(column.header));
  const rows = items.map((item) =>
    csvColumns.map((column) => {
      const value = column.value(item);
      return escapeCsvCell(
        column.kind === 'text' ? protectFormula(value) : value,
      );
    }),
  );

  return `\uFEFF${[header, ...rows].map((row) => row.join(',')).join('\r\n')}`;
}

export function buildCommissionExportFileName(
  filters: CommissionQueryFilters,
  exportedAt: Dayjs = dayjs(),
): string {
  const fromMonth = filters.commissionDateFrom?.slice(0, 7);
  const toMonth = filters.commissionDateTo?.slice(0, 7);

  if (fromMonth && toMonth) {
    if (fromMonth === toMonth) return `提成导出_${fromMonth}.csv`;
    return `提成导出_${fromMonth}至${toMonth}.csv`;
  }
  if (fromMonth) {
    return `提成导出_${fromMonth}起.csv`;
  }
  if (toMonth) {
    return `提成导出_截至${toMonth}.csv`;
  }
  return `提成导出_${exportedAt.format('YYYY-MM-DD')}.csv`;
}
