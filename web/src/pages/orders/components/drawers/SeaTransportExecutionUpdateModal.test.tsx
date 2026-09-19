import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import {
  seaOrderChangeServiceExecuteSeaTransportExecutionUpdate,
  seaOrderChangeServicePreviewSeaTransportExecutionUpdate,
} from '@/services/roncin/seaOrderChangeService';
import SeaTransportExecutionUpdateModal from './SeaTransportExecutionUpdateModal';

vi.mock('@/services/roncin/seaOrderChangeService', () => ({
  seaOrderChangeServicePreviewSeaTransportExecutionUpdate: vi.fn(),
  seaOrderChangeServiceExecuteSeaTransportExecutionUpdate: vi.fn(),
}));

vi.mock('@/services/roncin/orderAttachmentService', () => ({
  orderAttachmentServiceListAttachments: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

describe('SeaTransportExecutionUpdateModal', () => {
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
    ).mockResolvedValue({ success: true });
    const onSuccess = vi.fn();

    render(
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
});
