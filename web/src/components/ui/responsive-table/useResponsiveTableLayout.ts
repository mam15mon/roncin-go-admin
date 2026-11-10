import { layoutOffsets } from '@root/config/layout';
import type { TableProps } from 'antd';
import {
  useCallback,
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
} from 'react';
import {
  type LayoutColumn,
  releaseFixedColumns,
  shouldReleaseFixedColumns,
  visibleColumnsWidth,
} from './layout';

type Sticky = TableProps<object>['sticky'];
type Scroll = TableProps<object>['scroll'];
type LayoutState = {
  releaseFixed: boolean;
  scrollContainer: HTMLElement | Window | null;
  headerOffset: number;
};

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
  // 宽度动画会逐帧触发观察器；只发布固定列策略、滚动容器或吸顶偏移的变化。
  const [layout, setLayout] = useState<LayoutState>(() => ({
    releaseFixed: shouldReleaseFixedColumns(
      columns ?? [],
      0,
      auxiliaryFixedWidth,
    ),
    scrollContainer: null,
    headerOffset: layoutOffsets.topStackHeight,
  }));
  const measuredLayoutRef = useRef(layout);
  const fixedLayoutRef = useRef({ columns, auxiliaryFixedWidth });
  fixedLayoutRef.current = { columns, auxiliaryFixedWidth };
  const measureRef = useRef<() => void>(undefined);

  useEffect(() => {
    const element = containerRef.current;
    if (!element) return;
    const measure = () => {
      const current = fixedLayoutRef.current;
      const container = findScrollContainer(element);
      const next: LayoutState = {
        releaseFixed: shouldReleaseFixedColumns(
          current.columns ?? [],
          element.clientWidth,
          current.auxiliaryFixedWidth,
        ),
        scrollContainer: container,
        headerOffset: container === window ? pageHeaderOffset(element) : 0,
      };
      const previous = measuredLayoutRef.current;
      if (
        previous.releaseFixed !== next.releaseFixed ||
        previous.scrollContainer !== next.scrollContainer ||
        previous.headerOffset !== next.headerOffset
      ) {
        measuredLayoutRef.current = next;
        setLayout(next);
      }
    };
    measureRef.current = measure;
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
      measureRef.current = undefined;
      observer.disconnect();
      window.removeEventListener('resize', measure);
      document.removeEventListener('scroll', measure, true);
    };
  }, []);

  // 列偏好与选择列变化也会影响固定策略，不能只等待下一次容器缩放。
  useLayoutEffect(() => {
    measureRef.current?.();
  }, [columns, auxiliaryFixedWidth]);

  const { releaseFixed, scrollContainer, headerOffset } = layout;
  const getContainer = useCallback(
    () => scrollContainer ?? window,
    [scrollContainer],
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
