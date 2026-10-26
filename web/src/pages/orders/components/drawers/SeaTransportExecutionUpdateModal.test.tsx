import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import dayjs from 'dayjs';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { orderServiceGetOrder } from '@/services/roncin/orderService';
import {
  seaOrderChangeServiceExecuteSeaTransportExecutionUpdate,
  seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
} from '@/services/roncin/seaOrderChangeService';
import SeaTransportExecutionUpdateModal from './SeaTransportExecutionUpdateModal';

vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServicePreviewSeaTransportExecutionUpdate: vi.fn(),
  seaOrderChangeServiceExecuteSeaTransportExecutionUpdate: vi.fn(),
}));

vi.mock('@/services/roncin/orderService', () => ({
  orderServiceGetOrder: vi.fn(),
}));

vi.mock('@/services/roncin/orderAttachmentService', () => ({
  orderAttachmentServiceListAttachments: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

function renderModal(
  searchLocations?: (
    keyword?: string,
  ) => Promise<{ value: string; label: string }[]>,
) {
  const onClose = vi.fn();
  const onSuccess = vi.fn();
  const view = renderWithClient(
    <App>
      <SeaTransportExecutionUpdateModal
        open
        order={{
          id: 'order-1',
          orderNo: 'SE-001',
          seaMasterBill: {
            transportExecutionId: 'te-1',
            transportExecutionVersion: '3',
            voyageNo: 'V001',
            originLocationId: 'port-1',
            originLocationName: '上海港',
            etd: '2026-11-01T05:00:00Z',
          },
        }}
        onClose={onClose}
        onSuccess={onSuccess}
        searchLocations={searchLocations}
      />
    </App>,
  );
  fireEvent.change(screen.getByLabelText('调整原因'), {
    target: { value: '船代通知' },
  });
  fireEvent.change(screen.getByLabelText('外部确认方'), {
    target: { value: 'XX 船代' },
  });
  fireEvent.change(screen.getByLabelText('确认说明'), {
    target: { value: '邮件确认' },
  });
  return { ...view, onClose, onSuccess };
}

describe('SeaTransportExecutionUpdateModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.mocked(orderServiceGetOrder).mockImplementation(async ({ id }) => ({
      data: { id, orderNo: id === 'order-1' ? 'SE-001' : 'SE-002' },
    }));
  });
  it('预览共享影响后携带外部确认执行统一航次调整', async () => {
    vi.mocked(
      seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
    ).mockResolvedValue({
      data: {
        executable: true,
        memberOrderIds: ['order-1', 'order-2'],
        differences: [
          {
            fieldName: 'voyage_no',
            label: '航次',
            currentValue: 'V001',
            targetValue: 'V002',
          },
        ],
      },
    });
    vi.mocked(
      seaOrderChangeServiceExecuteSeaTransportExecutionUpdate,
    ).mockResolvedValue({
      success: true,
      transportExecution: { id: 'te-1' },
      versionId: 'version-2',
    });
    const onSuccess = vi.fn();

    renderWithClient(
      <App>
        <SeaTransportExecutionUpdateModal
          open
          order={{
            id: 'order-1',
            orderNo: 'SE-001',
            seaMasterBill: {
              transportExecutionId: 'te-1',
              transportExecutionVersion: '3',
              vesselName: 'VESSEL-A',
              voyageNo: 'V001',
            } as API.SeaMasterBillSummary & {
              transportExecutionVersion: string;
            },
          }}
          onClose={vi.fn()}
          onSuccess={onSuccess}
        />
      </App>,
    );

    fireEvent.change(screen.getByLabelText('航次'), {
      target: { value: 'V002' },
    });
    fireEvent.change(screen.getByLabelText('调整原因'), {
      target: { value: '船代通知换航次' },
    });
    fireEvent.change(screen.getByLabelText('外部确认方'), {
      target: { value: 'XX 船代' },
    });
    fireEvent.change(screen.getByLabelText('确认说明'), {
      target: { value: '邮件确认可调整' },
    });
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));

    await waitFor(() => {
      expect(
        seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
      ).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        expect.objectContaining({
          expectedTransportExecutionVersion: '3',
          input: expect.objectContaining({ voyageNo: 'V002' }),
        }),
      );
      expect(screen.getByText('本次将影响 2 张关联订单')).toBeInTheDocument();
    });
    expect(await screen.findByRole('link', { name: 'SE-001' })).toHaveAttribute(
      'href',
      '/orders/sea-export/order-1',
    );
    expect(await screen.findByRole('link', { name: 'SE-002' })).toHaveAttribute(
      'href',
      '/orders/sea-export/order-2',
    );
    expect(orderServiceGetOrder).toHaveBeenCalledTimes(2);

    fireEvent.click(screen.getByRole('button', { name: '确认统一调整' }));
    await waitFor(() => {
      expect(
        seaOrderChangeServiceExecuteSeaTransportExecutionUpdate,
      ).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        expect.objectContaining({
          expectedTransportExecutionVersion: '3',
          confirmation: expect.objectContaining({
            confirmedByParty: 'XX 船代',
            confirmationNote: '邮件确认可调整',
          }),
        }),
      );
      expect(onSuccess).toHaveBeenCalledTimes(1);
    });
  });

  it('按预览精确范围补读订单，失败保留对应 ID、数量与可读差异', async () => {
    vi.mocked(
      seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
    ).mockResolvedValue({
      data: {
        executable: false,
        memberOrderIds: ['cross-master-order', 'failed-order'],
        differences: [
          {
            fieldName: 'origin_location_id',
            label: '起运港',
            currentValue: 'port-1',
            targetValue: '',
          },
          {
            fieldName: 'etd',
            label: 'ETD',
            currentValue: '2026-11-01T05:00:00Z',
            targetValue: '2026-11-02T05:00:00Z',
          },
        ],
        impacts: [
          {
            factType: 'BUSINESS_LOCK',
            referenceId: 'failed-order',
            message: '关联订单已锁定',
          },
        ],
      },
    });
    vi.mocked(orderServiceGetOrder).mockImplementation(async ({ id }) => {
      if (id === 'failed-order') throw new Error('读取失败');
      return { data: { id, orderNo: 'SE-CROSS-MBL' } };
    });
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));

    expect(
      await screen.findByRole('link', { name: 'SE-CROSS-MBL' }),
    ).toBeInTheDocument();
    expect(
      await screen.findByText('订单读取失败（failed-order）'),
    ).toBeInTheDocument();
    expect(screen.getByText('本次将影响 2 张关联订单')).toBeInTheDocument();
    expect(orderServiceGetOrder).toHaveBeenCalledTimes(2);
    expect(orderServiceGetOrder).not.toHaveBeenCalledWith({ id: 'order-1' });
    expect(screen.getByText('未填写')).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: '上海港' })).toBeInTheDocument();
    expect(
      screen.getByRole('cell', {
        name: dayjs('2026-11-01T05:00:00Z').format('YYYY-MM-DD'),
      }),
    ).toBeInTheDocument();
    expect(screen.getByText('关联订单已锁定')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '确认统一调整' })).toBeDisabled();
    expect(
      seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
    ).toHaveBeenCalledWith(
      { orderId: 'order-1' },
      expect.objectContaining({
        input: expect.objectContaining({
          etd: dayjs('2026-11-01T05:00:00Z').startOf('day').toISOString(),
        }),
      }),
    );
  });

  it('修改输入立即清除预览与成员，失败的执行不关闭弹窗', async () => {
    vi.mocked(
      seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
    ).mockResolvedValue({
      data: { executable: true, memberOrderIds: ['order-1'] },
    });
    vi.mocked(
      seaOrderChangeServiceExecuteSeaTransportExecutionUpdate,
    ).mockResolvedValue({ success: false });
    const { onClose, onSuccess } = renderModal();
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    await screen.findByRole('link', { name: 'SE-001' });
    fireEvent.click(screen.getByRole('button', { name: '确认统一调整' }));
    await waitFor(() =>
      expect(
        seaOrderChangeServiceExecuteSeaTransportExecutionUpdate,
      ).toHaveBeenCalledTimes(1),
    );
    expect(onClose).not.toHaveBeenCalled();
    expect(onSuccess).not.toHaveBeenCalled();

    fireEvent.change(screen.getByLabelText('航次'), {
      target: { value: 'V003' },
    });
    expect(screen.queryByText(/本次将影响/)).not.toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: 'SE-001' }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '确认统一调整' })).toBeDisabled();
  });

  it('预览期间修改输入后，迟到响应不能恢复旧预览或订单范围', async () => {
    let resolvePreview!: (
      value: API.PreviewSeaTransportExecutionUpdateResponse,
    ) => void;
    vi.mocked(
      seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
    ).mockImplementation(
      () =>
        new Promise((resolve) => {
          resolvePreview = resolve;
        }),
    );
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    await waitFor(() =>
      expect(
        seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
      ).toHaveBeenCalledTimes(1),
    );
    fireEvent.change(screen.getByLabelText('航次'), {
      target: { value: 'V004' },
    });
    resolvePreview({
      data: { executable: true, memberOrderIds: ['obsolete-order'] },
    });
    await waitFor(() =>
      expect(screen.getByRole('button', { name: '预览影响' })).toBeEnabled(),
    );
    expect(screen.queryByText(/本次将影响/)).not.toBeInTheDocument();
    expect(orderServiceGetOrder).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: '确认统一调整' })).toBeDisabled();
  });

  it('旧港口搜索迟到不覆盖当前结果，已选港口名称保持可读', async () => {
    const oldSearch =
      Promise.withResolvers<{ value: string; label: string }[]>();
    const searchLocations = vi.fn(async (keyword?: string) => {
      if (keyword === '上海') return oldSearch.promise;
      if (keyword === '宁波') return [{ value: 'port-2', label: '宁波新结果' }];
      return [];
    });
    renderModal(searchLocations);
    const origin = screen.getByRole('combobox', { name: '起运港' });
    fireEvent.focus(origin);
    fireEvent.mouseDown(origin);
    fireEvent.change(origin, { target: { value: '上海' } });
    await waitFor(() => expect(searchLocations).toHaveBeenCalledWith('上海'));
    fireEvent.change(origin, { target: { value: '宁波' } });
    await screen.findByTitle('宁波新结果');
    await act(async () =>
      oldSearch.resolve([{ value: 'old-port', label: '上海旧结果' }]),
    );
    expect(screen.queryByTitle('上海旧结果')).not.toBeInTheDocument();
    expect(screen.getByTitle('宁波新结果')).toBeInTheDocument();
    expect(origin.closest('.ant-select')).toHaveTextContent('上海港');
  });

  it('旧预览迟到失败不提示当前输入的错误', async () => {
    const oldPreview =
      Promise.withResolvers<API.PreviewSeaTransportExecutionUpdateResponse>();
    vi.mocked(
      seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
    ).mockReturnValue(oldPreview.promise);
    renderModal();
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    await waitFor(() =>
      expect(
        seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
      ).toHaveBeenCalledOnce(),
    );
    fireEvent.change(screen.getByLabelText('航次'), {
      target: { value: 'V005' },
    });
    await act(async () => oldPreview.reject(new Error('过期预览失败')));
    expect(screen.queryByText('过期预览失败')).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '预览影响' })).toBeEnabled();
    expect(screen.getByRole('button', { name: '确认统一调整' })).toBeDisabled();
  });
});
