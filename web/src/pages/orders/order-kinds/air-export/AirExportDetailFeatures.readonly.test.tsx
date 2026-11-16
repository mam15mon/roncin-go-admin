import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen } from '@testing-library/react';
import { Form } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { history } from '@/router/history';
import type { OrderDetailFeaturesContext } from '../types';
import AirExportDetailFeatures from './AirExportDetailFeatures';

vi.mock('@/router/history', () => ({ history: { push: vi.fn() } }));
vi.mock('./AirCargoWorkspace', () => ({ default: () => null }));
vi.mock('./AirTransportMaintenance', () => ({ default: () => null }));
vi.mock('../../components/drawers/CargoItemDrawer', () => ({
  default: () => null,
}));

describe('航空只读订单单证入口', () => {
  it('审核后父表单禁用不阻止制单页面和历史查询导航', () => {
    const context: OrderDetailFeaturesContext = {
      kind: 'air-import',
      orderId: 'order',
      orderFormIdentity: 'air-import:order',
      order: {
        id: 'order',
        organizationId: 'company',
        businessType: OrderBusinessType.BUSINESS_TYPE_AI,
        airExport: {
          documentMode: 'HOUSE',
          master: { id: 'master' },
          house: { id: 'house' },
        },
      },
      businessWritesDisabled: true,
      canOrder: () => true,
      searchShippingLines: async () => [],
      searchLocations: async () => [],
      containerSpecOptions: [],
      refreshOrderAndLock: async () => {},
      ensureBusinessWriteAllowed: () => false,
    };
    renderWithApp(
      <AirExportDetailFeatures context={context}>
        {(features) => (
          <Form disabled>
            {features.airDocumentActions?.MAWB}
            {features.airDocumentActions?.HAWB}
          </Form>
        )}
      </AirExportDetailFeatures>,
    );
    for (const name of [
      '制作 / 预览 MAWB',
      '主单历史与成员',
      '制作 / 预览 HAWB',
      '查看分单历史',
    ])
      expect(screen.getByRole('button', { name })).toBeEnabled();
    fireEvent.click(screen.getByRole('button', { name: '制作 / 预览 MAWB' }));
    expect(history.push).toHaveBeenCalledWith(
      '/orders/air-import/masters/master/awb',
    );
    fireEvent.click(screen.getByRole('button', { name: '制作 / 预览 HAWB' }));
    expect(history.push).toHaveBeenCalledWith(
      '/orders/air-import/order/awb/HAWB',
    );
  });
});
