import type { Dayjs } from 'dayjs';
import dayjs from 'dayjs';
import { type CsvColumn, csvAmount, serializeCsv } from '@/utils/csv';
import { formatDate } from '@/utils/format';
import {
  calculationBasisText,
  commissionStatusMeta,
  personnelRoleText,
} from './types';

export type CommissionMonthRange = [Dayjs | null, Dayjs | null] | null;

export type CommissionSearchValues = {
  keyword?: string;
  status?: number;
  organizationId?: string;
  commissionMonth?: CommissionMonthRange;
};

export type CommissionQueryFilters = Pick<
  API.SettlementServiceExportCommissionsParams,
  | 'keyword'
  | 'status'
  | 'commissionDateFrom'
  | 'commissionDateTo'
  | 'organizationId'
>;

export function normalizeCommissionFilters(
  values: CommissionSearchValues,
): CommissionQueryFilters {
  const [startMonth, endMonth] = values.commissionMonth ?? [null, null];
  const keyword = values.keyword?.trim();

  return {
    keyword: keyword || undefined,
    status: values.status,
    organizationId: values.organizationId,
    commissionDateFrom: startMonth
      ? startMonth.startOf('month').format('YYYY-MM-DD')
      : undefined,
    commissionDateTo: endMonth
      ? endMonth.endOf('month').format('YYYY-MM-DD')
      : undefined,
  };
}

const csvColumns: CsvColumn<API.CommissionExportItem>[] = [
  {
    header: '所属公司',
    kind: 'text',
    value: (item) => item.organizationName ?? '',
  },
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
    header: '来源单号',
    kind: 'text',
    value: (item) => item.verificationNo || item.nettingNo || '',
  },
  {
    header: '归属日期',
    kind: 'text',
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
    kind: 'number',
    value: (item) => item.ratePercent ?? '',
  },
  {
    header: '本位币币种',
    kind: 'text',
    value: (item) => item.baseCurrency ?? '',
  },
  {
    header: '生成时间',
    kind: 'text',
    value: (item) => (item.createdAt ? formatDate(item.createdAt) : ''),
  },
  {
    header: '原始提成(本位币)',
    kind: 'number',
    value: (item) => csvAmount(item.commissionAmount),
  },
  {
    header: '原始提成(CNY)',
    kind: 'number',
    value: (item) => csvAmount(item.cnyCommissionAmount),
  },
  {
    header: '调整金额(本位币)',
    kind: 'number',
    value: (item) => csvAmount(item.adjustmentAmount),
  },
  {
    header: '调整金额(CNY)',
    kind: 'number',
    value: (item) => csvAmount(item.cnyAdjustmentAmount),
  },
  {
    header: '有效提成(本位币)',
    kind: 'number',
    value: (item) => csvAmount(item.effectiveCommissionAmount),
  },
  {
    header: '有效提成(CNY)',
    kind: 'number',
    value: (item) => csvAmount(item.cnyEffectiveCommissionAmount),
  },
];

export function serializeCommissionCsv(
  items: API.CommissionExportItem[],
): string | null {
  return serializeCsv(items, csvColumns);
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
