import type { ScrollToErrorOptions, ScrollToErrorResult } from './types';

/**
 * antd Form 校验失败态的容器类名。
 * 注意：该类名是 antd 内部实现细节而非公开 API 契约（antd 6.6 由
 * FormItem/ItemHolder 在 validateStatus === 'error' 时输出）。本模块的
 * 错误定位与统计都依赖它，因此统一引用本常量，并由
 * formErrorAntdContract.test.tsx 哨兵测试守护——antd 升级若移除该类名，
 * 测试会显式失败而不是让错误定位静默失效。
 */
export const ANT_FORM_ITEM_ERROR_SELECTOR = '.ant-form-item-has-error';

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
    ANT_FORM_ITEM_ERROR_SELECTOR,
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
    ANT_FORM_ITEM_ERROR_SELECTOR,
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
      : root.querySelectorAll(ANT_FORM_ITEM_ERROR_SELECTOR).length;

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
    targetEl = root.querySelector<HTMLElement>(ANT_FORM_ITEM_ERROR_SELECTOR);
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

/**
 * 实测吸顶叠层高度（全局 Header + TagsView + 吸顶页头壳）。
 * 以页头壳底边为准：页头多高避让多高，操作按钮行不会盖住落点标题；
 * 页面没有吸顶页头壳时回退到 fallback。
 */
export function measureStickyTopOffset(fallback = 146): number {
  const shell = document.querySelector('.roncin-page-header-shell');
  if (shell) {
    const bottom = shell.getBoundingClientRect().bottom;
    if (bottom > 0) return Math.ceil(bottom) + 12;
  }
  return fallback;
}

/**
 * 等待元素布局稳定：展开动画、数据加载等造成的位移逐帧收敛后返回；
 * 超时兜底，避免极端情况下挂起。
 */
export function waitForLayoutStable(
  el: HTMLElement,
  timeoutMs = 800,
): Promise<void> {
  return new Promise((resolve) => {
    let lastTop = el.getBoundingClientRect().top;
    const startedAt = performance.now();
    const tick = () => {
      if (!el.isConnected) {
        resolve();
        return;
      }
      const top = el.getBoundingClientRect().top;
      if (
        Math.abs(top - lastTop) < 0.5 ||
        performance.now() - startedAt >= timeoutMs
      ) {
        resolve();
        return;
      }
      lastTop = top;
      requestAnimationFrame(tick);
    };
    requestAnimationFrame(tick);
  });
}

/**
 * 滚动到分节：按实测吸顶高度落位，保证分节标题不被吸顶按钮栏遮挡。
 */
export function scrollToSectionWithStickyOffset(
  sectionEl: HTMLElement,
  fallbackOffset = 146,
): void {
  const rect = sectionEl.getBoundingClientRect();
  const scrollTop = window.pageYOffset || document.documentElement.scrollTop;
  window.scrollTo({
    top: Math.max(
      0,
      scrollTop + rect.top - measureStickyTopOffset(fallbackOffset),
    ),
    behavior: 'smooth',
  });
}

/**
 * 定位分节内校验错误：等待展开/加载布局稳定后查找错误项。
 * 找到错误项居中滚动并脉冲动效聚焦；无错误项时按吸顶补偿落到分节标题。
 * 调用方负责先行展开折叠分节（如通过 onSelect 展开 collapse keys）。
 */
export async function locateSectionError(
  sectionKey: string,
  fallbackOffset = 146,
): Promise<void> {
  const sectionEl =
    document.getElementById(`section-${sectionKey}`) ||
    document.querySelector<HTMLElement>(`[data-section-key="${sectionKey}"]`);
  if (!sectionEl) return;

  await waitForLayoutStable(sectionEl);

  const errorEl = sectionEl.querySelector<HTMLElement>(
    ANT_FORM_ITEM_ERROR_SELECTOR,
  );
  if (errorEl) {
    errorEl.scrollIntoView({ behavior: 'smooth', block: 'center' });
    pulseHighlightElement(errorEl);
    focusFieldInput(errorEl);
  } else {
    scrollToSectionWithStickyOffset(sectionEl, fallbackOffset);
  }
}
