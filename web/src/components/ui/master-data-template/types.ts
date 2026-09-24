import type { ActionType } from '@ant-design/pro-components';
import type { FormRule } from 'antd';
import type { ReactNode } from 'react';

export interface BaseMasterDataItem {
  id: string;
  code: string;
  name: string;
  nameEn?: string;
  enabled: boolean;
  updatedAt?: string;
  [key: string]: unknown;
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
  rules?: FormRule[];
  // 保留 any：同一份字段配置会分别喂给 ProFormSelect / ProFormRadio.Group /
  // ProFormCheckbox.Group，radio/checkbox 场景存在 boolean 取值（如「全货机」），
  // 而 antd Select 的 option value 类型不含 boolean，收紧会破坏其中一条边界。
  // biome-ignore lint/suspicious/noExplicitAny: antd Select option value 不含 boolean，radio/checkbox 场景需要（见上方注释）
  options?: { label: string; value: any }[];
  initialValue?: unknown;
  span?: number;
  disabledOnEdit?: boolean;
  extra?: string;
}

export interface MasterDataFilterOption {
  key: string;
  label: string;
  placeholder?: string;
  options: { label: string; value: string | number }[];
  defaultValue?: unknown;
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
  TFormValues = Record<string, unknown>,
> {
  // Page Header
  title: string;
  subtitle?: string;
  icon?: ReactNode;
  codeLabel?: string; // 例如 "港口五字码" / "机场三字码" / "航司二字码"

  // Data & State
  items?: T[];
  request?: (params: {
    pageSize?: number;
    current?: number;
    keyword?: string;
    enabled?: boolean;
    [key: string]: unknown;
  }) => Promise<{ data?: T[]; success?: boolean; total?: number }>;
  actionRef?: React.MutableRefObject<ActionType | undefined>;
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
    // 保留 any：消费方普遍以具体窄类型注解首参（如 `(icao: string)`、
    // `(level: number)`），在 strictFunctionTypes 逆变约束下改为 unknown /
    // ReactNode 等宽类型都会使这些消费方编译失败。
    // biome-ignore lint/suspicious/noExplicitAny: strictFunctionTypes 逆变边界，见上方注释
    render?: (value: any, record: T) => ReactNode;
  }>;

  // Actions Callbacks
  onCreate?: (values: TFormValues) => Promise<unknown>;
  onUpdate?: (id: string, values: TFormValues) => Promise<unknown>;
  onToggleActive?: (record: T) => Promise<void> | void;
  onSync?: () => Promise<void> | void;
  onExport?: () => void;

  /** 统一列设置表格标识（业务视图级）；缺省由页面标题派生。 */
  columnSettingsKey?: string;

  // Extra Quick Stats or Custom Full Stats
  customStats?: MasterDataStatItem[];
  extraStats?: MasterDataStatItem[];
  showStats?: boolean;

  // Custom Column Width & Render Hooks
  nameWidth?: number;
  renderCode?: (record: T, defaultDom: ReactNode) => ReactNode;
  renderStatus?: (record: T, defaultDom: ReactNode) => ReactNode;

  // 顶部提示横幅（非空时渲染）：A 型页签非系统管理提示「由系统管理员统一维护与共享」，
  // B 型页签非系统管理提示「系统管理共享基线 + 本地补充行仅本组织可见」。
  notice?: string;

  // 行级写入口门控（返回 false 时该行不渲染编辑与停用/启用按钮）：
  // B 型基线行（organizationId 为空）对非系统管理组织禁用编辑。
  canEditRecord?: (record: T) => boolean;

  // 是否展示「更新时间」列（默认 true，针对静态标准字典可传 false 隐藏冗余噪音）
  showUpdatedAt?: boolean;

  style?: React.CSSProperties;
  className?: string;
}
