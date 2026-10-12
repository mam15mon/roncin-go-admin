import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act } from '@testing-library/react';
import { App } from 'antd';
import dayjs from 'dayjs';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import {
  ListQueryStateProvider,
  useListQueryState,
} from '@/components/ui/list-query-state';
import { settlementServiceListFeeLedger } from '@/services/roncin/settlementService';
import FinanceFeeLedgerPage from './index';

type LedgerProps = {
  pagination: { defaultCurrent: number; defaultPageSize: number };
  actionRef: { current: unknown };
  customSearch: ReactNode;
  request: (params: { current: number; pageSize: number }) => Promise<unknown>;
};
type FilterProps = {
  initialLabels: Record<string, string>;
  onSearch: (v: Record<string, unknown>) => void;
  onReset: () => void;
};
const state = vi.hoisted(() => ({
  ledger: undefined as LedgerProps | undefined,
  search: undefined as FilterProps | undefined,
  reload: vi.fn(),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => true }),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'user', currentOrganization: { id: 'workspace-org' } },
    },
  }),
}));
vi.mock('@/components/ui', () => ({
  FinanceLedgerTemplate: (p: LedgerProps) => {
    state.ledger = p;
    p.actionRef.current = { reload: state.reload };
    return p.customSearch;
  },
  RowColorSettings: () => null,
  getDefaultRowColors: () => ({}),
}));
vi.mock('./components/FeeLedgerSearchFilter', () => ({
  FeeLedgerSearchFilter: (p: FilterProps) => {
    state.search = p;
    return null;
  },
}));
vi.mock('@/components/business-tag/BusinessTagModal', () => ({
  BusinessTagModal: () => null,
}));
vi.mock('@/features/finance/bill-creation', () => ({
  BillCreationWorkbench: () => null,
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListFeeLedger: vi
    .fn()
    .mockResolvedValue({ data: [], total: 0 }),
  settlementServiceListFinanceOrganizationOptions: vi
    .fn()
    .mockResolvedValue({ data: [] }),
  settlementServiceListFinanceFeeTagOptions: vi
    .fn()
    .mockResolvedValue({ tags: [] }),
}));
function Seed() {
  const q = useListQueryState({
    tabKey: '/finance/fees',
    listId: 'finance:fees',
    defaultPageSize: 40,
    defaultFilters: { search: {} },
  });
  q.commit(
    {
      search: {
        orderNo: '订单条件',
        settlementPartyId: 'party',
        expenseDateRange: [dayjs('2026-10-01'), dayjs('2026-10-12')],
        financeLocked: 'LOCKED',
      },
      organizationId: 'target-org',
      tagIds: ['tag'],
      tagLabels: { tag: '标签显示名' },
    } as never,
    { settlementPartyId: '恢复结算单位' },
  );
  q.pagination.onChange(4, 50);
  return null;
}
describe('费用查询恢复消费', () => {
  it('保存原始筛选而保留原keyword优先级，日期/公司/标签/分页恢复，提交立即生效，重置保公司标签', async () => {
    const { rerender, queryClient } = renderWithClient(
      <ListQueryStateProvider enabled>
        <Seed />
      </ListQueryStateProvider>,
    );
    rerender(
      <QueryClientProvider client={queryClient}>
        <ListQueryStateProvider enabled>
          <App>
            <FinanceFeeLedgerPage />
          </App>
        </ListQueryStateProvider>
      </QueryClientProvider>,
    );
    expect(state.ledger?.pagination).toMatchObject({
      defaultCurrent: 4,
      defaultPageSize: 50,
    });
    expect(state.search?.initialLabels).toEqual({
      settlementPartyId: '恢复结算单位',
    });
    await act(async () => {
      await state.ledger?.request({ current: 4, pageSize: 50 });
    });
    expect(settlementServiceListFeeLedger).toHaveBeenLastCalledWith(
      expect.objectContaining({
        page: 4,
        pageSize: 50,
        keyword: '订单条件',
        organizationId: 'target-org',
        tagIds: ['tag'],
        expenseDateFrom: '2026-10-01',
        expenseDateTo: '2026-10-12',
        financeLocked: true,
      }),
    );
    act(() => state.search?.onSearch({ feeName: '新费用条件' }));
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 50 });
    });
    expect(settlementServiceListFeeLedger).toHaveBeenLastCalledWith(
      expect.objectContaining({ keyword: '新费用条件' }),
    );
    expect(state.reload).toHaveBeenCalledWith(true);
    act(() => state.search?.onReset());
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 50 });
    });
    expect(settlementServiceListFeeLedger).toHaveBeenLastCalledWith(
      expect.objectContaining({
        keyword: undefined,
        organizationId: 'target-org',
        tagIds: ['tag'],
      }),
    );
  });
});
