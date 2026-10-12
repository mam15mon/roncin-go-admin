import React, { useEffect, useRef, useState } from 'react';

export const MIN_COLUMN_WIDTH = 50;
export const MAX_COLUMN_WIDTH = 600;

export function clampColumnWidth(width: number) {
  return Math.round(
    Math.max(MIN_COLUMN_WIDTH, Math.min(MAX_COLUMN_WIDTH, width)),
  );
}

export interface ColumnResizeConfig {
  fallbackWidth: number;
  onResize: (width: number) => void;
  onAutoFit: (measuredWidth: number) => void;
  onMeasure?: (width: number, cell: HTMLElement) => void;
}

export interface ResizableHeaderCellProps
  extends React.ThHTMLAttributes<HTMLTableCellElement> {
  columnResize?: ColumnResizeConfig;
  component?: React.ElementType;
}

/** 只在松手时提交列宽；原表头组件、事件与固定定位样式完整保留。 */
export function ResizableHeaderCell({
  columnResize,
  component: Cell = 'th',
  children,
  style,
  ...restProps
}: ResizableHeaderCellProps) {
  const [isHovered, setIsHovered] = useState(false);
  const [isDragging, setIsDragging] = useState(false);
  const dragCleanupRef = useRef<(() => void) | undefined>(undefined);

  useEffect(() => () => dragCleanupRef.current?.(), []);

  if (!columnResize) {
    return (
      <Cell style={style} {...restProps}>
        {children}
      </Cell>
    );
  }

  const measureCell = (handle: HTMLElement) => {
    const cell = handle.closest<HTMLElement>('th, td') ?? handle.parentElement;
    return cell ? { cell, rect: cell.getBoundingClientRect() } : undefined;
  };

  const handleMouseDown = (event: React.MouseEvent<HTMLElement>) => {
    event.preventDefault();
    event.stopPropagation();
    if (event.button !== 0) return;
    dragCleanupRef.current?.();
    const measurement = measureCell(event.currentTarget);
    if (!measurement) return;
    const { cell, rect: cellRect } = measurement;
    const tableWrapper = cell.closest('.ant-table-wrapper') ?? document.body;
    const tableRect = tableWrapper.getBoundingClientRect();
    // 短表会铺满容器，百分比列也必须以真实渲染宽度作为起点。
    const startWidth = cellRect.width || columnResize.fallbackWidth;
    columnResize.onMeasure?.(startWidth, cell);
    const startX = event.clientX;
    let targetWidth = startWidth;
    setIsDragging(true);

    const previousCursor = document.body.style.cursor;
    const previousUserSelect = document.body.style.userSelect;
    document.body.style.cursor = 'col-resize';
    document.body.style.userSelect = 'none';
    const guideLine = document.createElement('div');
    guideLine.className = 'roncin-table-resizer-guide';
    guideLine.style.cssText = `
      position: fixed; top: ${tableRect.top}px; left: ${cellRect.right}px;
      width: 2px; height: ${tableRect.height}px; background: #1677ff;
      box-shadow: 0 0 6px rgba(22, 119, 255, 0.6); z-index: 99999;
      pointer-events: none; will-change: transform; transform: translateX(0px);
    `;
    const tooltip = document.createElement('div');
    tooltip.style.cssText = `
      position: absolute; top: 4px; left: 6px; background: rgba(0, 0, 0, 0.78);
      color: #fff; font-size: 11px; font-family: monospace; font-weight: 500;
      padding: 2px 6px; border-radius: 3px; box-shadow: 0 2px 6px rgba(0,0,0,0.2);
      white-space: nowrap; pointer-events: none;
    `;
    tooltip.textContent = `宽度: ${Math.round(startWidth)}px`;
    guideLine.appendChild(tooltip);
    document.body.appendChild(guideLine);
    let animationFrameId: number | undefined;

    const handleMouseMove = (moveEvent: MouseEvent) => {
      targetWidth = clampColumnWidth(startWidth + moveEvent.clientX - startX);
      if (animationFrameId !== undefined)
        window.cancelAnimationFrame(animationFrameId);
      animationFrameId = window.requestAnimationFrame(() => {
        animationFrameId = undefined;
        guideLine.style.transform = `translateX(${targetWidth - startWidth}px)`;
        tooltip.textContent = `宽度: ${targetWidth}px`;
      });
    };
    const cleanup = () => {
      if (animationFrameId !== undefined)
        window.cancelAnimationFrame(animationFrameId);
      guideLine.remove();
      document.body.style.cursor = previousCursor;
      document.body.style.userSelect = previousUserSelect;
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
      window.removeEventListener('blur', cancel);
      window.removeEventListener('keydown', handleKeyDown);
      dragCleanupRef.current = undefined;
    };
    const cancel = () => {
      cleanup();
      setIsDragging(false);
    };
    const handleKeyDown = (keyEvent: KeyboardEvent) => {
      if (keyEvent.key === 'Escape') cancel();
    };
    const handleMouseUp = () => {
      cleanup();
      setIsDragging(false);
      if (targetWidth !== startWidth) columnResize.onResize(targetWidth);
    };
    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);
    window.addEventListener('blur', cancel);
    window.addEventListener('keydown', handleKeyDown);
    dragCleanupRef.current = cleanup;
  };

  // Antd 6 常规表头的 relative 与冻结表头的 sticky 均由 CSS 提供，不用内联 position 覆盖。
  return (
    <Cell style={style} {...restProps}>
      {children}
      <div
        style={{
          position: 'absolute',
          right: 0,
          top: 0,
          bottom: 0,
          width: 7,
          cursor: 'col-resize',
          zIndex: 2,
          backgroundColor: isDragging
            ? '#1677ff'
            : isHovered
              ? 'rgba(22, 119, 255, 0.45)'
              : 'transparent',
          transition: 'background-color 0.15s',
        }}
        onMouseEnter={() => setIsHovered(true)}
        onMouseLeave={() => setIsHovered(false)}
        onMouseDown={handleMouseDown}
        onClick={(event) => event.stopPropagation()}
        onDoubleClick={(event) => {
          event.preventDefault();
          event.stopPropagation();
          const measurement = measureCell(event.currentTarget);
          if (measurement)
            columnResize.onMeasure?.(
              measurement.rect.width || columnResize.fallbackWidth,
              measurement.cell,
            );
          columnResize.onAutoFit(
            measurement?.rect.width || columnResize.fallbackWidth,
          );
        }}
        data-column-resize-handle
        title="按住鼠标左右拖拽调节列宽；双击缝隙自动展开/还原"
      />
    </Cell>
  );
}
