/**
 * 筛选设置的浏览器本地存取与合并逻辑。
 *
 * key 按「页面 + 用户 + 公司」隔离，值只保存字段名；身份未就绪时
 * 不读不写，避免落到 anonymous/default 共享键。
 */
import type {
  FilterSettingsField,
  FilterSettingsPreference,
  FilterSettingsScope,
} from './types';

const STORAGE_PREFIX = 'roncin:filter-settings:v1';

export function isFilterSettingsScopeReady(
  scope: FilterSettingsScope,
): scope is Required<FilterSettingsScope> {
  return Boolean(scope.userId && scope.organizationId);
}

function storageKey(pageKey: string, scope: Required<FilterSettingsScope>) {
  return `${STORAGE_PREFIX}:${pageKey}:${scope.userId}:${scope.organizationId}`;
}

function isStringArray(value: unknown): value is string[] {
  return (
    Array.isArray(value) &&
    value.every((item) => typeof item === 'string' && item.length > 0)
  );
}

/** 读取偏好；身份未就绪、缺失、损坏或存储不可用时返回 null（使用默认可见集合）。 */
export function loadFilterSettingsPreference(
  pageKey: string,
  scope: FilterSettingsScope,
): FilterSettingsPreference | null {
  if (!isFilterSettingsScopeReady(scope)) return null;
  try {
    const raw = window.localStorage.getItem(storageKey(pageKey, scope));
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== 'object' || parsed === null) return null;
    const { visible, hidden } = parsed as Record<string, unknown>;
    if (!isStringArray(visible) || !isStringArray(hidden)) return null;
    return { visible, hidden };
  } catch {
    return null;
  }
}

/** 保存偏好；返回是否持久化成功。身份未就绪时不写入并返回 false。 */
export function saveFilterSettingsPreference(
  pageKey: string,
  scope: FilterSettingsScope,
  value: FilterSettingsPreference,
): boolean {
  if (!isFilterSettingsScopeReady(scope)) return false;
  try {
    window.localStorage.setItem(
      storageKey(pageKey, scope),
      JSON.stringify({ visible: value.visible, hidden: value.hidden }),
    );
    return true;
  } catch {
    return false;
  }
}

/** 删除偏好（恢复默认）。 */
export function clearFilterSettingsPreference(
  pageKey: string,
  scope: FilterSettingsScope,
): boolean {
  if (!isFilterSettingsScopeReady(scope)) return false;
  try {
    window.localStorage.removeItem(storageKey(pageKey, scope));
    return true;
  } catch {
    return false;
  }
}

/**
 * 合并存储偏好与当前字段声明，按页面字段顺序返回可见字段名：
 * 必选字段恒可见；偏好明确隐藏/保留的按偏好；偏好未提及的新字段按
 * 页面声明的默认可见性补入；未知字段名忽略。
 */
export function resolveFilterVisibleNames(
  fields: readonly FilterSettingsField[],
  stored: FilterSettingsPreference | null | undefined,
): string[] {
  const hidden = new Set(stored?.hidden ?? []);
  const visible = new Set(stored?.visible ?? []);
  return fields
    .filter((field) => {
      if (field.required) return true;
      if (hidden.has(field.name)) return false;
      if (visible.has(field.name)) return true;
      return field.defaultVisible;
    })
    .map((field) => field.name);
}

/** 剔除已隐藏字段的值，用于页签恢复；未在 fields 中声明的键原样保留。 */
export function omitHiddenFilterValues<T extends object>(
  values: T,
  fields: readonly FilterSettingsField[],
  visibleNames: readonly string[],
): T {
  const visible = new Set(visibleNames);
  const hidden = new Set(
    fields.filter((field) => !visible.has(field.name)).map((f) => f.name),
  );
  return Object.fromEntries(
    Object.entries(values).filter(([key]) => !hidden.has(key)),
  ) as T;
}
