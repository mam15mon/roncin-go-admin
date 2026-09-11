import type { CSSProperties, ReactNode } from 'react';

export interface FormAnchorItem {
  /** 唯一标识，与 SectionCard 的 key/id 对应 */
  key: string;
  /** 楼层标题显示 */
  title: string;
  /** 图标或额外前缀 */
  icon?: ReactNode;
}

export interface FormAnchorNavProps {
  /** 楼层分节列表 */
  items: FormAnchorItem[];
  /** 各分节当前的校验错误数映射，如 { basicInfo: 2, transport: 1 } */
  sectionErrors?: Record<string, number>;
  /** 当前选中的分节 key（受控） */
  activeKey?: string;
  /** 点击楼层分节的回调 */
  onSelect?: (key: string) => void;
  /** 点击带有错误的分节时的回调，用于精确定位至该分节首个错误 */
  onErrorClick?: (key: string) => void;
  /** 容器自定义样式 */
  style?: CSSProperties;
  className?: string;
  /** 吸顶偏移量，用于点击跳转时避开固定页头，默认 84 */
  targetOffset?: number;
  /** 是否默认折叠为迷你图标浮标，默认 false */
  defaultCollapsed?: boolean;
}

export interface ScrollToErrorOptions {
  /** 表单校验失败时返回的 errorFields 列表 */
  errorFields?: Array<{ name: (string | number)[]; errors: string[] }>;
  /** 目标查找容器，默认 document */
  container?: HTMLElement | null;
  /** 吸顶导航高度偏移量（像素），默认 84 */
  headerOffset?: number;
  /** 当错误字段在折叠卡片内时的展开回调 */
  onExpandSection?: (sectionKey: string) => void;
  /** 消息提示回调，若提供则在定位至首个错误时调用 */
  notify?: (message: string) => void;
}

export interface ScrollToErrorResult {
  /** 是否成功找到并滚动到首个错误项 */
  success: boolean;
  /** 首个错误字段信息 */
  firstErrorField?: { name: (string | number)[]; errors: string[] };
  /** 字段的中文显示名称 */
  fieldLabel?: string;
  /** 错误总数 */
  totalErrors: number;
  /** 各分节错误数统计映射 */
  errorsBySection: Record<string, number>;
}
