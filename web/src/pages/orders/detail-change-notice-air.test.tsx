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
      content: <AirReferenceFields />,
    },
  ],
}));

const state = getPortSaveState();

function AirReferenceFields() {
  state.form = Form.useFormInstance();
  return (
    <Form.Item name="customerReferenceNo" label="客户参考号">
      <Input />
    </Form.Item>
  );
}

describe('空运详情共用显式刷新保护', () => {
  beforeEach(() => {
    resetPortSaveState();
    state.routeKind = 'air-export';
    state.order = {
      ...state.order,
      orderNo: 'AE0001',
      customerReferenceNo: 'server-reference',
      seaMasterBill: undefined,
      seaDocumentLinkVersion: undefined,
      seaDocumentStructure: undefined,
      airExport: {
        documentMode: 'HOUSE',
        active: true,
        master: { id: 'mawb-1', version: '3', content: {} },
        house: {
          id: 'hawb-1',
          houseNo: 'HAWB1',
          version: '2',
          status: 'DRAFT',
          content: { remarks: '已保存正文' },
        },
        linkId: 'link-1',
        linkVersion: '1',
      },
    };
  });

  it('空运启用自身检查，显式刷新仍确认且在读取期间冻结输入', async () => {
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
    expect(getOrderReadMock()).toHaveBeenCalledOnce();
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
    expect(getOrderReadMock()).toHaveBeenCalledOnce();
  });
  it.each(['master', 'house', 'link'] as const)(
    '只变化航空 %s 版本时持续提醒且保留输入与原基线',
    async (target) => {
      const previous = state.order.airExport;
      if (!previous) throw new Error('测试缺少航空基线');
      const view = renderDetail();
      fireEvent.change(screen.getByLabelText('客户参考号'), {
        target: { value: '未保存修改' },
      });
      const latest: API.Order = {
        ...state.order,
        airExport: {
          ...previous,
          ...(target === 'master'
            ? { master: { ...previous.master, version: '4' } }
            : {}),
          ...(target === 'house'
            ? { house: { ...previous.house, version: '3' } }
            : {}),
          ...(target === 'link' ? { linkVersion: '2' } : {}),
        },
      };
      getOrderReadMock().mockResolvedValue({ success: true, data: latest });
      await act(async () => {
        await view.queryClient.refetchQueries({
          queryKey: ['orders', 'change-notice'],
        });
      });
      expect(
        await screen.findByText('这个订单已经发生变动，请重新刷新。'),
      ).toBeVisible();
      expect(screen.getByLabelText('客户参考号')).toHaveValue('未保存修改');
      expect(state.form?.getFieldValue('version')).toBe('4');
      expect(state.form?.getFieldValue('airExport')).toEqual(previous);
      await act(async () => {
        await view.queryClient.refetchQueries({
          queryKey: ['orders', 'change-notice'],
        });
      });
      expect(
        screen.getByText('这个订单已经发生变动，请重新刷新。'),
      ).toBeVisible();
    },
  );
});
