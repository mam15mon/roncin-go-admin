import dayjs from 'dayjs';

const DATE_FIELD_NAMES = new Set([
  'orderDate',
  'etd',
  'eta',
  'siCutoff',
  'docCutoff',
  'customsCutoff',
  'vgmCutoff',
  'createdAt',
  'updatedAt',
]);

const ISO_DATE_PATTERN =
  /^\d{4}-\d{2}-\d{2}(T\d{2}:\d{2}(:\d{2}(\.\d+)?)?(Z|[+-]\d{2}:?\d{2})?)?$/;
const FORM_DRAFT_STORAGE_PREFIX = 'roncin:form-draft';

/**
 * 递归还原草稿中的日期对象（反序列化后字符串还原为 Dayjs 实例）
 */
export function reviveFormDraft<T>(value: T): T {
  if (!value || typeof value !== 'object') {
    return value;
  }
  if (Array.isArray(value)) {
    return value.map((item) => reviveFormDraft(item)) as T;
  }
  const result: Record<string, unknown> = {};
  for (const [k, v] of Object.entries(value)) {
    const isDateField =
      DATE_FIELD_NAMES.has(k) ||
      k.endsWith('Date') ||
      k.endsWith('Cutoff') ||
      k.endsWith('At');
    if (isDateField && typeof v === 'string' && ISO_DATE_PATTERN.test(v)) {
      const d = dayjs(v);
      result[k] = d.isValid() ? d : v;
    } else if (v && typeof v === 'object') {
      result[k] = reviveFormDraft(v);
    } else {
      result[k] = v;
    }
  }
  return result as T;
}

/**
 * 构造当前用户与组织的草稿命名空间，避免切换身份后复用其他上下文的表单数据。
 */
export function getFormDraftScope(
  userId?: string,
  organizationId?: string,
): string | undefined {
  const normalizedUserId = userId?.trim();
  const normalizedOrganizationId = organizationId?.trim();
  if (!normalizedUserId || !normalizedOrganizationId) return undefined;
  return `${encodeURIComponent(normalizedUserId)}:${encodeURIComponent(normalizedOrganizationId)}`;
}

/**
 * 构造特定身份、页签与路径下的草稿键名。
 */
export function getFormDraftKey(
  tabKey?: string,
  pathname?: string,
  draftScope?: string,
): string {
  if (!tabKey || !pathname || !draftScope) {
    return '';
  }

  return `${FORM_DRAFT_STORAGE_PREFIX}:${draftScope}:${tabKey}:${pathname}`;
}

/**
 * 将表单临时输入暂存至 sessionStorage
 */
export function saveFormDraft(draftKey: string, values: unknown): void {
  try {
    if (typeof sessionStorage === 'undefined' || !draftKey || !values) return;
    sessionStorage.setItem(draftKey, JSON.stringify(values));
  } catch {
    // 忽略存储超限或受限环境错误
  }
}

/**
 * 读取并还原暂存的表单草稿
 */
export function getFormDraft<T = Record<string, unknown>>(
  draftKey: string,
): T | null {
  try {
    if (typeof sessionStorage === 'undefined' || !draftKey) return null;
    const raw = sessionStorage.getItem(draftKey);
    if (!raw) return null;
    const parsed = JSON.parse(raw);
    return reviveFormDraft(parsed) as T;
  } catch {
    return null;
  }
}

/**
 * 判断指定草稿是否存在
 */
export function hasFormDraft(draftKey: string): boolean {
  try {
    if (typeof sessionStorage === 'undefined' || !draftKey) return false;
    return sessionStorage.getItem(draftKey) !== null;
  } catch {
    return false;
  }
}

/**
 * 检查指定页签是否存在未提交的表单草稿
 */
export function hasTabDraft(tabKey: string, draftScope?: string): boolean {
  try {
    if (typeof sessionStorage === 'undefined' || !tabKey || !draftScope) {
      return false;
    }
    const prefix = `${FORM_DRAFT_STORAGE_PREFIX}:${draftScope}:${tabKey}:`;
    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i);
      if (key?.startsWith(prefix)) {
        return true;
      }
    }
    return false;
  } catch {
    return false;
  }
}

/**
 * 清除指定路径的草稿
 */
export function clearFormDraft(draftKey: string): void {
  try {
    if (typeof sessionStorage === 'undefined' || !draftKey) return;
    sessionStorage.removeItem(draftKey);
  } catch {
    // 忽略
  }
}

/**
 * 清除指定页签下的所有草稿（在页签确认关闭时调用）
 */
export function clearTabDrafts(tabKey: string, draftScope?: string): void {
  try {
    if (typeof sessionStorage === 'undefined' || !tabKey || !draftScope) return;
    const prefix = `${FORM_DRAFT_STORAGE_PREFIX}:${draftScope}:${tabKey}:`;
    const keysToRemove: string[] = [];
    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i);
      if (key?.startsWith(prefix)) {
        keysToRemove.push(key);
      }
    }
    for (const key of keysToRemove) {
      sessionStorage.removeItem(key);
    }
  } catch {
    // 忽略
  }
}
