import type { CSSProperties, ReactNode } from 'react';

export interface PageHeaderShellProps {
  /** 页面主标题 */
  title: ReactNode;
  /** 页面副标题或描述 */
  subTitle?: ReactNode;
  /** 返回按钮点击事件 */
  onBack?: () => void;
  /** 返回按钮文字，默认"返回列表" */
  backText?: string;
  /** 面包屑或上级分类导航项 */
  breadcrumbs?: Array<{ label: string; onClick?: () => void; href?: string }>;
  /** 标题右侧的标签徽章 */
  tags?: ReactNode;
  /** 右侧主要操作按钮组 */
  extra?: ReactNode;
  /** 是否固定在顶部 */
  sticky?: boolean;
  /** 额外样式 */
  style?: CSSProperties;
  className?: string;
}

export interface SectionCardProps {
  /** 区块唯一标识 */
  key?: string;
  /** 区块标题 */
  title: ReactNode;
  /** 区块右上角额外操作 */
  extra?: ReactNode;
  /** 内容 */
  children: ReactNode;
  /** 是否支持折叠，默认为 false */
  collapsible?: boolean;
  /** 折叠状态（受控） */
  collapsed?: boolean;
  /** 默认是否折叠 */
  defaultCollapsed?: boolean;
  /** 折叠切换回调 */
  onCollapseChange?: (collapsed: boolean) => void;
  /** 自定义样式 */
  style?: CSSProperties;
  bodyStyle?: CSSProperties;
  className?: string;
}

export interface StickyFooterBarProps {
  /** 左侧附加信息或提示 */
  info?: ReactNode;
  /** 操作按钮组 */
  children: ReactNode;
  /** 自定义样式 */
  style?: CSSProperties;
  className?: string;
}
