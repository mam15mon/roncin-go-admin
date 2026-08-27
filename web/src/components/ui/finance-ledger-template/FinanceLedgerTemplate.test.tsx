import { fireEvent, render } from '@testing-library/react';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ResizableHeaderCell } from './FinanceLedgerTemplate';

describe('ResizableHeaderCell', () => {
  afterEach(() => {
    vi.unstubAllGlobals();
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
            <ResizableHeaderCell width={120} onResize={onResize}>
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
            <ResizableHeaderCell width={120} onResize={onResize}>
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
});
