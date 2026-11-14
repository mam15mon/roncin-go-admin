/**
 * 统一筛选设置：偏好存取、显隐钩子与齿轮入口。字段声明由页面传入，
 * 本目录不导入 pages/features。
 */

export type { FilterSettingsEntryProps } from './FilterSettingsEntry';
export { FilterSettingsEntry } from './FilterSettingsEntry';
export {
  clearFilterSettingsPreference,
  isFilterSettingsScopeReady,
  loadFilterSettingsPreference,
  omitHiddenFilterValues,
  resolveFilterVisibleNames,
  saveFilterSettingsPreference,
} from './preference';
export type {
  FilterSettingsField,
  FilterSettingsPreference,
  FilterSettingsScope,
} from './types';
export type {
  UseFilterSettingsOptions,
  UseFilterSettingsResult,
} from './useFilterSettings';
export { useFilterSettings } from './useFilterSettings';
