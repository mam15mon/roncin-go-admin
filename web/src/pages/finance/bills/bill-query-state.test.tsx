import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, render as renderUI } from '@testing-library/react';
import { App } from 'antd';
import dayjs from 'dayjs';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { appInitialStateContext } from '@/app/initialStateContext';
import {
  ListQueryStateProvider,
  useListQueryState,
} from '@/components/ui/list-query-state';
import { FinanceBillStatus } from '@/enums.generated';
import { settlementServiceListBills } from '@/services/roncin/settlementService';
import FinanceBillsPage from './index';

function render(ui: ReactNode) {
  const client = createTestQueryClient();
  return renderUI(ui, {
    wrapper: ({ children }) => (
      <QueryClientProvider client={client}>
        <appInitialStateContext.Provider
          value={
            {
              initialState: {
                currentUser: {
                  id: 'user',
                  currentOrganization: { id: state.organizationId },
                },
              },
            } as never
          }
        >
          {children}
        </appInitialStateContext.Provider>
      </QueryClientProvider>
    ),
  });
}

type LedgerProps = {
  columns: { dataIndex?: unknown }[];
  topBar?: ReactNode;
  batchActions?: unknown[];
  pagination: { defaultCurrent: number; defaultPageSize: number };
  actionRef: { current: unknown };
  customSearch: ReactNode;
  request: (params: { current: number; pageSize: number }) => Promise<unknown>;
};
type FilterProps = {
  items: {
    name: string;
    fieldProps?: { mode?: string };
    options?: { label: string }[];
  }[];
  initialLabels: Record<string, string>;
  onSearch: (v: Record<string, unknown>) => void;
  onReset: () => void;
};
const state = vi.hoisted(() => ({
  ledger: undefined as undefined | LedgerProps,
  search: undefined as undefined | FilterProps,
  reload: vi.fn(),
  organizationId: 'workspace-org' as string | undefined,
  company: true,
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user',
        currentOrganization: { id: state.organizationId },
      },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => state.company }),
}));
vi.mock('@/components/ui', () => ({
  FinanceLedgerTemplate: (p: LedgerProps) => {
    state.ledger = p;
    p.actionRef.current = { reload: state.reload };
    return p.customSearch;
  },
  SearchFilterTemplate: (p: FilterProps) => {
    state.search = p;
    return null;
  },
}));
vi.mock('@/features/finance/bill-creation', () => ({
  BillCreationWorkbench: () => null,
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([]),
  fetchCurrencyOptions: vi.fn().mockResolvedValue([]),
}));
vi.mock('./components/BillEditModal', () => ({ default: () => null }));
vi.mock('./components/BillDetailModal', () => ({ default: () => null }));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListBills: vi
    .fn()
    .mockResolvedValue({ data: [], total: 0, matchedSummary: {} }),
}));
function Seed() {
  const state = useListQueryState({
    tabKey: '/finance/bills',
    listId: 'finance:bills',
    defaultPageSize: 40,
    defaultFilters: { search: {} },
  });
  state.commit(
    {
      search: {
        keyword: '账单条件',
        businessStatuses: [1, 2],
        status: FinanceBillStatus.FINANCE_BILL_STATUS_CANCELLED,
        settlementPartyId: 'party',
        billDateRange: [dayjs('2026-10-01'), dayjs('2026-10-12')],
        dueDateRange: [dayjs('2026-11-01'), dayjs('2026-11-12')],
        onlyUnsettled: 'true',
      },
      organizationId: 'other-org',
    } as never,
    { settlementPartyId: '恢复结算单位' },
  );
  state.pagination.onChange(3, 20);
  return null;
}
beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
  state.organizationId = 'workspace-org';
  state.company = true;
});
describe('账单查询恢复消费', () => {
  it('恢复日期、候选和分页时仍只查询有效账单，提交与重置不改变有效范围和当前公司', async () => {
    const { rerender } = render(
      <ListQueryStateProvider enabled>
        <Seed />
      </ListQueryStateProvider>,
    );
    rerender(
      <ListQueryStateProvider enabled>
        <App>
          <FinanceBillsPage />
        </App>
      </ListQueryStateProvider>,
    );
    expect(
      state.ledger?.columns.map((column) => column.dataIndex),
    ).not.toContain('tags');
    expect(state.search?.items.map((field) => field.name)).not.toContain(
      'status',
    );
    expect(
      state.search?.items.find((field) => field.name === 'businessStatuses'),
    ).toMatchObject({
      fieldProps: { mode: 'multiple' },
      options: [
        { label: '未开票' },
        { label: '部分开票' },
        { label: '未核销' },
        { label: '部分核销' },
        { label: '已完成' },
      ],
    });
    expect(state.ledger?.topBar).toBeUndefined();
    expect(state.ledger?.batchActions).toBeUndefined();
    expect(state.ledger?.pagination).toMatchObject({
      defaultCurrent: 3,
      defaultPageSize: 20,
    });
    expect(state.search?.initialLabels).toEqual({
      settlementPartyId: '恢复结算单位',
    });
    await act(async () => {
      await state.ledger?.request({ current: 3, pageSize: 20 });
    });
    expect(settlementServiceListBills).toHaveBeenLastCalledWith(
      expect.objectContaining({
        page: 3,
        pageSize: 20,
        keyword: '账单条件',
        businessStatuses: [1, 2],
        status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
        organizationId: 'workspace-org',
        billDateFrom: '2026-10-01',
        dueDateTo: '2026-11-12',
      }),
      { skipErrorHandler: true },
    );
    act(() => {
      state.search?.onSearch({ keyword: '新的条件', businessStatuses: [3, 4] });
    });
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 20 });
    });
    expect(settlementServiceListBills).toHaveBeenLastCalledWith(
      expect.objectContaining({
        keyword: '新的条件',
        businessStatuses: [3, 4],
        status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
        page: 1,
        organizationId: 'workspace-org',
      }),
      { skipErrorHandler: true },
    );
    expect(state.reload).toHaveBeenCalledWith(true);
    expect(
      vi.mocked(settlementServiceListBills).mock.lastCall?.[0],
    ).not.toHaveProperty('tagIds');
    act(() => state.search?.onReset());
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 20 });
    });
    expect(settlementServiceListBills).toHaveBeenLastCalledWith(
      expect.objectContaining({
        keyword: undefined,
        businessStatuses: undefined,
        status: FinanceBillStatus.FINANCE_BILL_STATUS_CONFIRMED,
        organizationId: 'workspace-org',
      }),
      { skipErrorHandler: true },
    );
  });
  it('恢复页签时剔除已隐藏五态条件，其他已提交条件仍生效', async () => {
    window.localStorage.setItem(
      'roncin:filter-settings:v1:finance:bills:user:workspace-org',
      JSON.stringify({ visible: [], hidden: ['businessStatuses'] }),
    );
    const { rerender } = render(
      <ListQueryStateProvider enabled>
        <Seed />
      </ListQueryStateProvider>,
    );
    rerender(
      <ListQueryStateProvider enabled>
        <App>
          <FinanceBillsPage />
        </App>
      </ListQueryStateProvider>,
    );
    await act(async () => {
      await state.ledger?.request({ current: 3, pageSize: 20 });
    });
    expect(
      vi.mocked(settlementServiceListBills).mock.lastCall?.[0],
    ).toMatchObject({
      keyword: '账单条件',
      businessStatuses: undefined,
      organizationId: 'workspace-org',
    });
    window.localStorage.clear();
  });

  it('当前公司尚未就绪时不发送无组织边界的列表查询', async () => {
    state.organizationId = undefined;
    render(
      <App>
        <FinanceBillsPage />
      </App>,
    );
    await expect(
      state.ledger?.request({ current: 1, pageSize: 40 }),
    ).resolves.toEqual({ data: [], success: false, total: 0 });
    expect(settlementServiceListBills).not.toHaveBeenCalled();
  });

  it('工作区重建后只查询新公司，翻页仍使用当前公司', async () => {
    const first = render(
      <ListQueryStateProvider enabled>
        <App>
          <FinanceBillsPage />
        </App>
      </ListQueryStateProvider>,
    );
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 40 });
    });
    expect(
      vi.mocked(settlementServiceListBills).mock.lastCall?.[0],
    ).toMatchObject({ organizationId: 'workspace-org' });
    first.unmount();
    state.organizationId = 'next-workspace-org';
    render(
      <ListQueryStateProvider enabled>
        <App>
          <FinanceBillsPage />
        </App>
      </ListQueryStateProvider>,
    );
    await act(async () => {
      await state.ledger?.request({ current: 2, pageSize: 20 });
    });
    expect(
      vi.mocked(settlementServiceListBills).mock.lastCall?.[0],
    ).toMatchObject({
      organizationId: 'next-workspace-org',
      page: 2,
      pageSize: 20,
    });
  });
  it('非公司工作区有组织ID也不查询经营账单', async () => {
    state.company = false;
    render(
      <App>
        <FinanceBillsPage />
      </App>,
    );
    await expect(
      state.ledger?.request({ current: 1, pageSize: 20 }),
    ).resolves.toMatchObject({ success: false });
    expect(settlementServiceListBills).not.toHaveBeenCalled();
  });
});
