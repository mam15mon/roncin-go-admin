import type * as React from 'react';
import { useCallback, useEffect, useRef, useState } from 'react';

/** 组织画布的平移 / 缩放 / 自适应视口交互状态。 */
export function useOrgChartViewport({
  containerRef,
  contentRef,
  chartDirection,
  inspectorOpen,
  selectedOrg,
}: {
  containerRef: React.RefObject<HTMLDivElement | null>;
  contentRef: React.RefObject<HTMLDivElement | null>;
  chartDirection: 'vertical' | 'horizontal';
  inspectorOpen: boolean;
  selectedOrg?: API.AdminOrganization | null;
}) {
  const [offset, setOffset] = useState({ x: 0, y: 0 });
  const [zoom, setZoom] = useState(1);
  const [isPanning, setIsPanning] = useState(false);

  const offsetRef = useRef(offset);
  useEffect(() => {
    offsetRef.current = offset;
  }, [offset]);

  const isPanningRef = useRef(false);
  const dragRef = useRef({
    startX: 0,
    startY: 0,
    startOffsetX: 0,
    startOffsetY: 0,
    moved: false,
  });

  // Mouse wheel zoom listener (passive: false to prevent document scroll)
  useEffect(() => {
    const container = containerRef.current;
    if (!container) return undefined;

    const handleWheel = (e: WheelEvent) => {
      e.preventDefault();
      const rect = container.getBoundingClientRect();
      const mouseX = e.clientX - rect.left;
      const mouseY = e.clientY - rect.top;

      const zoomFactor = e.deltaY < 0 ? 1.1 : 0.9;
      setZoom((prevZoom) => {
        const nextZoom = Math.min(Math.max(prevZoom * zoomFactor, 0.3), 2.5);
        if (nextZoom === prevZoom) return prevZoom;

        setOffset((prevOffset) => {
          const contentX = (mouseX - prevOffset.x) / prevZoom;
          const contentY = (mouseY - prevOffset.y) / prevZoom;
          return {
            x: mouseX - contentX * nextZoom,
            y: mouseY - contentY * nextZoom,
          };
        });
        return nextZoom;
      });
    };

    container.addEventListener('wheel', handleWheel, { passive: false });
    return () => {
      container.removeEventListener('wheel', handleWheel);
    };
  }, [containerRef]);

  // Canvas Pan handlers
  const handleMouseDown = (e: React.MouseEvent) => {
    if (e.button !== 0 && e.button !== 1) return;
    const target = e.target as HTMLElement;
    if (target.closest('button, [data-interactive="true"]')) {
      return;
    }
    isPanningRef.current = true;
    setIsPanning(true);
    dragRef.current = {
      startX: e.clientX,
      startY: e.clientY,
      startOffsetX: offsetRef.current.x,
      startOffsetY: offsetRef.current.y,
      moved: false,
    };
  };

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!isPanningRef.current) return;
      const dx = e.clientX - dragRef.current.startX;
      const dy = e.clientY - dragRef.current.startY;
      if (Math.abs(dx) > 4 || Math.abs(dy) > 4) {
        dragRef.current.moved = true;
      }
      setOffset({
        x: dragRef.current.startOffsetX + dx,
        y: dragRef.current.startOffsetY + dy,
      });
    };

    const handleMouseUp = () => {
      if (isPanningRef.current) {
        isPanningRef.current = false;
        setIsPanning(false);
      }
    };

    window.addEventListener('mousemove', handleMouseMove);
    window.addEventListener('mouseup', handleMouseUp);

    return () => {
      window.removeEventListener('mousemove', handleMouseMove);
      window.removeEventListener('mouseup', handleMouseUp);
    };
  }, []);

  // Fit View logic with inspector safe area compensation
  const fitView = useCallback(() => {
    if (!containerRef.current || !contentRef.current) return;
    const containerWidth = containerRef.current.clientWidth;
    const containerHeight = containerRef.current.clientHeight;
    const contentWidth =
      contentRef.current.offsetWidth || contentRef.current.scrollWidth;
    const contentHeight =
      contentRef.current.offsetHeight || contentRef.current.scrollHeight;

    if (!containerWidth || !containerHeight || !contentWidth || !contentHeight)
      return;

    const rightReserve = inspectorOpen && selectedOrg ? 390 : 0;
    const padding = 40;
    const availableWidth = Math.max(
      containerWidth - rightReserve - padding * 2,
      100,
    );
    const availableHeight = Math.max(containerHeight - padding * 2, 100);

    const scaleX = availableWidth / contentWidth;
    const scaleY = availableHeight / contentHeight;
    const fitScale = Math.min(Math.max(Math.min(scaleX, scaleY), 0.35), 1.15);

    const newOffsetX =
      (containerWidth - rightReserve - contentWidth * fitScale) / 2;
    const newOffsetY =
      chartDirection === 'vertical'
        ? Math.max(24, (containerHeight - contentHeight * fitScale) / 3)
        : (containerHeight - contentHeight * fitScale) / 2;

    setZoom(fitScale);
    setOffset({ x: Math.round(newOffsetX), y: Math.round(newOffsetY) });
  }, [chartDirection, containerRef, contentRef, inspectorOpen, selectedOrg]);

  // Center a node in the safe visible area (reserving inspector panel space)
  const centerNode = useCallback(
    (id: string) => {
      if (!containerRef.current) return;
      const targetEl = containerRef.current.querySelector(
        `[data-node-id="${id}"]`,
      ) as HTMLElement | null;
      if (!targetEl) return;

      const containerRect = containerRef.current.getBoundingClientRect();
      const nodeRect = targetEl.getBoundingClientRect();
      const containerWidth = containerRect.width;
      const containerHeight = containerRect.height;

      // Reserve space for inspector panel on the right (390px) if open
      const rightReserve = inspectorOpen ? 390 : 0;
      const safeCenterX = (containerWidth - rightReserve) / 2;
      const safeCenterY = containerHeight / 2;

      const nodeCenterX =
        nodeRect.left + nodeRect.width / 2 - containerRect.left;
      const nodeCenterY =
        nodeRect.top + nodeRect.height / 2 - containerRect.top;

      const deltaX = safeCenterX - nodeCenterX;
      const deltaY = safeCenterY - nodeCenterY;

      setOffset((prev) => ({
        x: Math.round(prev.x + deltaX),
        y: Math.round(prev.y + deltaY),
      }));
    },
    [containerRef, inspectorOpen],
  );

  // Zoom helpers for floating controls
  const handleZoomIn = () => {
    if (!containerRef.current) return;
    const nextZoom = Math.min(zoom * 1.2, 2.5);
    const cx = containerRef.current.clientWidth / 2;
    const cy = containerRef.current.clientHeight / 2;
    setOffset({
      x: cx - ((cx - offset.x) / zoom) * nextZoom,
      y: cy - ((cy - offset.y) / zoom) * nextZoom,
    });
    setZoom(nextZoom);
  };

  const handleZoomOut = () => {
    if (!containerRef.current) return;
    const nextZoom = Math.max(zoom * 0.8, 0.3);
    const cx = containerRef.current.clientWidth / 2;
    const cy = containerRef.current.clientHeight / 2;
    setOffset({
      x: cx - ((cx - offset.x) / zoom) * nextZoom,
      y: cy - ((cy - offset.y) / zoom) * nextZoom,
    });
    setZoom(nextZoom);
  };

  const handleResetZoom = () => {
    if (!containerRef.current || !contentRef.current) return;
    const cw = contentRef.current.offsetWidth || contentRef.current.scrollWidth;
    const ch =
      contentRef.current.offsetHeight || contentRef.current.scrollHeight;
    const vw = containerRef.current.clientWidth;
    const vh = containerRef.current.clientHeight;
    setZoom(1);
    setOffset({
      x: Math.max(24, (vw - cw) / 2),
      y: chartDirection === 'vertical' ? 36 : Math.max(24, (vh - ch) / 2),
    });
  };

  return {
    offset,
    zoom,
    isPanning,
    dragRef,
    fitView,
    centerNode,
    handleMouseDown,
    handleZoomIn,
    handleZoomOut,
    handleResetZoom,
  };
}
