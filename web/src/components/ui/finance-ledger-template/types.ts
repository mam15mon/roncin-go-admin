import type { ActionType, ProColumns } from '@ant-design/pro-components';
import type React from 'react';
import type { ReactNode } from 'react';

export interface FinanceLedgerMetricCard {
  key: string;
  title: string;
  value: number | string;
  precision?: number;
  suffix?: string;
  valueColor?: string;
}

export interface FinanceLedgerSummaryItem {
  id?: string;
  direction?: string | number;
  status?: string | number;
  currency?: string;
  totalAmount?: string | number;
  baseCurrency?: string;
  baseCurrencyAmount?: string | number;
}

export interface FinanceLedgerGlobalSummary {
  activeCount?: number | string;
  receivableBaseAmount?: string | number;
  payableBaseAmount?: string | number;
  profitBaseAmount?: string | number;
  baseCurrency?: string;
}

export interface FinanceBatchActionItem<T = any> {
  key: string;
  label: string;
  onClick: (selectedKeys: React.Key[], selectedRows: T[]) => void;
  disabled?: boolean;
}

export interface FinanceLedgerTemplateProps<
  T extends FinanceLedgerSummaryItem = FinanceLedgerSummaryItem,
> {
  headerTitle?: string;
  columns: ProColumns<T>[];
  rowKey?: string;
  scrollX?: number | string;
  actionRef?: React.MutableRefObject<ActionType | undefined>;

  // 顶部宏观统计指标卡配置
  metricCards?: FinanceLedgerMetricCard[];

  // 核心主操作按钮（如“创建账单”）
  primaryActionText?: string;
  primaryActionIcon?: ReactNode;
  onPrimaryAction?: (selectedKeys: React.Key[], selectedRows: T[]) => void;
  primaryActionRequiresSelection?: boolean;

  // 批量操作下拉组
  batchActions?: FinanceBatchActionItem<T>[];

  // 导出/导入
  exportFileName?: string;
  onExport?: (selectedRows: T[], allRows: T[]) => void;
  onImport?: () => void;

  // 自定义额外工具栏插槽
  extraToolBarActions?: ReactNode[];

  // ProTable 数据源请求
  request: (params: Record<string, any>) => Promise<{
    data: T[];
    total: number;
    success?: boolean;
    summary?: FinanceLedgerGlobalSummary;
  }>;

  // 是否展示底部双层多币种动态汇总底栏（默认 true）
  showSummaryBoard?: boolean;

  // 表头排序/设置弹窗入口
  onOpenColumnConfig?: () => void;

  // 5 类状态行背景高亮颜色配置
  rowColors?: API.FeeLedgerRowColors;
  getRowStatusColorKey?: (
    record: T,
  ) =>
    | 'unbilled'
    | 'unverifiedUninvoiced'
    | 'invoicedUnverified'
    | 'verifiedUninvoiced'
    | 'completed'
    | undefined;
}

