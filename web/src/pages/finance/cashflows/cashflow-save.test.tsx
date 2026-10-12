import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import dayjs from 'dayjs';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { settlementServiceCreateCashflow } from '@/services/roncin/settlementService';
import FinanceCashflowsPage from './index';

const state = vi.hoisted(() => ({ reload: vi.fn(), results: [] as boolean[] }));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canCreateFinanceCashflows: true,
    canOperateOrganization: () => true,
  }),
}));
vi.mock('@/components/ui', () => ({
  FinanceLedgerTemplate: ({
    onPrimaryAction,
    actionRef,
  }: {
    onPrimaryAction: () => void;
    actionRef: { current: unknown };
  }) => {
    actionRef.current = { reload: state.reload };
    return (
      <button type="button" onClick={onPrimaryAction}>
        登记流水
      </button>
    );
  },
  ProFormSearchableSelect: () => null,
}));
vi.mock('@/features/finance/credit-control', () => ({
  useCreditLimitIntervention: () => false,
  disableCreditExceededOptions: vi.fn(),
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([]),
}));
vi.mock('@ant-design/pro-components', () => ({
  ModalForm: (p: {
    open: boolean;
    onFinish: (v: unknown) => Promise<boolean>;
    children: ReactNode;
  }) =>
    p.open ? (
      <div>
        <input aria-label="流水备注" defaultValue="未保存备注" />
        <button
          type="button"
          onClick={() =>
            void p
              .onFinish({
                organizationId: 'org',
                direction: 'RECEIVABLE',
                settlementPartyId: 'partner',
                currency: 'CNY',
                amount: '100',
                transactionDate: dayjs('2026-10-12'),
                ourAccount: 'account',
                paymentMethod: 'BANK_TRANSFER',
              })
              .then((result) => state.results.push(result))
          }
        >
          提交流水
        </button>
      </div>
    ) : null,
  ProFormDatePicker: () => null,
  ProFormText: () => null,
  ProFormTextArea: () => null,
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceCreateCashflow: vi.fn(),
  settlementServiceListFinanceOrganizationOptions: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));
beforeEach(() => {
  vi.clearAllMocks();
  state.results = [];
  vi.mocked(settlementServiceCreateCashflow).mockReset();
});
describe('资金流水登记失败判定', () => {
  it('reject、undefined、缺ID返回false且不关闭；成功才返回true并刷新', async () => {
    vi.mocked(settlementServiceCreateCashflow)
      .mockRejectedValueOnce(new Error('模拟失败'))
      .mockResolvedValueOnce(undefined as never)
      .mockResolvedValueOnce({ data: {} })
      .mockResolvedValueOnce({ success: false, data: { id: 'cashflow' } })
      .mockResolvedValueOnce({ data: { id: 'cashflow' } });
    renderWithClient(
      <App>
        <FinanceCashflowsPage />
      </App>,
    );
    fireEvent.click(screen.getByText('登记流水'));
    fireEvent.change(screen.getByLabelText('流水备注'), {
      target: { value: '编辑备注' },
    });
    for (let n = 1; n <= 4; n++) {
      fireEvent.click(screen.getByText('提交流水'));
      await waitFor(() => expect(state.results).toHaveLength(n));
      expect(state.results[n - 1]).toBe(false);
      expect(state.reload).not.toHaveBeenCalled();
      expect(screen.getByLabelText('流水备注')).toHaveValue('编辑备注');
      expect(screen.queryByText('资金流水已登记')).not.toBeInTheDocument();
    }
    fireEvent.click(screen.getByText('提交流水'));
    await waitFor(() =>
      expect(state.results).toEqual([false, false, false, false, true]),
    );
    expect(screen.queryByLabelText('流水备注')).not.toBeInTheDocument();
    expect(state.reload).toHaveBeenCalledTimes(1);
  });
});
