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
const FORM_DRAFT_STORAGE_PREFIX = 'roncin:tab-draft';
const draftSessions = new Map<string, object>();

function getTabDraftPrefix(tabKey: string, scope: string): string {
  return `${FORM_DRAFT_STORAGE_PREFIX}:${scope}:${encodeURIComponent(tabKey)}:`;
}

/** 捕获当前编辑实例，确认关闭后拒绝旧实例的迟到写入。 */
export function captureFormDraftSession(draftKey: string): object | undefined {
  if (!draftKey) return undefined;
  const session = draftSessions.get(draftKey) ?? {};
  draftSessions.set(draftKey, session);
  return session;
}

function isCurrentDraftSession(draftKey: string, session?: object): boolean {
  return !session || draftSessions.get(draftKey) === session;
}

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

/** 元数据保留接口原始格式，日期表单字段仍由正文还原。 */
export function parseFormDraftMetadata<T>(
  value?: string | null,
): T | undefined {
  if (!value) return undefined;
  try {
    return JSON.parse(value) as T;
  } catch {
    return undefined;
  }
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

  return `${getTabDraftPrefix(tabKey, draftScope)}${pathname}`;
}

/**
 * 将表单临时输入暂存至 sessionStorage
 */
export function saveFormDraft(
  draftKey: string,
  values: unknown,
  session?: object,
): boolean {
  if (!draftKey || !values || !isCurrentDraftSession(draftKey, session)) {
    return false;
  }
  try {
    // 保留显式清空，避免恢复时重新带入默认值。
    sessionStorage.setItem(
      draftKey,
      JSON.stringify(values, (_key, value) =>
        value === undefined ? null : value,
      ),
    );
    return true;
  } catch {
    // 旧快照无法代表刚发生的编辑，实时守卫需要继续拦截。
    clearFormDraft(draftKey, session);
    return false;
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
  if (!tabKey || !draftScope) return false;
  return tabDraftKeys(getTabDraftPrefix(tabKey, draftScope)).length > 0;
}

function tabDraftKeys(prefix: string): string[] {
  const keys: string[] = [];
  try {
    for (let i = 0; i < sessionStorage.length; i++) {
      const key = sessionStorage.key(i);
      if (key?.startsWith(prefix)) keys.push(key);
    }
  } catch {
    // 浏览器禁止存储时只使用表单的实时守卫。
  }
  return keys;
}

/** 清除指定路径的草稿。 */
export function clearFormDraft(draftKey: string, session?: object): void {
  if (!draftKey || !isCurrentDraftSession(draftKey, session)) return;
  try {
    sessionStorage.removeItem(draftKey);
  } catch {
    // 浏览器禁止存储。
  }
}

/** 保存已确认但原页面已卸载时，只清理提交时的那份快照。 */
export function prepareFormDraftClear(
  draftKey: string,
  session?: object,
): () => void {
  const capturedSession = session ?? captureFormDraftSession(draftKey);
  let submitted: string | null = null;
  try {
    if (draftKey) submitted = sessionStorage.getItem(draftKey);
  } catch {
    // 浏览器禁止存储。
  }

  return () => {
    if (
      submitted === null ||
      !isCurrentDraftSession(draftKey, capturedSession)
    ) {
      return;
    }
    try {
      if (sessionStorage.getItem(draftKey) === submitted) {
        clearFormDraft(draftKey, capturedSession);
      }
    } catch {
      // 浏览器禁止存储。
    }
  };
}

/** 明确放弃一份编辑快照时，撤销旧实例的后续写入资格。 */
export function discardFormDraft(draftKey: string, session?: object): void {
  if (!draftKey || !isCurrentDraftSession(draftKey, session)) return;
  clearFormDraft(draftKey, session);
  draftSessions.delete(draftKey);
}

function discardDrafts(prefix: string): void {
  for (const key of draftSessions.keys()) {
    if (key.startsWith(prefix)) draftSessions.delete(key);
  }
  for (const key of tabDraftKeys(prefix)) clearFormDraft(key);
}

/** 确认关闭页签后，撤销旧编辑实例并清除其草稿。 */
export function clearTabDrafts(tabKey: string, draftScope?: string): void {
  if (tabKey && draftScope)
    discardDrafts(getTabDraftPrefix(tabKey, draftScope));
}

/** 只清理已关闭工作区所属的草稿与实例。 */
export function clearWorkspaceDrafts(draftScope: string): void {
  discardDrafts(`${FORM_DRAFT_STORAGE_PREFIX}:${draftScope}:`);
}
