import type { TooltipProps } from 'antd';
import type { CSSProperties, ReactNode } from 'react';

export interface EllipsisTooltipProps {
  /**
   * 需要展示的文本或内容
   */
  children?: ReactNode;

  /**
   * Tooltip 自定义提示内容（默认复用 children 纯文本内容）
   */
  title?: ReactNode;

  /**
   * 容器最大宽度（例如 '100%', 200, '200px'）
   */
  maxWidth?: number | string;

  /**
   * 容器固定宽度
   */
  width?: number | string;

  /**
   * 最大展示行数（默认为 1，即单行省略）
   */
  maxLines?: number;

  /**
   * 是否自动检测溢出（仅在文本实际超出容器被省略时才激活 Tooltip，默认为 true）
   */
  autoDetect?: boolean;

  /**
   * 是否始终开启 Tooltip（即使未溢出也显示，优先级高于 autoDetect）
   */
  alwaysShowTooltip?: boolean;

  /**
   * 是否支持点击一键复制完整文本
   */
  copyable?: boolean;

  /**
   * 复制成功后的提示文案
   */
  copySuccessText?: string;

  /**
   * 传递给 antd Tooltip 的底层配置
   */
  tooltipProps?: Omit<TooltipProps, 'title' | 'children'>;

  /**
   * 自定义类名
   */
  className?: string;

  /**
   * 自定义内联样式
   */
  style?: CSSProperties;
}
