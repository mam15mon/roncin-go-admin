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
  it('锁定详情的订舱号、合约号和工厂均不可编辑', async () => {
    const fields = getSeaBaseInfoFields({
      ...props,
      readonly: true,
      isDetail: true,
    });
    render(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <Form>
          {fields.booking}
          {fields.contract}
          {fields.commercial}
        </Form>
      </ConfigProvider>,
    );
    expect(screen.getAllByPlaceholderText('请输入')).toHaveLength(2);
    for (const input of screen.getAllByPlaceholderText('请输入'))
      expect(input).toBeDisabled();
    expect(screen.getByPlaceholderText('请输入工厂')).toBeDisabled();
  });
});
