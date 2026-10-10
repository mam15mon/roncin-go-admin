import { act, fireEvent, render, waitFor } from '@testing-library/react';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useResponsiveTableLayout } from './useResponsiveTableLayout';

describe('共享表格布局配置', () => {
  afterEach(() => vi.restoreAllMocks());

  it('按实际容器取消固定，保留显式纵向滚动和业务列内容', async () => {
    vi.spyOn(HTMLElement.prototype, 'clientWidth', 'get').mockReturnValue(299);
    let layout: ReturnType<typeof useResponsiveTableLayout> | undefined;
    const columns = [
      { width: 220, fixed: 'left' as const },
      { width: 120, fixed: 'right' as const },
    ];
    function Harness() {
      layout = useResponsiveTableLayout({
        columns,
        scroll: { y: 260, scrollToFirstRowOnChange: false },
      });
      return <div ref={layout.containerRef} />;
    }
    render(<Harness />);
    await waitFor(() => expect(layout?.releaseFixed).toBe(true));
    expect(layout?.scroll).toEqual({
      x: 340,
      y: 260,
      scrollToFirstRowOnChange: false,
    });
    expect(layout?.columns?.every((column) => !column.fixed)).toBe(true);
    expect(columns[0].fixed).toBe('left');
    await act(async () => {});
  });

  it('调用方明确的滚动宽度和sticky关闭保持生效', async () => {
    let layout: ReturnType<typeof useResponsiveTableLayout> | undefined;
    function Harness() {
      layout = useResponsiveTableLayout({
        scroll: { x: 1400, y: 400 },
        sticky: false,
      });
      return <div ref={layout.containerRef} />;
    }
    render(<Harness />);
    expect(layout?.scroll).toEqual({ x: 1400, y: 400 });
    expect(layout?.sticky).toBe(false);
    await act(async () => {});
  });

  it('滚动到表格中段后仍按前置业务吸顶页头避让', async () => {
    let tableTop = 220;
    const rect = (top: number, height: number) =>
      new DOMRect(0, top, 800, height);
    vi.spyOn(HTMLElement.prototype, 'getClientRects').mockImplementation(
      function (this: HTMLElement) {
        const item = this.getBoundingClientRect();
        return Object.assign([item], { item: () => item });
      },
    );
    vi.spyOn(HTMLElement.prototype, 'getBoundingClientRect').mockImplementation(
      function (this: HTMLElement) {
        return this.classList.contains('roncin-page-header-shell')
          ? rect(84, 64)
          : rect(tableTop, 500);
      },
    );
    let layout: ReturnType<typeof useResponsiveTableLayout> | undefined;
    function Harness() {
      layout = useResponsiveTableLayout({});
      return (
        <>
          <div
            className="roncin-page-header-shell"
            style={{ position: 'sticky', top: 84 }}
          />
          <div ref={layout.containerRef} />
        </>
      );
    }
    render(<Harness />);
    const offset = () =>
      typeof layout?.sticky === 'object'
        ? layout.sticky.offsetHeader
        : undefined;
    await waitFor(() => expect(offset()).toBe(148));
    tableTop = -220;
    fireEvent.scroll(document);
    await waitFor(() => expect(offset()).toBe(148));
    await act(async () => {});
  });

  it('弹层使用真正纵向溢出的祖先作为官方滚动容器', async () => {
    vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockImplementation(
      function (this: HTMLElement) {
        return this.dataset.scrollOwner ? 1000 : 0;
      },
    );
    vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockImplementation(
      function (this: HTMLElement) {
        return this.dataset.scrollOwner ? 400 : 0;
      },
    );
    let layout: ReturnType<typeof useResponsiveTableLayout> | undefined;
    function Harness() {
      layout = useResponsiveTableLayout({});
      return (
        <div role="dialog">
          <div data-scroll-owner="true" style={{ overflowY: 'auto' }}>
            <div ref={layout.containerRef} />
          </div>
        </div>
      );
    }
    const { container } = render(<Harness />);
    await waitFor(() => {
      const sticky = layout?.sticky;
      expect(typeof sticky).toBe('object');
      if (typeof sticky === 'object') {
        expect(sticky.getContainer?.()).toBe(
          container.querySelector('[data-scroll-owner]'),
        );
        expect(sticky.offsetHeader).toBe(0);
      }
    });
    await act(async () => {});
  });

  it('弹层其他区块展开后首次滚动重新绑定实际滚动容器', async () => {
    let scrollHeight = 200;
    vi.spyOn(HTMLElement.prototype, 'scrollHeight', 'get').mockImplementation(
      function (this: HTMLElement) {
        return this.dataset.scrollOwner ? scrollHeight : 0;
      },
    );
    vi.spyOn(HTMLElement.prototype, 'clientHeight', 'get').mockImplementation(
      function (this: HTMLElement) {
        return this.dataset.scrollOwner ? 400 : 0;
      },
    );
    let layout: ReturnType<typeof useResponsiveTableLayout> | undefined;
    function Harness() {
      layout = useResponsiveTableLayout({});
      return (
        <div role="dialog">
          <div data-scroll-owner="true" style={{ overflowY: 'auto' }}>
            <div ref={layout.containerRef} />
          </div>
        </div>
      );
    }
    const { container } = render(<Harness />);
    const owner = container.querySelector('[data-scroll-owner]') as HTMLElement;
    const currentOwner = () =>
      typeof layout?.sticky === 'object'
        ? layout.sticky.getContainer?.()
        : undefined;
    await waitFor(() => expect(currentOwner()).toBe(window));
    scrollHeight = 1000;
    fireEvent.scroll(owner);
    await waitFor(() => expect(currentOwner()).toBe(owner));
    await act(async () => {});
  });
});
