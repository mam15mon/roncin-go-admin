import type { RowColorsConfig } from '@/components/ui/finance-ledger-template';

/**
 * 费用台账行配色偏好的浏览器本地存取，与全站统一列设置同一套隔离口径：
 * 存储 key 按「用户 + 当前组织」隔离，同一浏览器内不同账号/组织互不串用。
 * 列显隐与顺序由统一列设置（column-settings）按表格标识独立持久化，
 * 本模块只负责行配色，不含任何业务数据。
 */

const STORAGE_PREFIX = 'roncin:fee-ledger-row-colors:v1';

export interface FeeLedgerRowColorScope {
  userId?: string;
  organizationId?: string;
}

function storageKey(scope: FeeLedgerRowColorScope): string {
  return `${STORAGE_PREFIX}:${scope.userId || 'anonymous'}:${
    scope.organizationId || 'default'
  }`;
}

function isRowColors(value: unknown): value is RowColorsConfig {
  if (typeof value !== 'object' || value === null) return false;
  return Object.values(value as Record<string, unknown>).every(
    (color) => typeof color === 'string' && color.length > 0,
  );
}

/** 读取本浏览器行配色偏好；缺失、损坏或存储不可用时返回 null（使用默认配色）。 */
export function loadFeeLedgerRowColors(
  scope: FeeLedgerRowColorScope,
): RowColorsConfig | null {
  try {
    const raw = window.localStorage.getItem(storageKey(scope));
    if (!raw) return null;
    const parsed: unknown = JSON.parse(raw);
    return isRowColors(parsed) ? parsed : null;
  } catch {
    return null;
  }
}

/** 保存行配色偏好；返回是否持久化成功，失败时调用方需明确提示且不得声称已保存。 */
export function saveFeeLedgerRowColors(
  scope: FeeLedgerRowColorScope,
  colors: RowColorsConfig,
): boolean {
  try {
    window.localStorage.setItem(storageKey(scope), JSON.stringify(colors));
    return true;
  } catch {
    return false;
  }
}
