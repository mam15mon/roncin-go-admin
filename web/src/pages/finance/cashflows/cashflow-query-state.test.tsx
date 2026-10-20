import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act } from '@testing-library/react';
import { App } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import {
  ListQueryStateProvider,
  useListQueryState,
} from '@/components/ui/list-query-state';
import { settlementServiceListCashflows } from '@/services/roncin/settlementService';
import FinanceCashflowsPage from './index';

type LedgerProps = {
  form: { initialValues: Record<string, unknown> };
  formRef: { current: unknown };
  onReset: () => void;
  onSubmit: (v: { direction: string; status: string }) => void;
  request: (p: {
    current: number;
    pageSize: number;
    direction?: string;
    status?: string;
  }) => Promise<unknown>;
};
const state = vi.hoisted(() => ({
  ledger: undefined as undefined | LedgerProps,
  searchForm: undefined as
    | undefined
    | { setFieldsValue: ReturnType<typeof vi.fn> },
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canCreateFinanceCashflows: false,
    canOperateOrganization: () => true,
  }),
}));
vi.mock('@/components/ui', async () => ({
  GuardedModalForm: (await import('@/components/ui/GuardedModalForm'))
    .GuardedModalForm,
  FinanceLedgerTemplate: (p: LedgerProps) => {
    state.ledger = p;
    state.searchForm = { setFieldsValue: vi.fn() };
    p.formRef.current = state.searchForm;
    return null;
  },
  ProFormSearchableSelect: () => null,
}));
vi.mock('@/features/finance/credit-control', () => ({
  useCreditLimitIntervention: () => false,
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([]),
}));
vi.mock('@ant-design/pro-components', () => ({
  ModalForm: () => null,
  ProFormDatePicker: () => null,
  ProFormText: () => null,
  ProFormTextArea: () => null,
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListCashflows: vi
    .fn()
    .mockResolvedValue({ data: [], total: 0 }),
  settlementServiceListFinanceOrganizationOptions: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));
function Seed() {
  const q = useListQueryState({
    tabKey: '/finance/cashflows',
    listId: 'finance:cashflows',
    defaultPageSize: 40,
    defaultFilters: {},
  });
  q.commit({ direction: 'PAYABLE', status: '2', organizationId: 'target-org' });
  q.pagination.onChange(2, 20);
  return null;
}
describe('流水原生搜索恢复消费', () => {
  it('恢复方向状态与初始化分页，重置清恢复条件与表单但保目标公司', async () => {
    const { rerender, queryClient } = renderWithClient(
      <ListQueryStateProvider enabled>
        <Seed />
      </ListQueryStateProvider>,
    );
    const { QueryClientProvider } = await import('@tanstack/react-query');
    rerender(
      <QueryClientProvider client={queryClient}>
        <ListQueryStateProvider enabled>
          <App>
            <FinanceCashflowsPage />
          </App>
        </ListQueryStateProvider>
      </QueryClientProvider>,
    );
    expect(state.ledger?.form.initialValues).toMatchObject({
      direction: 'PAYABLE',
      status: '2',
      current: 2,
      pageSize: 20,
    });
    await act(async () => {
      await state.ledger?.request({ current: 2, pageSize: 20 });
    });
    expect(settlementServiceListCashflows).toHaveBeenLastCalledWith(
      expect.objectContaining({
        page: 2,
        pageSize: 20,
        direction: 'PAYABLE',
        status: 2,
        organizationId: 'target-org',
      }),
    );
    act(() => state.ledger?.onReset());
    expect(state.searchForm?.setFieldsValue).toHaveBeenCalledWith(
      expect.objectContaining({ direction: undefined, status: undefined }),
    );
    await act(async () => {
      await state.ledger?.request({
        current: 1,
        pageSize: 20,
        direction: 'PAYABLE',
        status: '2',
      });
    });
    expect(settlementServiceListCashflows).toHaveBeenLastCalledWith(
      expect.objectContaining({
        direction: undefined,
        status: undefined,
        organizationId: 'target-org',
      }),
    );
    act(() => state.ledger?.onSubmit({ direction: 'RECEIVABLE', status: '1' }));
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 20 });
    });
    expect(settlementServiceListCashflows).toHaveBeenLastCalledWith(
      expect.objectContaining({ direction: 'RECEIVABLE', status: 1 }),
    );
  });
});
