import type { FormInstance } from 'antd';
import type { ReactNode } from 'react';
import type { SearchableSelectProps } from '../searchable-select';

export type SearchFilterFieldType =
  | 'input'
  | 'select'
  | 'searchable-select'
  | 'date'
  | 'date-range'
  | 'digit'
  | 'custom';

export interface SearchFilterFieldItem {
  /** 字段名称 */
  name: string;
  /** 字段标签 */
  label?: ReactNode;
  /** 字段类型，默认 'input' */
  type?: SearchFilterFieldType;
  /** 占位提示文案 */
  placeholder?: string | [string, string];
  /** 下拉候选项列表（当 type 为 select 或 searchable-select 时） */
  options?: SearchableSelectProps['options'];
  /** 异步请求获取候选项（当 type 为 select 或 searchable-select 时） */
  request?: (params: { keyWords?: string }) => Promise<{ label: string; value: any; [k: string]: any }[]>;
  /** 栅格跨度（默认 6，即 24 栅格下一行 4 列） */
  span?: number;
  /** 自定义渲染组件（当 type 为 'custom' 时） */
  render?: (form: FormInstance) => ReactNode;
  /** 初始值 */
  initialValue?: any;
  /** 是否允许清除，默认 true */
  allowClear?: boolean;
  /** 附加组件属性 */
  fieldProps?: Record<string, any>;
}

export interface QuickFilterOption {
  /** 字段名 */
  name: string;
  /** 下拉占位文案 */
  placeholder?: string;
  /** 下拉候选项列表 */
  options: { label: string; value: any }[];
  /** 下拉宽度，默认 140 */
  width?: number | string;
  /** 是否支持搜索，默认 true */
  showSearch?: boolean;
  /** 初始值 */
  initialValue?: any;
}

export interface SearchFilterTemplateProps<TValues = any> {
  /** 模式：'bar' 快捷单行搜索栏 | 'grid' 配置化网格表单 | 'custom' 自由插槽，默认 'grid' */
  layout?: 'bar' | 'grid' | 'custom';
  /** 表单排布方式：'horizontal' 水平行内紧凑 | 'vertical' 垂直上下 | 'inline' 行内，默认 'horizontal' */
  formLayout?: 'horizontal' | 'vertical' | 'inline';
  /** 标签固定宽度（当 formLayout='horizontal' 时生效），默认 80 */
  labelWidth?: number | string;
  /** 默认栅格跨度（默认 4，即 24 栅格下一行 6 列高密度排布） */
  colSpan?: number;
  /** 是否可折叠（当 layout='grid' 时），默认 true */
  collapsible?: boolean;
  /** 默认是否折叠，默认 true */
  defaultCollapsed?: boolean;
  /** 折叠状态下默认展示的字段数量，默认 5 */
  defaultVisibleCount?: number;
  /** 表单字段配置项列表（用于 'grid' 模式） */
  items?: SearchFilterFieldItem[];
  /** 快捷搜索栏模式下的关键字占位符（用于 'bar' 模式） */
  keywordPlaceholder?: string;
  /** 快捷搜索栏模式下的关键字字段名，默认 'keyword' */
  keywordName?: string;
  /** 快捷搜索栏模式下的下拉筛选组（用于 'bar' 模式） */
  quickFilters?: QuickFilterOption[];
  /** 提交搜索回调 */
  onSearch?: (values: TValues) => void;
  /** 重置搜索回调 */
  onReset?: () => void;
  /** 搜索加载中状态 */
  loading?: boolean;
  /** 容器外层样式 */
  style?: React.CSSProperties;
  /** 容器类名 */
  className?: string;
  /** 右侧快捷操作插槽（如新建、刷新、导出按钮组） */
  extraRight?: ReactNode;
  /** 自定义表单插槽渲染函数（用于 'custom' 模式） */
  children?: ReactNode | ((context: { form: FormInstance; collapsed: boolean; toggleCollapse: () => void }) => ReactNode);
  /** 外部传入的 Form 实例 */
  form?: FormInstance;
}
