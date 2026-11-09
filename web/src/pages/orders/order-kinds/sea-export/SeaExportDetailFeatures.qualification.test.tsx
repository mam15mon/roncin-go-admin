import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, Menu } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { seaOrderChangeServiceGetSeaOrderChangeActions } from '@/services/roncin/seaOrderChangeService';
import type { OrderDetailFeaturesContext } from '../types';
import SeaExportDetailFeatures from './SeaExportDetailFeatures';

vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServiceGetSeaOrderChangeActions: vi.fn(),
}));
vi.mock('../../components/drawers/SeaOrderReassignmentModal', () => ({
  default: ({ open, disabled }: { open: boolean; disabled: boolean }) => (
    <output
      data-testid="资格门控"
      data-open={String(open)}
      data-disabled={String(disabled)}
    />
  ),
}));
vi.mock('../../components/drawers/SeaTransportExecutionUpdateModal', () => ({
  default: () => null,
}));
vi.mock('../../components/drawers/SeaOrderChangeHistoryDrawer', () => ({
  default: () => null,
  SeaOrderChangeHistorySection: () => null,
}));
vi.mock('../../components/drawers/SeaSharedContainerDrawer', () => ({
  default: () => null,
}));

const getActions = vi.mocked(seaOrderChangeServiceGetSeaOrderChangeActions);
const context: OrderDetailFeaturesContext = {
  kind: 'sea-export',
  orderId: 'order-1',
  orderFormIdentity: 'sea-export:order-1',
  order: { id: 'order-1', version: '1' },
  businessWritesDisabled: false,
  canOrder: () => true,
  searchShippingLines: vi.fn(),
  searchLocations: vi.fn(),
  containerSpecOptions: [],
  refreshOrderAndLock: vi.fn(),
  ensureBusinessWriteAllowed: () => true,
};

function renderFeature() {
  let refresh: (() => Promise<void>) | undefined;
  render(
    <App>
      <SeaExportDetailFeatures context={context}>
        {(features) => {
          refresh = features.refreshTypeState;
          return (
            <>
              <Menu items={features.actionMenuItems} />
              {features.overlays}
            </>
          );
        }}
      </SeaExportDetailFeatures>
    </App>,
  );
  return { refresh: async () => refresh?.() };
}

beforeEach(() => {
  getActions.mockReset();
  getActions.mockResolvedValue({ success: true, data: { canReassign: true } });
});

describe('改配入口与已打开弹窗资格同步', () => {
  it.each(['省略允许标识', '请求失败'])(
    '%s 后两个入口同时禁止操作',
    async (scenario) => {
      const harness = renderFeature();
      const menu = await screen.findByRole('menuitem', { name: /改配/ });
      await waitFor(() =>
        expect(menu).not.toHaveAttribute('aria-disabled', 'true'),
      );
      fireEvent.click(menu);
      expect(screen.getByTestId('资格门控')).toHaveAttribute(
        'data-open',
        'true',
      );
      expect(screen.getByTestId('资格门控')).toHaveAttribute(
        'data-disabled',
        'false',
      );
      if (scenario === '省略允许标识') {
        getActions.mockResolvedValueOnce({
          success: true,
          data: { reassignBlockedReasons: ['订单状态已变化'] },
        });
      } else {
        getActions.mockRejectedValueOnce(new Error('资格加载失败'));
      }
      await act(async () => harness.refresh());
      expect(menu).toHaveAttribute('aria-disabled', 'true');
      expect(screen.getByTestId('资格门控')).toHaveAttribute(
        'data-disabled',
        'true',
      );
    },
  );
});
