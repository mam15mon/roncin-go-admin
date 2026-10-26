import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { screen, waitFor } from '@testing-library/react';
import { ConfigProvider, Form } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { props } from '../../sea-create-layout.fixtures';
import { getSeaBaseInfoFields } from './SeaBasicInfoSection';

describe('来源船公司初始选项', () => {
  it.each(['来源船公司', undefined])(
    '远程空候选仍保留来源名称 %s',
    async (shippingLineName) => {
      const searchShippingLines = vi.fn().mockResolvedValue([]);
      const { carrier } = getSeaBaseInfoFields({
        ...props,
        searchShippingLines,
        seaHouseOrderCreateSource: {
          orderId: 'order-1',
          masterBillId: 'mbl-1',
          masterNo: 'MBL001',
          transportExecutionId: 'te-1',
          shippingLineId: 'carrier-uuid',
          shippingLineName,
        },
      });
      render(
        <ConfigProvider theme={{ token: { motion: false } }}>
          <Form initialValues={{ shippingLineId: 'carrier-uuid' }}>
            {carrier}
          </Form>
        </ConfigProvider>,
      );
      const label = shippingLineName || '船公司名称读取失败';
      expect(await screen.findByText(label)).toBeVisible();
      await waitFor(() => expect(searchShippingLines).toHaveBeenCalled());
      expect(screen.getByText(label)).toBeVisible();
      expect(screen.queryByText('carrier-uuid')).toBeNull();
    },
  );
});
