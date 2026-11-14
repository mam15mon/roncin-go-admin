import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act } from '@testing-library/react';
import { App } from 'antd';
import dayjs from 'dayjs';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import { appInitialStateContext } from '@/app/initialStateContext';
import {
  ListQueryStateProvider,
  useListQueryState,
} from '@/components/ui/list-query-state';
import { settlementServiceListFeeLedger } from '@/services/roncin/settlementService';
import FinanceFeeLedgerPage from './index';

type LedgerProps = {
  actionRef: { current: unknown };
  customSearch: ReactNode;
  request: (p: { current: number; pageSize: number }) => Promise<unknown>;
};
const state = vi.hoisted(() => ({
  ledger: undefined as LedgerProps | undefined,
  search: undefined as
    | {
        visibleNames: string[];
        onSearch: (v: Record<string, unknown>) => void;
      }
    | undefined,
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => true }),
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'u1', currentOrganization: { id: 'o1' } },
    },
  }),
}));
vi.mock('@/components/ui', () => ({
  FinanceLedgerTemplate: (p: LedgerProps) => {
    state.ledger = p;
    p.actionRef.current = { reload: vi.fn() };
    return p.customSearch;
  },
  RowColorSettings: () => null,
  getDefaultRowColors: () => ({}),
}));
vi.mock('./components/FeeLedgerSearchFilter', async (orig) => ({
  ...(await orig<typeof import('./components/FeeLedgerSearchFilter')>()),
  FeeLedgerSearchFilter: (p: never) => {
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
  q.commit({
    search: {
      masterNo: 'MBL1',
      orderNo: '订单条件',
      createdAtRange: [dayjs('2026-10-01'), dayjs('2026-10-12')],
    },
  } as never);
  return null;
}

describe('费用台账筛选设置接入', () => {
  it('页签恢复剔除已隐藏筛选项，请求不再携带隐藏条件', async () => {
    window.localStorage.setItem(
      'roncin:filter-settings:v1:finance:fees:u1:o1',
      JSON.stringify({ visible: [], hidden: ['masterNo', 'createdAtRange'] }),
    );
    const ctx = {
      initialState: {
        currentUser: { id: 'u1', currentOrganization: { id: 'o1' } },
      },
    } as never;
    const { rerender, queryClient } = renderWithClient(
      <ListQueryStateProvider enabled>
        <Seed />
      </ListQueryStateProvider>,
    );
    rerender(
      <QueryClientProvider client={queryClient}>
        <ListQueryStateProvider enabled>
          <App>
            <appInitialStateContext.Provider value={ctx}>
              <FinanceFeeLedgerPage />
            </appInitialStateContext.Provider>
          </App>
        </ListQueryStateProvider>
      </QueryClientProvider>,
    );
    expect(state.search?.visibleNames).not.toContain('masterNo');
    await act(async () => {
      await state.ledger?.request({ current: 1, pageSize: 40 });
    });
    const params = vi.mocked(settlementServiceListFeeLedger).mock
      .lastCall?.[0] as Record<string, unknown>;
    expect(params.keyword).toBe('订单条件');
    expect(params.masterNo).toBeUndefined();
    expect(params.createdAtFrom).toBeUndefined();
    expect(params.createdAtTo).toBeUndefined();
    window.localStorage.clear();
  });
});
