import { act, render, waitFor } from '@testing-library/react';
import { createContext, StrictMode, useContext, useRef } from 'react';
import { createMemoryRouter, RouterProvider, useParams } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { bindRouter } from '@/router/history';
import type { AbnormalCasePanelRef } from './abnormal-case-panel';
import { useOrderAbnormalEntry } from './use-order-abnormal-entry';

type EntryState = {
  order?: API.Order;
  loading: boolean;
  error: boolean;
  canRead: boolean;
};
const EntryContext = createContext<EntryState>({
  loading: false,
  error: false,
  canRead: true,
});
const openPanel = vi.fn();

function EntryHarness() {
  const { id, kind } = useParams();
  const state = useContext(EntryContext);
  const panelRef = useRef<AbnormalCasePanelRef>({ open: openPanel });
  useOrderAbnormalEntry({ ...state, orderId: id, kind, panelRef });
  return <div>订单详情</div>;
}

function renderEntry(path: string, initial: Partial<EntryState> = {}) {
  const router = createMemoryRouter(
    [{ path: '/orders/:kind/:id', element: <EntryHarness /> }],
    { initialEntries: [path] },
  );
  bindRouter(router);
  const state: EntryState = {
    order: { id: 'order-1', orderNo: '订单001' },
    loading: false,
    error: false,
    canRead: true,
    ...initial,
  };
  const tree = (value: EntryState) => (
    <StrictMode>
      <EntryContext.Provider value={value}>
        <RouterProvider router={router} />
      </EntryContext.Provider>
    </StrictMode>
  );
  const result = render(tree(state));
  return {
    router,
    update: (next: Partial<EntryState>) => {
      Object.assign(state, next);
      result.rerender(tree({ ...state }));
    },
  };
}

describe('订单异常直达参数消费', () => {
  beforeEach(() => {
    openPanel.mockClear();
  });

  it.each(['sea-export', 'air-export'])(
    '%s 首次打开在 StrictMode 下仅执行一次，清理参数且保留其它查询和锚点',
    async (kind) => {
      const { router, update } = renderEntry(
        `/orders/${kind}/order-1?panel=abnormal&source=workbench#summary`,
      );
      await waitFor(() => {
        expect(openPanel).toHaveBeenCalledTimes(1);
        expect(router.state.location.search).toBe('?source=workbench');
      });
      expect(router.state.location.hash).toBe('#summary');
      expect(router.state.historyAction).toBe('REPLACE');
      update({ order: { id: 'order-1', version: '2' } });
      expect(openPanel).toHaveBeenCalledTimes(1);
    },
  );

  it('已有详情再次收到直达参数会重新打开，同一目标维持相同页签键', async () => {
    const { router } = renderEntry('/orders/sea-export/order-1');
    expect(openPanel).not.toHaveBeenCalled();
    await act(async () => {
      await router.navigate('/orders/sea-export/order-1?panel=abnormal');
    });
    await waitFor(() => expect(openPanel).toHaveBeenCalledTimes(1));
    await act(async () => {
      await router.navigate('/orders/sea-export/order-1?panel=abnormal');
    });
    await waitFor(() => expect(openPanel).toHaveBeenCalledTimes(2));
    expect(router.state.location.pathname).toBe('/orders/sea-export/order-1');
    expect(router.state.location.search).toBe('');
  });

  it('加载中及旧订单数据都不消费，当前目标加载完成后才打开', async () => {
    const { router, update } = renderEntry(
      '/orders/air-export/order-1?panel=abnormal',
      { loading: true, order: { id: 'old-order' } },
    );
    expect(openPanel).not.toHaveBeenCalled();
    update({ loading: false });
    expect(openPanel).not.toHaveBeenCalled();
    expect(router.state.location.search).toBe('?panel=abnormal');
    update({ order: { id: 'order-1' } });
    await waitFor(() => expect(openPanel).toHaveBeenCalledTimes(1));
    expect(router.state.location.search).toBe('');
  });

  it.each([{ canRead: false }, { error: true }])(
    '无权或加载失败时清理意图，之后刷新不循环弹出：%j',
    async (state) => {
      const { router, update } = renderEntry(
        '/orders/sea-export/order-1?panel=abnormal',
        state,
      );
      await waitFor(() => expect(router.state.location.search).toBe(''));
      expect(openPanel).not.toHaveBeenCalled();
      update({ canRead: true, error: false });
      expect(openPanel).not.toHaveBeenCalled();
    },
  );
});
