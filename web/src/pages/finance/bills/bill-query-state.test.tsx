import { act, render } from '@testing-library/react';
import { App } from 'antd';
import dayjs from 'dayjs';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  ListQueryStateProvider,
  useListQueryState,
} from '@/components/ui/list-query-state';
import { settlementServiceListBills } from '@/services/roncin/settlementService';
import FinanceBillsPage from './index';

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
  initialLabels: Record<string, string>;
  onSearch: (v: Record<string, unknown>) => void;
  onReset: () => void;
};
const state = vi.hoisted(() => ({
  ledger: undefined as undefined | LedgerProps,
  search: undefined as undefined | FilterProps,
  reload: vi.fn(),
  organizationId: 'workspace-org' as string | undefined,
  organizations: vi.fn(),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { currentOrganization: { id: state.organizationId } },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => true }),
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
}));
vi.mock('./components/BillEditModal', () => ({ default: () => null }));
vi.mock('./components/BillDetailDrawer', () => ({ default: () => null }));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListBills: vi
    .fn()
    .mockResolvedValue({ data: [], total: 0, matchedSummary: {} }),
  settlementServiceListFinanceOrganizationOptions: state.organizations,
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
  state.organizationId = 'workspace-org';
});
describe('账单查询恢复消费', () => {
  it('恢复已提交日期、候选名称和分页，旧公司筛选不能覆盖当前工作区，提交同步生效并回第一页，重置清查询', async () => {
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
    expect(state.ledger?.topBar).toBeUndefined();
    expect(state.ledger?.batchActions).toBeUndefined();
    expect(state.organizations).not.toHaveBeenCalled();
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
        organizationId: 'workspace-org',
        billDateFrom: '2026-10-01',
        dueDateTo: '2026-11-12',
      }),
      { skipErrorHandler: true },
    );
    act(() => {
      state.search?.onSearch({ keyword: '新的条件' });
    });
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 20 });
    });
    expect(settlementServiceListBills).toHaveBeenLastCalledWith(
      expect.objectContaining({
        keyword: '新的条件',
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
        organizationId: 'workspace-org',
      }),
      { skipErrorHandler: true },
    );
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
    expect(state.organizations).not.toHaveBeenCalled();
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
});
