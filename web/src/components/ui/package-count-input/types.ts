import type React from 'react';

export interface PackageCountInputProps {
  /** 件数表单字段名（支持顶层字符串或路径数组） */
  countName: string | (string | number)[];
  /** 包装单位表单字段名（支持顶层字符串或路径数组） */
  unitName: string | (string | number)[];
  /** 件数输入框占位符，默认为 '件数' */
  countPlaceholder?: string;
  /** 包装单位下拉框占位符，默认为 '单位' */
  unitPlaceholder?: string;
  /** 是否禁用 */
  disabled?: boolean;
  /** 单位下拉框固定宽度，默认为 130px */
  unitWidth?: number;
  /** 包装单位选项列表，默认使用海运标准 514 种预置单位 */
  unitOptions?: { label: string; value: string }[];
  /** 件数最小值，默认为 0 */
  min?: number;
  /** 根节点样式 */
  style?: React.CSSProperties;
  /** 根节点类名 */
  className?: string;
}
