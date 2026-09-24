/**
 * 全站统一列设置：设置弹窗（标准版/增强版）、统一入口按钮与
 * 列数组适配钩子。页面只消费本目录公开能力，不再各自实现设置 UI。
 */
import ColumnSettingsModal from './ColumnSettingsModal';

export { ColumnSettingsModal };
export { ColumnSettingsEntry } from './ColumnSettingsEntry';
export type { ColumnSettingsEntryProps } from './ColumnSettingsEntry';
export type { ColumnSettingsModalProps } from './ColumnSettingsModal';
export {
  clearColumnSettingsPreference,
  defaultColumnSettingsValue,
  loadColumnSettingsPreference,
  resolveColumnSettingsValue,
  saveColumnSettingsPreference,
} from './preference';
export type {
  ColumnSettingsField,
  ColumnSettingsScope,
  ColumnSettingsValue,
} from './types';
export { useColumnSettings } from './useColumnSettings';
export type {
  UseColumnSettingsOptions,
  UseColumnSettingsResult,
} from './useColumnSettings';
