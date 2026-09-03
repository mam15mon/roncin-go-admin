import React from 'react';
import { App } from 'antd';
import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi, beforeEach } from 'vitest';
import SeaOrderReassignmentModal from './SeaOrderReassignmentModal';
import * as changeService from '@/services/roncin/seaOrderChangeService';

describe('SeaOrderReassignmentModal', () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it('打开改配弹窗并展示母单变更对比差异项', async () => {
    const mockPreview: API.SeaOrderReassignmentPreviewData = {
      isValid: true,
      differences: [
        {
          fieldName: 'master_no',
          label: '提单号(MBL)',
          currentValue: 'OLDMBL123',
          targetValue: 'NEWMBL456',
          isDifferent: true,
        },
        {
          fieldName: 'vessel_name',
          label: '船名',
          currentValue: 'EVER GIVEN',
          targetValue: 'EVER GIVEN',
          isDifferent: false,
        },
      ],
      currentMasterBill: {
        masterNo: 'OLDMBL123',
        vesselName: 'EVER GIVEN',
      },
      targetMasterBill: {
        masterNo: 'NEWMBL456',
        vesselName: 'EVER GIVEN',
      },
    };

    vi.spyOn(changeService, 'seaOrderChangeServicePreviewSeaOrderReassignment').mockResolvedValue({
      data: mockPreview,
    } as Awaited<ReturnType<typeof changeService.seaOrderChangeServicePreviewSeaOrderReassignment>>);

    render(
      <App>
        <SeaOrderReassignmentModal
          open={true}
          orderId="order-123"
          orderNo="SE20260903001"
          onClose={vi.fn()}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    await waitFor(() => {
      expect(screen.getByText(/订单整票改配/)).toBeInTheDocument();
      expect(screen.getByText('手工录入新母单')).toBeInTheDocument();
      expect(screen.getByText('匹配已有共享母单')).toBeInTheDocument();
      expect(screen.getByText('改配原因说明')).toBeInTheDocument();
      expect(screen.getByText('责任归属类型')).toBeInTheDocument();
    });
  });
});
