import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { Form, Input } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  bundle,
  getOrderReadMock,
  getPortSaveState,
  renderDetail,
  resetPortSaveState,
} from './detail-port-save.fixtures';
import type { OrderDetailFeaturesProps } from './order-kinds/types';

vi.mock('./order-kinds/air-export/AirExportDetailFeatures', () => ({
  default: ({ children }: OrderDetailFeaturesProps) => children({}),
}));
vi.mock('./templates/air-template', () => ({
  getAirTemplateSections: () => [
    {
      key: 'basic',
      title: '空运订单资料',
      content: (
        <Form.Item name="customerReferenceNo" label="客户参考号">
          <Input />
        </Form.Item>
      ),
    },
  ],
}));

const state = getPortSaveState();

describe('空运详情共用显式刷新保护', () => {
  beforeEach(() => {
    resetPortSaveState();
    state.routeKind = 'air-export';
    state.order = {
      ...state.order,
      orderNo: 'AE0001',
      customerReferenceNo: 'server-reference',
    };
  });

  it('空运不启用海运检查，显式刷新仍确认且在读取期间冻结输入', async () => {
    let finishRefresh!: (value: ReturnType<typeof bundle>) => void;
    state.loadData.mockImplementation(
      () =>
        new Promise<ReturnType<typeof bundle>>((resolve) => {
          finishRefresh = resolve;
        }),
    );
    const view = renderDetail();
    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: 'unsaved-reference' },
    });
    await act(async () => {
      await view.queryClient.refetchQueries({
        queryKey: ['orders', 'change-notice'],
      });
      fireEvent.click(screen.getByRole('button', { name: '刷新数据' }));
    });
    expect(getOrderReadMock()).not.toHaveBeenCalled();
    const dialog = await screen.findByRole('dialog');
    const confirm = within(dialog).getByRole('button', { name: '确定刷新' });
    await waitFor(() => expect(confirm).toBeVisible());
    await act(async () => {
      fireEvent.click(confirm);
    });
    expect(state.loadData).toHaveBeenCalledOnce();
    expect(screen.getByLabelText('客户参考号')).toBeDisabled();
    expect(screen.getByLabelText('客户参考号')).toHaveValue(
      'unsaved-reference',
    );
    state.order = {
      ...state.order,
      version: '5',
      customerReferenceNo: 'latest-reference',
    };
    await act(async () => {
      finishRefresh(bundle());
    });
    await waitFor(() => {
      expect(screen.getByLabelText('客户参考号')).toBeEnabled();
      expect(screen.getByLabelText('客户参考号')).toHaveValue(
        'latest-reference',
      );
    });
    expect(getOrderReadMock()).not.toHaveBeenCalled();
  });
});
