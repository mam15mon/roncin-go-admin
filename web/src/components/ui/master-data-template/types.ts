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

export interface MasterDataTemplateProps<T extends BaseMasterDataItem = BaseMasterDataItem> {
  // Page Header
  title: string;
  subtitle: string;
  icon?: ReactNode;
  codeLabel?: string; // 例如 "港口五字码" / "机场三字码" / "航司二字码"

  // Data & State
  items: T[];
  loading?: boolean;
  total?: number;
  onRefresh?: () => void;

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

  // Extra Quick Stats (e.g. { label: '国家覆盖', value: '186 个' })
  extraStats?: Array<{ label: string; value: string | number; color?: string }>;
}
