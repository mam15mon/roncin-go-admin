import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { expect, it, vi } from 'vitest';
import {
  settlementServiceMarkCommissionPaid,
  settlementServicePreviewCommissionPayment,
} from '@/services/roncin/settlementService';
import CommissionPaymentModal from './CommissionPaymentModal';

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: { currentUser: { currentOrganization: { id: 'org-1' } } },
  }),
}));
vi.mock('@/services/roncin/settlementService', async (original) => ({
  ...(await original<typeof import('@/services/roncin/settlementService')>()),
  settlementServiceMarkCommissionPaid: vi
    .fn()
    .mockResolvedValue({ success: true }),
  settlementServicePreviewCommissionPayment: vi.fn(),
}));

it('登记实际净额与付款预览身份，父单和抵扣清单由同一提交处理', async () => {
  vi.mocked(settlementServicePreviewCommissionPayment).mockResolvedValue({
    data: {
      commissionId: 'c1',
      version: '3',
      originalAmount: '200',
      adjustmentAmount: '-50',
      actualAmount: '150',
      baseCurrency: 'CNY',
      previewToken: 'net-preview',
      adjustments: [
        {
          id: 'a1',
          adjustmentNo: 'ADJ-1',
          amount: '50',
          direction: 'DECREASE',
        },
      ],
    },
  });
  const paid = vi.fn();
  renderWithClient(
    <App>
      <CommissionPaymentModal
        record={{ id: 'c1', version: '3', commissionNo: 'COM-1' }}
        onClose={vi.fn()}
        onPaid={paid}
      />
    </App>,
  );
  expect(await screen.findByText('150.00 CNY')).toBeInTheDocument();
  expect(screen.getByText('ADJ-1')).toBeInTheDocument();
  fireEvent.click(screen.getByRole('checkbox'));
  fireEvent.click(screen.getByRole('button', { name: '登记实际发放' }));
  await waitFor(() => expect(paid).toHaveBeenCalledOnce());
  expect(settlementServiceMarkCommissionPaid).toHaveBeenCalledWith(
    { id: 'c1' },
    expect.objectContaining({
      expectedVersion: '3',
      actualAmount: '150',
      previewToken: 'net-preview',
      idempotencyKey: expect.any(String),
    }),
  );
  await act(async () => {});
});
