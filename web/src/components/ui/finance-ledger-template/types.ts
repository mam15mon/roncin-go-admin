import type {
  ActionType,
  ProColumns,
  ProTableProps,
} from '@ant-design/pro-components';
import type { TableProps } from 'antd';
import type React from 'react';
import type { ReactNode } from 'react';

/** ProTable 注入到 request 参数中的分页与关键字字段（与筛选表单字段合并） */
export interface FinanceLedgerRequestParams {
  current?: number;
  pageSize?: number;
  keyword?: string;
}

/** ProTable 内置搜索表单配置对象（false 表示关闭内置搜索） */
export type FinanceLedgerSearchConfig = Exclude<
  ProTableProps<FinanceLedgerSummaryItem, Record<string, unknown>>['search'],
  false
>;

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

export interface FinanceBatchActionItem<T = FinanceLedgerSummaryItem> {
  key: string;
  label: string;
  onClick: (selectedKeys: React.Key[], selectedRows: T[]) => void;
  disabled?: boolean;
}

export interface FinanceLedgerTemplateProps<
  T extends FinanceLedgerSummaryItem = FinanceLedgerSummaryItem,
  TFilter extends Record<string, unknown> = Record<string, unknown>,
> {
  pageTitle?: ReactNode;
  pageSubTitle?: ReactNode;
  headerExtra?: ReactNode;
  headerTitle?: string;
  columns: ProColumns<T>[];
  rowKey?: string;
  scrollX?: number | string;
  actionRef?: React.MutableRefObject<ActionType | undefined>;

  // 顶部全局工具栏/筛选插槽（如所属公司选择器、标签筛选等，置于指标卡上方）
  topBar?: ReactNode;

  // 顶部宏观统计指标卡配置
  metricCards?: FinanceLedgerMetricCard[];

  // 核心主操作按钮（如“创建账单”）
  primaryActionText?: string;
  primaryActionIcon?: ReactNode;
  onPrimaryAction?: (selectedKeys: React.Key[], selectedRows: T[]) => void;
  primaryActionRequiresSelection?: boolean;

  // 自定义表格行选择配置
  rowSelection?: TableProps<T>['rowSelection'] | false;

  // 批量操作下拉组
  batchActions?: FinanceBatchActionItem<T>[];

  // 导出/导入
  exportFileName?: string;
  onExport?: (selectedRows: T[], allRows: T[]) => void;
  onImport?: () => void;

  // 自定义额外工具栏插槽
  extraToolBarActions?: ReactNode[];

  // ProTable 数据源请求；params 为分页字段与 TFilter 筛选字段的合并结果
  request: (params: TFilter & FinanceLedgerRequestParams) => Promise<{
    data: T[];
    total: number;
    success?: boolean;
    summary?: FinanceLedgerGlobalSummary;
  }>;

  // 是否展示底部双层多币种动态汇总底栏（默认 true）
  showSummaryBoard?: boolean;

  // 表头排序/设置弹窗入口
  onOpenColumnConfig?: () => void;

  // 7 类业务状态行背景高亮颜色配置
  rowColors?: API.FeeLedgerRowColors;
  getRowStatusColorKey?: (
    record: T,
  ) => keyof API.FeeLedgerRowColors | undefined;

  // 整行点击事件（如点击行跳转详情）
  onRowClick?: (record: T, event: React.MouseEvent) => void;

  // ProTable 搜索表单配置覆盖（默认固定 labelWidth: 80 保持对齐）
  search?: FinanceLedgerSearchConfig | false;

  // 自定义嵌入式搜索筛选栏插槽（置于顶部指标统计卡与表格台账之间）
  customSearch?: ReactNode;
}
