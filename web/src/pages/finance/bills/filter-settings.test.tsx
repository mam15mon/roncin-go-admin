import { act, render } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import { appInitialStateContext } from '@/app/initialStateContext';
import FinanceBillsPage from './index';

const state = vi.hoisted(() => ({
  search: undefined as undefined | { visibleNames: string[] },
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'u1', currentOrganization: { id: 'o1' } },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => true }),
}));
vi.mock('@/components/ui', async (orig) => ({
  ...(await orig<typeof import('@/components/ui')>()),
  FinanceLedgerTemplate: (p: { customSearch: React.ReactNode }) =>
    p.customSearch,
  SearchFilterTemplate: (p: { visibleNames: string[] }) => {
    state.search = p;
    return null;
  },
}));
vi.mock('@/features/finance/bill-creation', async (orig) => ({
  ...(await orig<typeof import('@/features/finance/bill-creation')>()),
  BillCreationWorkbench: () => null,
}));
vi.mock('@/services/roncin/settlementService', () => ({}));

describe('账单筛选设置接入', () => {
  it('偏好隐藏的账单日期不在可见字段中，其余默认可见', async () => {
    window.localStorage.setItem(
      'roncin:filter-settings:v1:finance:bills:u1:o1',
      JSON.stringify({ visible: [], hidden: ['billDateRange'] }),
    );
    render(
      <App>
        <appInitialStateContext.Provider
          value={
            {
              initialState: {
                currentUser: { id: 'u1', currentOrganization: { id: 'o1' } },
              },
            } as never
          }
        >
          <FinanceBillsPage />
        </appInitialStateContext.Provider>
      </App>,
    );
    await act(async () => {});
    expect(state.search?.visibleNames).toHaveLength(8);
    expect(state.search?.visibleNames).not.toContain('billDateRange');
    window.localStorage.clear();
  });
});
