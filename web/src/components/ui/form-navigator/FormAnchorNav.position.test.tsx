import { layoutDimensions } from '@root/config/layout';
import { act, fireEvent, render } from '@testing-library/react';
import { afterEach, expect, it, vi } from 'vitest';
import { FormAnchorNav } from './FormAnchorNav';

afterEach(() => {
  vi.unstubAllGlobals();
  vi.restoreAllMocks();
});

it.each([true, false])(
  '大纲折叠为 %s 时随页头高度和滚动同步避让，卸载后停止监听',
  (defaultCollapsed) => {
    let onResize: ResizeObserverCallback = () => {};
    const observe = vi.fn();
    const disconnect = vi.fn();
    vi.stubGlobal(
      'ResizeObserver',
      class {
        constructor(callback: ResizeObserverCallback) {
          onResize = callback;
        }
        observe = observe;
        disconnect = disconnect;
      },
    );

    const shell = document.createElement('header');
    shell.className = 'roncin-page-header-shell';
    document.body.appendChild(shell);
    let bottom = 180;
    const measure = vi
      .spyOn(shell, 'getBoundingClientRect')
      .mockImplementation(() => ({ bottom }) as DOMRect);

    try {
      const { container, unmount } = render(
        <FormAnchorNav
          items={[{ key: 'basic', title: '基础信息' }]}
          defaultCollapsed={defaultCollapsed}
        />,
      );
      const nav = container.firstElementChild;
      const expectBelowHeader = () =>
        expect(nav).toHaveStyle({
          top: `${bottom + layoutDimensions.formScrollGap}px`,
        });

      expectBelowHeader();
      expect(observe).toHaveBeenCalledWith(shell);

      bottom = 220;
      act(() => onResize([], {} as ResizeObserver));
      expectBelowHeader();

      bottom = 140;
      fireEvent.scroll(window);
      expectBelowHeader();

      bottom = 160;
      fireEvent.resize(window);
      expectBelowHeader();

      unmount();
      expect(disconnect).toHaveBeenCalledOnce();
      measure.mockClear();
      fireEvent.scroll(window);
      fireEvent.resize(window);
      expect(measure).not.toHaveBeenCalled();
    } finally {
      shell.remove();
    }
  },
);
