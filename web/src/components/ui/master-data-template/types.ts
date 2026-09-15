import type { ReactNode } from 'react';

export interface BaseMasterDataItem {
  id: string;
  code: string;
  name: string;
  nameEn?: string;
  enabled: boolean;
  updatedAt?: string;
  [key: string]: any;
}

export interface MasterDataStatItem {
  label: string;
  value: string | number;
  color?: string;
  prefix?: ReactNode;
}

export interface MasterDataFieldConfig {
  name: string;
  label: string;
  type?: 'text' | 'select' | 'number' | 'textarea' | 'checkboxGroup' | 'radio';
  placeholder?: string;
  required?: boolean;
  rules?: any[];
  options?: { label: string; value: any }[];
  initialValue?: any;
  span?: number;
  disabledOnEdit?: boolean;
  extra?: string;
}

export interface MasterDataFilterOption {
  key: string;
  label: string;
  placeholder?: string;
  options: { label: string; value: any }[];
  defaultValue?: any;
  width?: number;
}

export interface MasterDataListQuery {
  page: number;
  pageSize: number;
  keyword?: string;
  enabled?: boolean;
}

export interface MasterDataTemplateProps<
  T extends BaseMasterDataItem = BaseMasterDataItem,
> {
  // Page Header
  title: string;
  subtitle?: string;
  icon?: ReactNode;
  codeLabel?: string; // 例如 "港口五字码" / "机场三字码" / "航司二字码"

  // Data & State
  items: T[];
  loading?: boolean;
  total?: number;
  activeTotal?: number;
  disabledTotal?: number;
  query?: MasterDataListQuery;
  onQueryChange?: (query: MasterDataListQuery) => void;
  onRefresh?: () => Promise<void> | void;

  // Search & Filter
  searchPlaceholder?: string;
  filterOptions?: MasterDataFilterOption[];

  // Form Fields Config for Create/Edit Modal
  formFields: MasterDataFieldConfig[];

  // Custom Columns to Insert (between Name and Status)
  extraColumns?: Array<{
    title: string;
    dataIndex?: string;
    key: string;
    width?: number;
    render?: (value: any, record: T) => ReactNode;
  }>;

  // Actions Callbacks
  onCreate?: (values: any) => Promise<any>;
  onUpdate?: (id: string, values: any) => Promise<any>;
  onToggleActive?: (record: T) => Promise<void> | void;
  onSync?: () => Promise<void> | void;
  onExport?: () => void;

  // Extra Quick Stats or Custom Full Stats
  customStats?: MasterDataStatItem[];
  extraStats?: MasterDataStatItem[];
  showStats?: boolean;

  // Custom Column Width & Render Hooks
  nameWidth?: number;
  renderCode?: (record: T, defaultDom: ReactNode) => ReactNode;
  renderStatus?: (record: T, defaultDom: ReactNode) => ReactNode;

  // 顶部提示横幅（非空时渲染）：A 型页签非总部提示「由总部统一维护与共享」，
  // B 型页签非总部提示「总部共享基线 + 本地补充行仅本组织可见」。
  notice?: string;

  // 行级写入口门控（返回 false 时该行不渲染编辑与停用/启用按钮）：
  // B 型基线行（organizationId 为空）对非总部组织禁用编辑。
  canEditRecord?: (record: T) => boolean;

  // 是否展示「更新时间」列（默认 true，针对静态标准字典可传 false 隐藏冗余噪音）
  showUpdatedAt?: boolean;

  style?: React.CSSProperties;
  className?: string;
}
