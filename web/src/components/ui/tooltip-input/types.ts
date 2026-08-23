import type { InputProps } from 'antd';
import type { TooltipPlacement } from 'antd/es/tooltip';
import type { ReactNode } from 'react';

export interface TooltipInputProps extends InputProps {
  /**
   * 是否开启悬停气泡提示（当输入框有值时展示，默认 true）
   */
  showTooltip?: boolean;

  /**
   * 自定义 Tooltip 提示内容，不传时默认展示当前 input 的 value
   */
  tooltipTitle?: ReactNode;

  /**
   * Tooltip 弹出位置，默认为 'top'
   */
  tooltipPlacement?: TooltipPlacement;

  /**
   * 是否仅在内容超过输入框可视范围时才显示 Tooltip（默认 false，输入内容 hover 即展示）
   */
  autoDetectOverflow?: boolean;
}
