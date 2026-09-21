import type React from 'react';

export interface QuarterRingProps extends React.ComponentProps<'span'> {
  /**
   * 尺寸（支持像素数值或 CSS 单位字符串，如 16, 24, '1.5em'；未指定时自适应父级 1em）
   */
  size?: number | string;
  /**
   * 旋转一圈耗时，默认 '1s'
   */
  duration?: number | string;
  /**
   * 线条粗细（支持数值或单位字符串，如 2.5 或 '2.5px'，默认 2.5px）
   */
  strokeWidth?: number | string;
}
