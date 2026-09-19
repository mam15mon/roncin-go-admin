import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, type MenuProps } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { seaOrderChangeServiceGetSeaOrderChangeActions } from '@/services/roncin/seaOrderChangeService';
import type { OrderDetailFeaturesContext } from '../types';
import SeaExportDetailFeatures from './SeaExportDetailFeatures';

const mockPush = vi.hoisted(() => vi.fn());

vi.mock('@umijs/max', () => ({
  history: { push: mockPush },
}));

vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServiceGetSeaOrderChangeActions: vi.fn(),
}));

vi.mock('../../components/drawers/SeaOrderReassignmentModal', () => ({
  default: ({
    open,
    onSuccess,
    onClose,
  }: {
    open: boolean;
    onSuccess?: () => Promise<void>;
    onClose?: () => void;
  }) => (
    <div data-testid="reassign-modal">
      {String(open)}
      <button type="button" onClick={() => void onSuccess?.()}>
        弹窗成功
      </button>
      <button type="button" onClick={onClose}>
        弹窗关闭
      </button>
    </div>
  ),
}));

vi.mock('../../components/drawers/SeaTransportExecutionUpdateModal', () => ({
  default: ({ open }: { open: boolean }) => (
    <div data-testid="voyage-modal">{String(open)}</div>
  ),
}));

vi.mock('../../components/drawers/SeaOrderChangeHistoryDrawer', () => ({
  default: ({ open }: { open: boolean }) => (
    <div data-testid="history-drawer">{String(open)}</div>
  ),
  SeaOrderChangeHistorySection: () => (
    <div data-testid="history-section">拆票与改配记录区块</div>
  ),
}));

vi.mock('../../components/drawers/SeaSharedContainerDrawer', () => ({
  default: ({
    open,
    transportExecutionId,
    orderId,
  }: {
    open: boolean;
    transportExecutionId?: string;
    orderId: string;
  }) => (
    <div data-testid="shared-container-drawer">
      {String(open)}|{orderId}|{transportExecutionId ?? ''}
    </div>
  ),
}));

const mockGetChangeActions = vi.mocked(
  seaOrderChangeServiceGetSeaOrderChangeActions,
);

function buildContext(
  overrides?: Partial<OrderDetailFeaturesContext>,
): OrderDetailFeaturesContext {
  return {
    kind: 'sea-export',
    orderId: 'ord-1',
    order: {
      id: 'ord-1',
      orderNo: 'SE-001',
      version: '1',
      seaMasterBill: { transportExecutionId: 'TE-1' },
    } as API.Order,
    orderFormIdentity: 'sea-export:ord-1',
    businessWritesDisabled: false,
    canOrder: () => true,
    searchShippingLines: vi.fn().mockResolvedValue([]),
    searchLocations: vi.fn().mockResolvedValue([]),
    containerSpecOptions: [],
    refreshOrderAndLock: vi.fn().mockResolvedValue(undefined),
    ensureBusinessWriteAllowed: () => true,
    ...overrides,
  };
}

function triggerMenuItem(items: MenuProps['items'] | undefined, key: string) {
  const item = items?.find((entry) => entry?.key === key) as
    | { onClick?: () => void }
    | undefined;
  item?.onClick?.();
}

function renderContribution(
  context: OrderDetailFeaturesContext,
  mountKey: string,
) {
  return (
    <SeaExportDetailFeatures key={mountKey} context={context}>
      {(features) => (
        <div>
          {features.headerActions}
          {features.overlays}
          <button
            type="button"
            onClick={() =>
              triggerMenuItem(features.moreMenuItems, 'shared-voyage-update')
            }
          >
            触发共享航次调整
          </button>
          <button
            type="button"
            onClick={() =>
              triggerMenuItem(
                features.moreMenuItems,
                'shared-container-workbench',
              )
            }
          >
            触发共享箱工作台
          </button>
          <button
            type="button"
            onClick={() =>
              triggerMenuItem(features.moreMenuItems, 'change-history')
            }
          >
            触发历史入口
          </button>
        </div>
      )}
    </SeaExportDetailFeatures>
  );
}

type Harness = {
  rerenderWith: (context: OrderDetailFeaturesContext, key?: string) => void;
};

function renderFeatures(
  context: OrderDetailFeaturesContext,
  mountKey = context.orderFormIdentity,
): Harness {
  const view = render(<App>{renderContribution(context, mountKey)}</App>);
  return {
    rerenderWith: (nextContext, key = nextContext.orderFormIdentity) =>
      view.rerender(<App>{renderContribution(nextContext, key)}</App>),
  };
}

