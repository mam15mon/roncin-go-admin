import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { appInitialStateContext } from '@/app/initialStateContext';
import FinanceBillsPage from './index';

const state = vi.hoisted(() => ({
  search: undefined as undefined | { visibleNames: string[] },
  hiddenColumns: undefined as string[] | undefined,
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
  FinanceLedgerTemplate: (p: {
    customSearch: React.ReactNode;
    columnSettingsDefaultHiddenKeys: string[];
  }) => {
    state.hiddenColumns = p.columnSettingsDefaultHiddenKeys;
    return p.customSearch;
  },
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

function renderBills() {
  return render(
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
}

describe('账单列与筛选默认项', () => {
  beforeEach(() => window.localStorage.clear());

  it('首屏默认只展示常用筛选与一套状态列', async () => {
    renderBills();
    await act(async () => {});
    expect(state.search?.visibleNames).toEqual([
      'keyword',
      'direction',
      'businessStatuses',
      'dueDateRange',
      'settlementPartyId',
    ]);
    expect(state.hiddenColumns).toEqual(
      expect.arrayContaining([
        'settlementProgress',
        'invoiceProgress',
        'completed',
      ]),
    );
    expect(state.hiddenColumns).not.toContain('status');
  });

  it('采用个人开启的进阶字段，并尊重隐藏的常用字段', async () => {
    window.localStorage.setItem(
      'roncin:filter-settings:v1:finance:bills:u1:o1',
      JSON.stringify({
        visible: ['invoiceProgress'],
        hidden: ['dueDateRange'],
      }),
    );
    renderBills();
    await act(async () => {});
    expect(state.search?.visibleNames).not.toContain('billDateRange');
    expect(state.search?.visibleNames).not.toContain('dueDateRange');
    expect(state.search?.visibleNames).not.toContain('status');
    expect(state.search?.visibleNames).toContain('businessStatuses');
    expect(state.search?.visibleNames).toContain('invoiceProgress');
    window.localStorage.clear();
  });
});
