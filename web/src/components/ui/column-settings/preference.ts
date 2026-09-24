/**
 * 列偏好的浏览器本地存取与合并逻辑。
 *
 * 存储 key 按「表格标识 + 用户 + 组织」隔离，同一业务视图跨实体复用，
 * 不使用实体 ID，避免逐订单产生配置；只存列 key、显隐与顺序，
 * 不含任何业务数据或身份秘密。
 */
import type {
  ColumnSettingsField,
  ColumnSettingsScope,
  ColumnSettingsValue,
} from './types';

const STORAGE_PREFIX = 'roncin:column-settings:v1';

function storageKey(tableKey: string, scope: ColumnSettingsScope): string {
  return `${STORAGE_PREFIX}:${tableKey}:${
    scope.userId || 'anonymous'
  }:${scope.organizationId || 'default'}`;
}

function isStringArray(value: unknown): value is string[] {
  return (
    Array.isArray(value) &&
    value.every((item) => typeof item === 'string' && item.length > 0)
  );
}

/** 读取本浏览器偏好；缺失、损坏或存储不可用时返回 null（即使用默认列）。 */
export function loadColumnSettingsPreference(
  tableKey: string,
  scope: ColumnSettingsScope,
): ColumnSettingsValue | null {
  try {
    const raw = window.localStorage.getItem(storageKey(tableKey, scope));
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    if (typeof parsed !== 'object' || parsed === null) return null;
    const { order, hidden } = parsed as Record<string, unknown>;
    if (!isStringArray(order) || !isStringArray(hidden)) return null;
    return { order, hidden };
  } catch {
    return null;
  }
}

/** 保存偏好；返回是否持久化成功，失败时调用方需明确提示且不得声称已保存。 */
export function saveColumnSettingsPreference(
  tableKey: string,
  scope: ColumnSettingsScope,
  value: ColumnSettingsValue,
): boolean {
  try {
    window.localStorage.setItem(
      storageKey(tableKey, scope),
      JSON.stringify({ order: value.order, hidden: value.hidden }),
    );
    return true;
  } catch {
    return false;
  }
}

/** 删除本表格偏好（偏好等价于默认时调用，避免残留冗余配置）。 */
export function clearColumnSettingsPreference(
  tableKey: string,
  scope: ColumnSettingsScope,
): boolean {
  try {
    window.localStorage.removeItem(storageKey(tableKey, scope));
    return true;
  } catch {
    return false;
  }
}

/** 默认列配置：按元数据顺序排列，显隐以 hiddenKeys 为准。 */
export function defaultColumnSettingsValue(
  fields: ColumnSettingsField[],
  hiddenKeys: string[] = [],
): ColumnSettingsValue {
  const hidden = new Set(hiddenKeys);
  return {
    order: fields.map((field) => field.key),
    hidden: fields
      .filter((field) => hidden.has(field.key))
      .map((field) => field.key),
  };
}

/**
 * 合并存储偏好与当前字段元数据：
 * - 忽略未知 key 与重复 key；
 * - 偏好未提及的新增列按默认显隐兜底并追加到末尾；
 * - 必显列无论存储内容如何都不允许隐藏；
 * - 至少保留一列可见，全部隐藏时退回默认显隐。
 */
export function resolveColumnSettingsValue(
  fields: ColumnSettingsField[],
  defaultValue: ColumnSettingsValue,
  stored: ColumnSettingsValue | null | undefined,
): ColumnSettingsValue {
  const known = new Set(fields.map((field) => field.key));
  const locked = new Set(
    fields.filter((field) => field.lockVisible).map((field) => field.key),
  );
  const defaultHidden = new Set(defaultValue.hidden);

  const seen = new Set<string>();
  const order: string[] = [];
  for (const key of stored?.order ?? []) {
    if (known.has(key) && !seen.has(key)) {
      order.push(key);
      seen.add(key);
    }
  }
  for (const field of fields) {
    if (!seen.has(field.key)) {
      order.push(field.key);
      seen.add(field.key);
    }
  }

  const mentioned = new Set<string>([
    ...(stored?.order ?? []),
    ...(stored?.hidden ?? []),
  ]);
  const hidden = new Set<string>();
  for (const key of stored?.hidden ?? []) {
    if (known.has(key) && !locked.has(key)) {
      hidden.add(key);
    }
  }
  for (const field of fields) {
    if (
      !mentioned.has(field.key) &&
      defaultHidden.has(field.key) &&
      !locked.has(field.key)
    ) {
      hidden.add(field.key);
    }
  }

  if (hidden.size >= order.length) {
    return {
      order,
      hidden: order.filter((key) => defaultHidden.has(key) && !locked.has(key)),
    };
  }
  return { order, hidden: order.filter((key) => hidden.has(key)) };
}
