import { cleanup, fireEvent, render } from '@testing-library/react';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ResizableHeaderCell } from './ResizableHeaderCell';

describe('公共列宽拖动表头', () => {
  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
    document.body.style.cursor = '';
    document.body.style.userSelect = '';
  });

  it('拖动期间通过辅助参考线实时反馈并在松手时提交最终列宽', () => {
    const frames = new Map<number, FrameRequestCallback>();
    let frameSequence = 0;
    vi.stubGlobal('requestAnimationFrame', (callback: FrameRequestCallback) => {
      frameSequence += 1;
      frames.set(frameSequence, callback);
      return frameSequence;
    });
    vi.stubGlobal('cancelAnimationFrame', (frameId: number) => {
      frames.delete(frameId);
    });

    const onResize = vi.fn();
    const { container } = render(
      <table>
        <thead>
          <tr>
            <ResizableHeaderCell
              columnResize={{
                fallbackWidth: 120,
                onResize,
                onAutoFit: vi.fn(),
              }}
            >
              费用名称
            </ResizableHeaderCell>
          </tr>
        </thead>
      </table>,
    );

    const handle = container.querySelector<HTMLElement>(
      '[data-column-resize-handle]',
    );
    expect(handle).not.toBeNull();
    if (!handle) throw new Error('未渲染列宽拖动手柄');

    fireEvent.mouseDown(handle, { clientX: 100 });
    const guideLine = document.querySelector<HTMLElement>(
      '.roncin-table-resizer-guide',
    );
    expect(guideLine).not.toBeNull();

    fireEvent.mouseMove(window, { clientX: 140 });
    fireEvent.mouseMove(window, { clientX: 180 });

    expect(onResize).not.toHaveBeenCalled();
    expect(frames.size).toBe(1);

    const [frameId, frame] = [...frames.entries()][0];
    frames.delete(frameId);
    frame(0);

    expect(guideLine?.style.transform).toBe('translateX(80px)');
    expect(guideLine?.textContent).toContain('200px');
    expect(onResize).not.toHaveBeenCalled();

    fireEvent.mouseUp(window);
    expect(document.querySelector('.roncin-table-resizer-guide')).toBeNull();
    expect(onResize).toHaveBeenCalledTimes(1);
    expect(onResize).toHaveBeenCalledWith(200);
  });

  it('未改变宽度时不触发整表状态提交', () => {
    vi.stubGlobal(
      'requestAnimationFrame',
      vi.fn(() => 1),
    );
    vi.stubGlobal('cancelAnimationFrame', vi.fn());

    const onResize = vi.fn();
    const { container } = render(
      <table>
        <thead>
          <tr>
            <ResizableHeaderCell
              columnResize={{
                fallbackWidth: 120,
                onResize,
                onAutoFit: vi.fn(),
              }}
            >
              费用名称
            </ResizableHeaderCell>
          </tr>
        </thead>
      </table>,
    );

    const handle = container.querySelector<HTMLElement>(
      '[data-column-resize-handle]',
    );
    if (!handle) throw new Error('未渲染列宽拖动手柄');
    fireEvent.mouseDown(handle, { clientX: 100 });
    fireEvent.mouseUp(window);

    expect(onResize).not.toHaveBeenCalled();
  });

  it.each(['blur', 'Escape', 'unmount'])(
    '取消方式 %s 清理参考线、监听和body样式且不提交',
    (cancelType) => {
      const frames = new Map<number, FrameRequestCallback>();
      vi.stubGlobal(
        'requestAnimationFrame',
        (callback: FrameRequestCallback) => {
          frames.set(1, callback);
          return 1;
        },
      );
      vi.stubGlobal('cancelAnimationFrame', (id: number) => frames.delete(id));
      document.body.style.cursor = 'crosshair';
      document.body.style.userSelect = 'text';
      const onResize = vi.fn();
      const { container, unmount } = render(
        <table>
          <thead>
            <tr>
              <ResizableHeaderCell
                columnResize={{
                  fallbackWidth: 120,
                  onResize,
                  onAutoFit: vi.fn(),
                }}
              >
                费用名称
              </ResizableHeaderCell>
            </tr>
          </thead>
        </table>,
      );
      const handle = container.querySelector<HTMLElement>(
        '[data-column-resize-handle]',
      );
      if (!handle) throw new Error('没有调宽手柄');
      fireEvent.mouseDown(handle, { clientX: 100 });
      fireEvent.mouseMove(window, { clientX: 200 });
      expect(frames.size).toBe(1);
      if (cancelType === 'blur') fireEvent.blur(window);
      else if (cancelType === 'Escape')
        fireEvent.keyDown(window, { key: 'Escape' });
      else unmount();
      expect(document.querySelector('.roncin-table-resizer-guide')).toBeNull();
      expect(document.body.style.cursor).toBe('crosshair');
      expect(document.body.style.userSelect).toBe('text');
      expect(frames.size).toBe(0);
      fireEvent.mouseMove(window, { clientX: 300 });
      fireEvent.mouseUp(window);
      expect(onResize).not.toHaveBeenCalled();
      expect(frames.size).toBe(0);
    },
  );
});
