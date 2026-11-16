import { fireEvent, render, screen } from '@testing-library/react';
import { App, Menu } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { seaOrderChangeServiceGetSeaOrderChangeActions } from '@/services/roncin/seaOrderChangeService';
import type { OrderDetailFeaturesContext } from '../types';
import SeaExportDetailFeatures from './SeaExportDetailFeatures';

vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServiceGetSeaOrderChangeActions: vi.fn(),
}));
vi.mock('../../components/drawers/SeaOrderReassignmentModal', () => ({
  default: () => null,
}));
vi.mock('../../components/drawers/SeaTransportExecutionUpdateModal', () => ({
  default: ({ open }: { open: boolean }) => (
    <output data-testid="共同航程弹窗">{String(open)}</output>
  ),
}));
vi.mock('../../components/drawers/SeaOrderChangeHistoryDrawer', () => ({
  default: () => null,
  SeaOrderChangeHistorySection: () => null,
}));
vi.mock('../../components/drawers/SeaSharedContainerDrawer', () => ({
  default: () => null,
}));

beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(seaOrderChangeServiceGetSeaOrderChangeActions).mockResolvedValue({
    data: { canSplit: true, canReassign: true },
  });
});

describe('共同航程真实品类权限', () => {
  it.each([
    {
      kind: 'sea-import',
      permission: 'update',
      allowed: true,
      readonly: false,
    },
    { kind: 'sea-import', permission: 'read', allowed: false, readonly: false },
    {
      kind: 'sea-import',
      permission: 'update',
      allowed: false,
      readonly: true,
    },
    {
      kind: 'sea-export',
      permission: 'update',
      allowed: false,
      readonly: false,
    },
    {
      kind: 'sea-export',
      permission: 'reassign',
      allowed: true,
      readonly: false,
    },
  ] as const)(
    '$kind/$permission/只读$readonly门禁',
    ({ kind, permission, allowed, readonly }) => {
      const context: OrderDetailFeaturesContext = {
        kind,
        orderId: 'order',
        orderFormIdentity: `${kind}:order`,
        order: {
          id: 'order',
          version: '1',
          businessType:
            kind === 'sea-import'
              ? OrderBusinessType.BUSINESS_TYPE_SI
              : OrderBusinessType.BUSINESS_TYPE_SE,
          seaMasterBill: { transportExecutionId: 'voyage' },
        },
        businessWritesDisabled: readonly,
        canOrder: (operation) => operation === permission,
        searchShippingLines: vi.fn(),
        searchLocations: vi.fn(),
        containerSpecOptions: [],
        refreshOrderAndLock: vi.fn(),
        ensureBusinessWriteAllowed: () => !readonly,
      };
      render(
        <App>
          <SeaExportDetailFeatures context={context}>
            {(features) => (
              <>
                {features.sharedVoyageMaintenance}
                <Menu
                  items={features.moreMenuItems?.filter(
                    (item) => item?.key === 'shared-voyage-update',
                  )}
                />
                <Menu items={features.actionMenuItems} />
                {features.overlays}
              </>
            )}
          </SeaExportDetailFeatures>
        </App>,
      );
      const button = screen.getByRole('button', { name: '维护共同航程' });
      expect(button).toHaveProperty('disabled', !allowed);
      if (allowed) {
        expect(
          screen.getByRole('menuitem', { name: /修改共享航程/ }),
        ).toBeVisible();
        fireEvent.click(button);
        expect(screen.getByTestId('共同航程弹窗')).toHaveTextContent('true');
      } else {
        expect(
          screen.queryByRole('menuitem', { name: /修改共享航程/ }),
        ).toBeNull();
        expect(screen.getByTestId('共同航程弹窗')).toHaveTextContent('false');
      }
      if (kind === 'sea-import') {
        expect(
          seaOrderChangeServiceGetSeaOrderChangeActions,
        ).not.toHaveBeenCalled();
        expect(
          screen.queryByRole('menuitem', { name: /整票改配|拆票/ }),
        ).toBeNull();
      }
    },
  );
});