describe('SeaExportDetailFeatures', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockPush.mockReset();
    mockGetChangeActions.mockResolvedValue({
      data: { canSplit: true, canReassign: true },
    } as never);
  });

  it('拆票与改配按钮按权限渲染，阻断原因进入 Tooltip', async () => {
    mockGetChangeActions.mockResolvedValue({
      data: {
        canSplit: false,
        splitBlockedReasons: ['该订单已拆票'],
        canReassign: true,
        reassignBlockedReasons: [],
      },
    } as never);
    renderFeatures(buildContext());

    const splitButton = await screen.findByRole('button', { name: /拆票/ });
    expect(splitButton).toBeDisabled();
    expect(screen.getByRole('button', { name: /改配/ })).toBeEnabled();

    fireEvent.mouseEnter(splitButton);
    expect(await screen.findByText('该订单已拆票')).toBeInTheDocument();
  });

  it('无对应操作权限时不渲染拆票或改配按钮', async () => {
    renderFeatures(buildContext({ canOrder: () => false }));

    expect(
      screen.queryByRole('button', { name: /拆票/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /改配/ }),
    ).not.toBeInTheDocument();
    // 挂载期动作资格请求在 act 内落地，避免用例结束后迟到更新
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });
  });

  it('拆票跳转经锁单写入口校验，锁单关闭时不导航', async () => {
    mockGetChangeActions.mockResolvedValue({
      data: { canSplit: true, canReassign: true },
    } as never);
    const harness = renderFeatures(
      buildContext({
        businessWritesDisabled: true,
        businessWriteBlockedReason: '订单已锁定',
        ensureBusinessWriteAllowed: () => false,
      }),
    );

    const splitButton = await screen.findByRole('button', { name: /拆票/ });
    expect(splitButton).toBeDisabled();
    fireEvent.click(splitButton);
    expect(mockPush).not.toHaveBeenCalledWith('/orders/sea-export/ord-1/split');

    harness.rerenderWith(
      buildContext({
        ensureBusinessWriteAllowed: () => true,
        businessWritesDisabled: false,
      }),
    );
    const enabledButton = await screen.findByRole('button', { name: /拆票/ });
    fireEvent.click(enabledButton);
    await waitFor(() =>
      expect(mockPush).toHaveBeenCalledWith('/orders/sea-export/ord-1/split'),
    );
  });

  it('改配弹窗成功后执行普通刷新命令并重载动作资格', async () => {
    const refreshOrderAndLock = vi.fn().mockResolvedValue(undefined);
    renderFeatures(buildContext({ refreshOrderAndLock }));

    await screen.findByRole('button', { name: /改配/ });
    expect(mockGetChangeActions).toHaveBeenCalledTimes(1);

    fireEvent.click(screen.getByRole('button', { name: /改配/ }));
    await waitFor(() =>
      expect(screen.getByTestId('reassign-modal')).toHaveTextContent('true'),
    );

    fireEvent.click(screen.getByRole('button', { name: '弹窗成功' }));
    await waitFor(() => {
      expect(refreshOrderAndLock).toHaveBeenCalledTimes(1);
      expect(mockGetChangeActions).toHaveBeenCalledTimes(2);
    });
  });

  it('业务写入关闭时自动关闭已打开的改配弹窗', async () => {
    const harness = renderFeatures(buildContext());

    fireEvent.click(await screen.findByRole('button', { name: /改配/ }));
    await waitFor(() =>
      expect(screen.getByTestId('reassign-modal')).toHaveTextContent('true'),
    );

    harness.rerenderWith(buildContext({ businessWritesDisabled: true }));
    await waitFor(() =>
      expect(screen.getByTestId('reassign-modal')).toHaveTextContent('false'),
    );
  });

  it('共享航次调整菜单项按 reassign 权限出现', async () => {
    const harness = renderFeatures(
      buildContext({ canOrder: (operation) => operation !== 'reassign' }),
    );

    await screen.findByRole('button', { name: /拆票/ });
    fireEvent.click(screen.getByRole('button', { name: '触发共享航次调整' }));
    expect(screen.getByTestId('voyage-modal')).toHaveTextContent('false');

    harness.rerenderWith(buildContext());
    await screen.findByRole('button', { name: /改配/ });
    fireEvent.click(screen.getByRole('button', { name: '触发共享航次调整' }));
    expect(screen.getByTestId('voyage-modal')).toHaveTextContent('true');
  });

  it('共享箱工作台缺少运输执行时仅提示，存在时以当前运输执行打开', async () => {
    renderFeatures(
      buildContext({
        order: { id: 'ord-1', orderNo: 'SE-001' } as API.Order,
      }),
    );

    await screen.findByRole('button', { name: /拆票/ });
    fireEvent.click(screen.getByRole('button', { name: '触发共享箱工作台' }));
    expect(screen.getByTestId('shared-container-drawer')).toHaveTextContent(
      'false|ord-1|',
    );
  });

  it('共享箱以订单当前运输执行打开，订单身份变化重挂载后立即关闭', async () => {
    const harness = renderFeatures(buildContext());

    await screen.findByRole('button', { name: /拆票/ });
    fireEvent.click(screen.getByRole('button', { name: '触发共享箱工作台' }));
    expect(screen.getByTestId('shared-container-drawer')).toHaveTextContent(
      'true|ord-1|TE-1',
    );

    harness.rerenderWith(
      buildContext({
        kind: 'sea-export',
        orderId: 'ord-2',
        order: {
          id: 'ord-2',
          orderNo: 'SE-002',
          seaMasterBill: { transportExecutionId: 'TE-2' },
        } as API.Order,
        orderFormIdentity: 'sea-export:ord-2',
      }),
    );

    await waitFor(() =>
      expect(screen.getByTestId('shared-container-drawer')).toHaveTextContent(
        'false|ord-2|',
      ),
    );
  });

  it('refreshTypeState 直接触发动作资格刷新', async () => {
    let boundRefresh: (() => Promise<void>) | undefined;
    render(
      <App>
        <SeaExportDetailFeatures context={buildContext()}>
          {(features) => {
            boundRefresh = features.refreshTypeState;
            return null;
          }}
        </SeaExportDetailFeatures>
      </App>,
    );

    expect(mockGetChangeActions).toHaveBeenCalledTimes(1);
    // 挂载期资格请求先在 act 内落地，再触发手动刷新
    await act(async () => {
      await Promise.resolve();
      await Promise.resolve();
      await Promise.resolve();
    });
    await waitFor(() => expect(boundRefresh).toBeDefined());
    await act(async () => {
      await boundRefresh?.();
    });
    expect(mockGetChangeActions).toHaveBeenCalledTimes(2);
  });
});
