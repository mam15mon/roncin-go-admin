import type { ScrollToErrorOptions, ScrollToErrorResult } from './types';

/**
 * 将字段路径数组转换为用于 DOM 查找的标识串
 */
function normalizeFieldPath(name: (string | number)[]): {
  full: string;
  tail: string;
} {
  const full = name.map(String).join('_');
  const tail = String(name[name.length - 1] ?? '');
  return { full, tail };
}

/**
 * 查找指定表单字段的 DOM 节点（通常为 FormItem 或其内部的输入控件）
 */
export function findFieldDomElement(
  namePath: (string | number)[],
  root: HTMLElement | Document = document,
): HTMLElement | null {
  const { full, tail } = normalizeFieldPath(namePath);

  // 1. 优先按完整路径匹配（保证 Form.List 多品目如 documents_1_houseNo 不会误命中 documents_0_houseNo）
  if (full) {
    const byIdFull = root.querySelector<HTMLElement>(
      `[id$="${full}"], [id="${full}"]`,
    );
    if (byIdFull) {
      return byIdFull.closest('.ant-form-item') || byIdFull;
    }

    const byNameFull = root.querySelector<HTMLElement>(`[name="${full}"]`);
    if (byNameFull) {
      return byNameFull.closest('.ant-form-item') || byNameFull;
    }
  }

  // 2. 仅当单层字段或未命中完整路径时，尝试尾缀匹配
  if (tail) {
    const byIdTail = root.querySelector<HTMLElement>(
      `[id$="_${tail}"], [id="${tail}"]`,
    );
    if (byIdTail) {
      return byIdTail.closest('.ant-form-item') || byIdTail;
    }

    const byNameTail = root.querySelector<HTMLElement>(`[name="${tail}"]`);
    if (byNameTail) {
      return byNameTail.closest('.ant-form-item') || byNameTail;
    }
  }

  // 3. 降级：从所有标红的表单项中查找
  const errorItems = root.querySelectorAll<HTMLElement>(
    '.ant-form-item-has-error',
  );
  if (errorItems.length > 0) {
    return errorItems[0];
  }

  return null;
}

/**
 * 从当前表单项向上查找所属的 SectionCard 业务标识
 */
export function findParentSectionKey(el: HTMLElement): string | null {
  const sectionCard = el.closest<HTMLElement>(
    '[data-section-key], [id^="section-"]',
  );
  if (!sectionCard) return null;

  const keyFromAttr = sectionCard.getAttribute('data-section-key');
  if (keyFromAttr) return keyFromAttr;

  const id = sectionCard.id || '';
  if (id.startsWith('section-')) {
    return id.substring(8);
  }

  return null;
}

/**
 * 提取表单项的中文 Label
 */
export function getFormItemLabel(formItemEl: HTMLElement): string {
  const labelEl = formItemEl.querySelector(
    '.ant-form-item-label label, .ant-form-item-label',
  );
  if (labelEl) {
    const text = labelEl.textContent?.replace(/\*/g, '').trim();
    if (text) return text;
  }
  return '';
}

/**
 * 为表单项触发呼吸红光脉冲动效，吸引操作员视线
 */
export function pulseHighlightElement(el: HTMLElement): void {
  const target = el.closest<HTMLElement>('.ant-form-item') || el;
  target.classList.remove('roncin-form-error-pulse');
  // 强制回流以重置 CSS 动画
  void target.offsetWidth;
  target.classList.add('roncin-form-error-pulse');

  window.setTimeout(() => {
    target.classList.remove('roncin-form-error-pulse');
  }, 2800);
}

/**
 * 将光标自动聚焦至字段输入控件内部
 */
export function focusFieldInput(formItemEl: HTMLElement): void {
  const inputEl = formItemEl.querySelector<HTMLElement>(
    'input:not([type="hidden"]):not([disabled]), textarea:not([disabled]), .ant-select-selection-search-input, [tabindex="0"]',
  );
  if (inputEl && typeof inputEl.focus === 'function') {
    try {
      inputEl.focus({ preventScroll: true });
    } catch {
      // 忽略部分自定义组件可能存在的 focus 异常
    }
  }
}

/**
 * 统计全表单当前各 Section 内的错误数量
 */
export function collectFormSectionErrors(
  root: HTMLElement | Document = document,
): Record<string, number> {
  const result: Record<string, number> = {};
  const errorElements = root.querySelectorAll<HTMLElement>(
    '.ant-form-item-has-error',
  );

  errorElements.forEach((errorEl) => {
    const sectionKey = findParentSectionKey(errorEl);
    if (sectionKey) {
      result[sectionKey] = (result[sectionKey] || 0) + 1;
    }
  });

  return result;
}

/**
 * 核心方法：平滑居中滚动至首个校验错误项，自动展开折叠卡片并高亮聚焦
 */
export function scrollToFirstFormError(
  options: ScrollToErrorOptions = {},
): ScrollToErrorResult {
  const {
    errorFields = [],
    container = document.body,
    headerOffset = 146,
    onExpandSection,
    notify,
  } = options;

  const resolvedContainer =
    container && container !== document.body
      ? container
      : document.querySelector<HTMLElement>('.ant-form') ||
        container ||
        document;
  const root = resolvedContainer;
  const totalErrors =
    errorFields.length > 0
      ? errorFields.length
      : root.querySelectorAll('.ant-form-item-has-error').length;

  if (totalErrors === 0) {
    return {
      success: false,
      totalErrors: 0,
      errorsBySection: {},
    };
  }

  const firstError = errorFields[0];
  let targetEl: HTMLElement | null = null;

  if (firstError?.name) {
    targetEl = findFieldDomElement(firstError.name, root);
  }

  if (!targetEl) {
    targetEl = root.querySelector<HTMLElement>('.ant-form-item-has-error');
  }

  if (!targetEl) {
    return {
      success: false,
      totalErrors,
      errorsBySection: {},
    };
  }

  // 1. 如果该字段所在的父级 Section 处于折叠状态，先通知父组件将其展开
  const parentSectionKey = findParentSectionKey(targetEl);
  if (parentSectionKey && onExpandSection) {
    onExpandSection(parentSectionKey);
  }

  // 2. 提取字段中文名
  const fieldLabel =
    getFormItemLabel(targetEl) ||
    (firstError ? normalizeFieldPath(firstError.name).tail : '');

  // 3. 精准计算平滑滚动：考虑吸顶头部高度并居中展示
  window.setTimeout(() => {
    if (!targetEl) return;
    const rect = targetEl.getBoundingClientRect();
    const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
    // 居中视口并减去 headerOffset 保证不被吸顶遮挡
    const viewportHeight = window.innerHeight;
    const targetY =
      scrollTop +
      rect.top -
      (viewportHeight / 2 - rect.height / 2) -
      headerOffset / 3;

    window.scrollTo({
      top: Math.max(0, targetY),
      behavior: 'smooth',
    });

    // 4. 脉冲呼吸动效与聚焦
    pulseHighlightElement(targetEl);
    focusFieldInput(targetEl);
  }, 60);

  // 5. 统计分节错误
  const errorsBySection = collectFormSectionErrors(root);

  // 6. 弹出友好的操作员提示
  if (notify) {
    const errorDesc = fieldLabel ? `「${fieldLabel}」` : '';
    notify(
      `表单发现 ${totalErrors} 处校验错误，已为您自动定位至首项${errorDesc}`,
    );
  }

  return {
    success: true,
    firstErrorField: firstError,
    fieldLabel,
    totalErrors,
    errorsBySection,
  };
}
