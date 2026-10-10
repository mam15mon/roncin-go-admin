import { layoutOffsets } from '@root/config/layout';
import type { TableProps } from 'antd';
import { useCallback, useEffect, useRef, useState } from 'react';
import {
  type LayoutColumn,
  releaseFixedColumns,
  shouldReleaseFixedColumns,
  visibleColumnsWidth,
} from './layout';

type Sticky = TableProps<object>['sticky'];
type Scroll = TableProps<object>['scroll'];

/** 页面自身吸顶区会盖住表头，按其 top 和真实高度避让。 */
function pageHeaderOffset(element: HTMLElement): number {
  if (element.closest('[role="dialog"]')) return 0;
  let offset: number = layoutOffsets.topStackHeight;
  const anchors = document.querySelectorAll<HTMLElement>(
    '.roncin-page-header-shell, .roncin-parameter-setting-template [role="tablist"]',
  );
  for (const anchor of anchors) {
    if (element.contains(anchor) || anchor.getClientRects().length === 0)
      continue;
    const style = getComputedStyle(anchor);
    const top = Number.parseFloat(style.top);
    const rect = anchor.getBoundingClientRect();
    if (
      style.position === 'sticky' &&
      Number.isFinite(top) &&
      anchor.compareDocumentPosition(element) & Node.DOCUMENT_POSITION_FOLLOWING
    ) {
      offset = Math.max(offset, top + rect.height);
    }
  }
  return Math.ceil(offset);
}

/** 寻找实际纵向滚动区域，跳过仅用于横向滚动且没有纵向溢出的祖先。 */
function findScrollContainer(element: HTMLElement): HTMLElement | Window {
  let parent = element.parentElement;
  while (parent && parent !== document.body) {
    const style = getComputedStyle(parent);
    if (
      /auto|scroll/.test(style.overflowY) &&
      parent.scrollHeight > parent.clientHeight
    ) {
      return parent;
    }
    parent = parent.parentElement;
  }
  return window;
}

/** 布局状态只来自容器尺寸，不接管表格请求、列偏好或编辑生命周期。 */
export function useResponsiveTableLayout<C extends LayoutColumn>(options: {
  columns?: C[];
  scroll?: Scroll;
  sticky?: Sticky;
  auxiliaryFixedWidth?: number;
  auxiliaryWidth?: number;
}) {
  const {
    columns,
    scroll,
    sticky,
    auxiliaryFixedWidth = 0,
    auxiliaryWidth = 0,
  } = options;
  const containerRef = useRef<HTMLDivElement>(null);
  const [containerWidth, setContainerWidth] = useState(0);
  const [scrollContainer, setScrollContainer] = useState<
    HTMLElement | Window | null
  >(null);
  const [headerOffset, setHeaderOffset] = useState<number>(
    layoutOffsets.topStackHeight,
  );

  useEffect(() => {
    const element = containerRef.current;
    if (!element) return;
    const measure = () => {
      setContainerWidth(element.clientWidth);
      const container = findScrollContainer(element);
      setScrollContainer(container);
      setHeaderOffset(container === window ? pageHeaderOffset(element) : 0);
    };
    measure();
    const observer = new ResizeObserver(measure);
    observer.observe(element);
    let parent = element.parentElement;
    while (parent && parent !== document.body) {
      observer.observe(parent);
      parent = parent.parentElement;
    }
    window.addEventListener('resize', measure);
    // 弹层其他区块展开可能只增加 scrollHeight；捕获其首次滚动再确认容器归属。
    document.addEventListener('scroll', measure, true);
    return () => {
      observer.disconnect();
      window.removeEventListener('resize', measure);
      document.removeEventListener('scroll', measure, true);
    };
  }, []);

  const getContainer = useCallback(
    () => scrollContainer ?? window,
    [scrollContainer],
  );
  const releaseFixed = shouldReleaseFixedColumns(
    columns ?? [],
    containerWidth,
    auxiliaryFixedWidth,
  );
  const columnsWidth = visibleColumnsWidth(columns ?? []);
  const defaultScrollWidth =
    columnsWidth === 'max-content' || columnsWidth === 0
      ? 'max-content'
      : columnsWidth + auxiliaryWidth;

  return {
    containerRef,
    releaseFixed,
    columns: columns && releaseFixed ? releaseFixedColumns(columns) : columns,
    scroll: { ...scroll, x: scroll?.x ?? defaultScrollWidth },
    sticky:
      sticky === false
        ? false
        : {
            offsetHeader: headerOffset,
            offsetScroll: 0,
            getContainer,
            ...(typeof sticky === 'object' ? sticky : {}),
          },
  };
}
