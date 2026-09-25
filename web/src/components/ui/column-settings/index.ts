/**
 * 全站统一列设置：齿轮入口（含锚定浮层与高级设置二级弹窗）、
 * 列数组适配钩子与浏览器偏好存取。页面只消费本目录公开能力，
 * 不再各自实现设置 UI。
 */

export type { ColumnSettingsEntryProps } from './ColumnSettingsEntry';
export { ColumnSettingsEntry } from './ColumnSettingsEntry';
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
export type {
  UseColumnSettingsOptions,
  UseColumnSettingsResult,
} from './useColumnSettings';
export { useColumnSettings } from './useColumnSettings';
