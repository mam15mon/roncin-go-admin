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
vi.mock('@/components/business-tag/BusinessTagModal', () => ({
  BusinessTagModal: () => null,
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
  settlementServiceListBills: vi.fn().mockResolvedValue({ data: [], total: 0 }),
  settlementServiceListFinanceOrganizationOptions: vi
    .fn()
    .mockResolvedValue({ data: [] }),
  settlementServiceListFinanceBillTagOptions: vi
    .fn()
    .mockResolvedValue({ tags: [] }),
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
      organizationId: 'target-org',
      tagIds: ['tag'],
      tagLabels: { tag: '标签显示名' },
    } as never,
    { settlementPartyId: '恢复结算单位' },
  );
  state.pagination.onChange(3, 20);
  return null;
}
beforeEach(() => vi.clearAllMocks());
describe('账单查询恢复消费', () => {
  it('已提交日期、候选名称、目标公司标签和分页恢复，提交同步生效并回第一页，重置清查询', async () => {
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
        organizationId: 'target-org',
        tagIds: ['tag'],
        billDateFrom: '2026-10-01',
        dueDateTo: '2026-11-12',
      }),
    );
    act(() => {
      state.search?.onSearch({ keyword: '新的条件' });
    });
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 20 });
    });
    expect(settlementServiceListBills).toHaveBeenLastCalledWith(
      expect.objectContaining({ keyword: '新的条件', page: 1 }),
    );
    expect(state.reload).toHaveBeenCalledWith(true);
    act(() => state.search?.onReset());
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 20 });
    });
    expect(settlementServiceListBills).toHaveBeenLastCalledWith(
      expect.objectContaining({
        keyword: undefined,
        organizationId: 'target-org',
        tagIds: ['tag'],
      }),
    );
  });
});
