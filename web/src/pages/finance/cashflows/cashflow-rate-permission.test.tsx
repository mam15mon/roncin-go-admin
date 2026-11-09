import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen } from '@testing-library/react';
import { App } from 'antd';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import getAccess from '@/access';
import { AuthOrganizationKind } from '@/enums.generated';
import FinanceCashflowsPage from './index';

const state = vi.hoisted(() => ({ permissions: [] as string[] }));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: { id: 'user', currentOrganization: { id: 'workspace-org' } },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () =>
    getAccess({
      currentUser: {
        id: 'user',
        currentOrganization: {
          id: 'workspace-org',
          kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
        },
        permissions: state.permissions,
        permissionCapabilities: state.permissions.map((key) => ({
          key,
          dataScope: 'organization',
        })),
        roleScopes: [{ roleCode: 'finance' }],
      } as API.CurrentUser,
    }),
}));
vi.mock('@/components/ui', () => ({
  GuardedModalForm: ({
    open,
    children,
  }: {
    open: boolean;
    children: ReactNode;
  }) => (open ? <div>{children}</div> : null),
  FinanceLedgerTemplate: ({
    onPrimaryAction,
  }: {
    onPrimaryAction?: () => void;
  }) => (
    <button type="button" onClick={onPrimaryAction}>
      登记流水
    </button>
  ),
  ProFormSearchableSelect: () => null,
}));
vi.mock('@/features/finance/credit-control', () => ({
  useCreditLimitIntervention: () => false,
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([]),
}));
vi.mock('@ant-design/pro-components', () => ({
  ProFormDatePicker: () => null,
  ProFormText: ({ label }: { label: string }) => <input aria-label={label} />,
  ProFormTextArea: () => null,
}));
vi.mock('@/services/roncin/settlementService', () => ({}));

describe('流水手工汇率入口权限', () => {
  it.each([
    ['汇率维护权限', ['system.finance.exchange_rate.update'], true],
    [
      '仅订单费用汇率权限',
      ['system.finance.order_fee.exchange_rate.override'],
      false,
    ],
    ['没有汇率权限', [], false],
  ] as const)('%s按服务端权限显示入口', async (_, permissions, visible) => {
    state.permissions = ['system.finance.cashflow.create', ...permissions];
    renderWithClient(
      <App>
        <FinanceCashflowsPage />
      </App>,
    );
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: '登记流水' }));
    });
    if (visible) {
      expect(screen.getByLabelText('结算汇率（可选）')).toBeVisible();
    } else {
      expect(
        screen.queryByLabelText('结算汇率（可选）'),
      ).not.toBeInTheDocument();
    }
  });
});
