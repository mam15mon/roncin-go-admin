import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider, Form } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { props } from '../../sea-create-layout.fixtures';
import { getSeaBaseInfoFields } from './SeaBasicInfoSection';

describe('来源船公司初始选项', () => {
  it('悬停已选船公司时显示完整名称', async () => {
    const name = '中远海运集装箱运输有限公司 / COSCO SHIPPING LINES';
    const { carrier } = getSeaBaseInfoFields({
      ...props,
      readonly: true,
      seaDocuments: {
        masterBill: { shippingLineId: 'carrier-uuid', shippingLineName: name },
      },
    });
    render(
      <Form initialValues={{ shippingLineId: 'carrier-uuid' }}>{carrier}</Form>,
    );
    fireEvent.mouseOver(await screen.findByText(name));
    expect(await screen.findByRole('tooltip')).toHaveTextContent(name);
  });
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
  it.each(['中远海运集运', undefined])(
    '退关只读字段显示保留单证名称 %s，重新进入仍不搜索',
    async (name) => {
      const searchShippingLines = vi.fn().mockResolvedValue([]);
      const renderCarrier = () => {
        const { carrier } = getSeaBaseInfoFields({
          ...props,
          readonly: true,
          isDetail: true,
          searchShippingLines,
          seaDocuments: {
            masterBill: {
              shippingLineId: 'carrier-uuid',
              shippingLineName: name,
            },
          },
        });
        return render(
          <ConfigProvider theme={{ token: { motion: false } }}>
            <Form initialValues={{ shippingLineId: 'carrier-uuid' }}>
              {carrier}
            </Form>
          </ConfigProvider>,
        );
      };
      const expected = name || '船公司名称读取失败';
      const first = renderCarrier();
      expect(await screen.findByText(expected)).toBeVisible();
      expect(screen.queryByText('carrier-uuid')).toBeNull();
      first.unmount();
      renderCarrier();
      expect(await screen.findByText(expected)).toBeVisible();
      expect(screen.queryByText('carrier-uuid')).toBeNull();
      expect(searchShippingLines).not.toHaveBeenCalled();
    },
  );
  it('共享活动主单仍使用自身船公司名称且禁止修改', async () => {
    const searchShippingLines = vi.fn().mockResolvedValue([]);
    const { carrier } = getSeaBaseInfoFields({
      ...props,
      isDetail: true,
      searchShippingLines,
      seaDocuments: {
        masterBill: {
          shippingLineId: 'other-id',
          shippingLineName: '其他船公司',
        },
      },
    });
    render(
      <ConfigProvider theme={{ token: { motion: false } }}>
        <Form
          initialValues={{
            shippingLineId: 'carrier-uuid',
            seaMasterBill: {
              shippingLineId: 'carrier-uuid',
              shippingLineName: '活动船公司',
              memberCount: 2,
            },
          }}
        >
          {carrier}
        </Form>
      </ConfigProvider>,
    );
    expect(await screen.findByText('活动船公司')).toBeVisible();
    expect(screen.getByRole('combobox')).toBeDisabled();
    expect(screen.queryByText('其他船公司')).toBeNull();
    expect(searchShippingLines).not.toHaveBeenCalled();
  });
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
          {fields.factory}
        </Form>
      </ConfigProvider>,
    );
    expect(screen.getAllByPlaceholderText('请输入')).toHaveLength(2);
    for (const input of screen.getAllByPlaceholderText('请输入'))
      expect(input).toBeDisabled();
    expect(screen.getByPlaceholderText('请输入工厂')).toBeDisabled();
  });
});
