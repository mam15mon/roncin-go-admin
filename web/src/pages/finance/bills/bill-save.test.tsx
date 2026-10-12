import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App, Form, type FormInstance, Input } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { settlementServiceUpdateBill } from '@/services/roncin/settlementService';
import FinanceBillsPage from './index';

const state = vi.hoisted(() => ({
  openEdit: undefined as undefined | ((bill: API.FinanceBill) => void),
  reload: vi.fn(),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => true }),
}));
vi.mock('@/components/ui', () => ({
  FinanceLedgerTemplate: ({
    actionRef,
  }: {
    actionRef: { current: unknown };
  }) => {
    actionRef.current = { reload: state.reload };
    return (
      <button
        type="button"
        onClick={() =>
          state.openEdit?.({
            id: 'bill',
            version: '1',
            statementTitle: '原抬头',
            billDate: '2026-10-12',
          })
        }
      >
        编辑账单
      </button>
    );
  },
  SearchFilterTemplate: () => null,
}));
vi.mock('@/components/business-tag/BusinessTagModal', () => ({
  BusinessTagModal: () => null,
}));
vi.mock('@/features/finance/bill-creation', () => ({
  BillCreationWorkbench: () => null,
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([]),
}));
vi.mock('./components/billColumns', () => ({
  getFinanceBillColumns: (p: {
    onOpenEdit: (bill: API.FinanceBill) => void;
  }) => {
    state.openEdit = p.onOpenEdit;
    return [];
  },
}));
vi.mock('./components/BillDetailDrawer', () => ({ default: () => null }));
vi.mock('./components/BillEditModal', () => ({
  default: (p: {
    open: boolean;
    form: FormInstance;
    onOk: () => Promise<void>;
  }) =>
    p.open ? (
      <Form form={p.form}>
        <Form.Item name="billDate" hidden getValueProps={() => ({})}>
          <Input />
        </Form.Item>
        <Form.Item name="statementTitle" label="抬头">
          <Input />
        </Form.Item>
        <button type="button" onClick={() => void p.onOk()}>
          提交账单
        </button>
      </Form>
    ) : null,
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceUpdateBill: vi.fn(),
  settlementServiceListFinanceOrganizationOptions: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));
beforeEach(() => {
  vi.clearAllMocks();
  vi.mocked(settlementServiceUpdateBill).mockReset();
});
describe('账单编辑失败判定', () => {
  it('reject、undefined、缺ID均保持编辑；实体响应成功才关闭并刷新', async () => {
    vi.mocked(settlementServiceUpdateBill)
      .mockRejectedValueOnce(new Error('模拟失败'))
      .mockResolvedValueOnce(undefined as never)
      .mockResolvedValueOnce({ data: {} })
      .mockResolvedValueOnce({ success: false, data: { id: 'bill' } })
      .mockResolvedValueOnce({ data: { id: 'bill' } });
    render(
      <App>
        <FinanceBillsPage />
      </App>,
    );
    fireEvent.click(screen.getByText('编辑账单'));
    fireEvent.change(screen.getByLabelText('抬头'), {
      target: { value: '编辑值' },
    });
    for (let n = 1; n <= 4; n++) {
      fireEvent.click(screen.getByText('提交账单'));
      await waitFor(() =>
        expect(settlementServiceUpdateBill).toHaveBeenCalledTimes(n),
      );
      if (n !== 2)
        await screen.findAllByText(
          n === 1 ? '模拟失败' : '更新账单失败，请重试',
        );
      expect(screen.getByLabelText('抬头')).toHaveValue('编辑值');
      expect(state.reload).not.toHaveBeenCalled();
      expect(
        screen.queryByText('账单已成功更新并自动刷新汇率快照'),
      ).not.toBeInTheDocument();
    }
    fireEvent.click(screen.getByText('提交账单'));
    await waitFor(() =>
      expect(screen.queryByLabelText('抬头')).not.toBeInTheDocument(),
    );
    expect(state.reload).toHaveBeenCalledTimes(1);
  });
});
