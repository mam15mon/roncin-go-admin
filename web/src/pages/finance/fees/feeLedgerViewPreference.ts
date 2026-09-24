import type { FinanceLedgerViewConfig } from '@/components/ui/finance-ledger-template';

/**
 * 费用台账视图偏好的浏览器本地存取，与全站统一列设置同一套隔离口径：
 * 存储 key 按「用户 + 当前组织」隔离，同一浏览器内不同账号/组织互不串用。
 * 只存列 key、显隐顺序与配色，不含任何业务数据。
 */

const STORAGE_PREFIX = 'roncin:fee-ledger-view:v1';

export interface FeeLedgerViewScope {
  userId?: string;
  organizationId?: string;
}

function storageKey(scope: FeeLedgerViewScope): string {
  return `${STORAGE_PREFIX}:${scope.userId || 'anonymous'}:${
    scope.organizationId || 'default'
  }`;
}

function isStringArray(value: unknown): value is string[] {
  return (
    Array.isArray(value) &&
    value.every((item) => typeof item === 'string' && item.length > 0)
  );
}

function isViewConfig(value: unknown): value is FinanceLedgerViewConfig {
  if (typeof value !== 'object' || value === null) return false;
  const { columns, rowColors } = value as Record<string, unknown>;
  if (
    typeof columns !== 'object' ||
    columns === null ||
    !isStringArray((columns as { order?: unknown }).order) ||
    !isStringArray((columns as { hidden?: unknown }).hidden)
  ) {
    return false;
  }
  if (typeof rowColors !== 'object' || rowColors === null) return false;
  return Object.values(rowColors as Record<string, unknown>).every(
    (color) => typeof color === 'string' && color.length > 0,
  );
}

/** 读取本浏览器视图偏好；缺失、损坏或存储不可用时返回 null（使用默认配置）。 */
export function loadFeeLedgerViewConfig(
  scope: FeeLedgerViewScope,
): FinanceLedgerViewConfig | null {
  try {
    const raw = window.localStorage.getItem(storageKey(scope));
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    return isViewConfig(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

/** 保存视图偏好；返回是否持久化成功，失败时调用方需明确提示且不得声称已保存。 */
export function saveFeeLedgerViewConfig(
  scope: FeeLedgerViewScope,
  config: FinanceLedgerViewConfig,
): boolean {
  try {
    window.localStorage.setItem(storageKey(scope), JSON.stringify(config));
    return true;
  } catch {
    return false;
  }
}
