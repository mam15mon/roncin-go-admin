import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act } from '@testing-library/react';
import { App } from 'antd';
import dayjs from 'dayjs';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  ListQueryStateProvider,
  useListQueryState,
} from '@/components/ui/list-query-state';
import { settlementServiceListFeeLedger } from '@/services/roncin/settlementService';
import FinanceFeeLedgerPage from './index';

type LedgerProps = {
  columns: { dataIndex?: unknown }[];
  batchActions: { key: string }[];
  metricCards: { key: string; value: string | number }[];
  onAcceptedSnapshot: (snapshot: {
    total: number;
    summary?: API.FeeLedgerSummary;
  }) => void;
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
  company: true,
  organizationId: 'workspace-org' as string | undefined,
  ledger: undefined as LedgerProps | undefined,
  search: undefined as FilterProps | undefined,
  reload: vi.fn(),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => state.company }),
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
vi.mock('@/features/finance/bill-creation', () => ({
  BillCreationWorkbench: () => null,
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListFeeLedger: vi
    .fn()
    .mockResolvedValue({ data: [], total: 0, matchedSummary: {} }),
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
        masterNo: 'MBL12345',
        settlementPartyId: 'party',
        createdAtRange: [dayjs('2026-10-01'), dayjs('2026-10-12')],
        orderLocked: 'LOCKED',
      },
      organizationId: 'target-org',
    } as never,
    { settlementPartyId: '恢复结算单位' },
  );
  q.pagination.onChange(4, 50);
  return null;
}
beforeEach(() => {
  vi.clearAllMocks();
  state.company = true;
  state.organizationId = 'workspace-org';
});
describe('费用查询恢复消费', () => {
  it('消费后的缺失响应明确失败，不读取 undefined 或伪造空汇总', async () => {
    vi.mocked(settlementServiceListFeeLedger).mockResolvedValueOnce(
      undefined as unknown as API.ListFeeLedgerResponse,
    );
    renderWithClient(
      <ListQueryStateProvider enabled>
        <App>
          <FinanceFeeLedgerPage />
        </App>
      </ListQueryStateProvider>,
    );
    await expect(
      state.ledger?.request({ current: 1, pageSize: 40 }),
    ).rejects.toThrow('财务列表加载失败，请稍后重试');
    expect(settlementServiceListFeeLedger).toHaveBeenLastCalledWith(
      expect.objectContaining({ page: 1, pageSize: 40 }),
      { skipErrorHandler: true },
    );
  });

  it('顶部汇总金额统一千分位和两位小数，保留负值并将空金额显示为零', async () => {
    vi.mocked(settlementServiceListFeeLedger).mockResolvedValueOnce({
      data: [],
      total: '0',
      matchedSummary: {},
      summary: {
        amountsByBaseCurrency: [
          {
            baseCurrency: 'CNY',
            receivableBaseAmount: '3106.4252',
            payableBaseAmount: '1739.1046',
            profitBaseAmount: '1367.3206',
          },
          {
            baseCurrency: 'USD',
            payableBaseAmount: '111.00000000',
            profitBaseAmount: '-111.00000000',
          },
        ],
      },
    });
    renderWithClient(
      <ListQueryStateProvider enabled>
        <App>
          <FinanceFeeLedgerPage />
        </App>
      </ListQueryStateProvider>,
    );
    await act(async () => {
      const result = (await state.ledger?.request({
        current: 1,
        pageSize: 40,
      })) as { total: number; summary?: API.FeeLedgerSummary };
      state.ledger?.onAcceptedSnapshot(result);
    });
    expect(state.ledger?.metricCards).toEqual(
      expect.arrayContaining([
        expect.objectContaining({
          key: 'receivable-base',
          value: '3,106.43 CNY / 0.00 USD',
        }),
        expect.objectContaining({
          key: 'payable-base',
          value: '1,739.10 CNY / 111.00 USD',
        }),
        expect.objectContaining({
          key: 'profit-base',
          value: '1,367.32 CNY / -111.00 USD',
        }),
      ]),
    );
  });

  it('保存原始筛选而保留原keyword优先级，日期/公司/分页恢复，提交立即生效，重置保公司', async () => {
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
    expect(
      state.ledger?.columns.map((column) => column.dataIndex),
    ).not.toContain('tags');
    expect(
      state.ledger?.batchActions.some((action) => action.key === 'manage-tags'),
    ).toBe(false);
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
        masterNo: 'MBL12345',
        organizationId: 'workspace-org',
        createdAtFrom: '2026-10-01',
        createdAtTo: '2026-10-12',
        orderLocked: true,
      }),
      { skipErrorHandler: true },
    );
    act(() => state.search?.onSearch({ feeName: '新费用条件' }));
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 50 });
    });
    expect(settlementServiceListFeeLedger).toHaveBeenLastCalledWith(
      expect.objectContaining({ keyword: '新费用条件' }),
      { skipErrorHandler: true },
    );
    expect(state.reload).toHaveBeenCalledWith(true);
    expect(
      vi.mocked(settlementServiceListFeeLedger).mock.lastCall?.[0],
    ).not.toHaveProperty('tagIds');
    act(() => state.search?.onReset());
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 50 });
    });
    expect(settlementServiceListFeeLedger).toHaveBeenLastCalledWith(
      expect.objectContaining({
        keyword: undefined,
        organizationId: 'workspace-org',
      }),
      { skipErrorHandler: true },
    );
  });
  it.each(['未就绪', '系统工作台'])('%s不会查询经营数据', async (scenario) => {
    if (scenario === '未就绪') state.organizationId = undefined;
    else state.company = false;
    renderWithClient(
      <App>
        <FinanceFeeLedgerPage />
      </App>,
    );
    await expect(
      state.ledger?.request({ current: 1, pageSize: 20 }),
    ).resolves.toMatchObject({ success: false });
    expect(settlementServiceListFeeLedger).not.toHaveBeenCalled();
  });
});
