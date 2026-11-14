import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import FinanceCashflowsPage from '../cashflows';
import FinanceNettingsPage from '../nettings';
import FinanceVerificationsPage from './index';

type LedgerProps = {
  customSearch?: ReactNode;
  actionRef: { current: unknown };
  request: (params: { current: number; pageSize: number }) => Promise<unknown>;
};
const state = vi.hoisted(() => ({
  ledger: undefined as LedgerProps | undefined,
  reload: vi.fn(),
  list: vi.fn(),
}));
vi.mock('@/features/finance/credit-control', () => ({
  useCreditLimitIntervention: () => false,
  disableCreditExceededOptions: (options: unknown) => options,
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => true }),
}));
vi.mock('@/components/ui', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/components/ui')>()),
  FinanceLedgerTemplate: (props: LedgerProps) => {
    state.ledger = props;
    props.actionRef.current = { reload: state.reload };
    return props.customSearch;
  },
}));
vi.mock('@/services/roncin/settlementService', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/services/roncin/settlementService')
  >()),
  settlementServiceListCashflows: state.list,
  settlementServiceListVerifications: state.list,
  settlementServiceListNettings: state.list,
}));

beforeEach(() => {
  localStorage.clear();
  vi.clearAllMocks();
  state.list.mockResolvedValue({ data: [], total: '0', matchedSummary: {} });
});

describe('收付、核销和对冲筛选设置', () => {
  it.each([
    ['收付', FinanceCashflowsPage, '流水号／结算单位／银行参考号'],
    ['核销', FinanceVerificationsPage, '核销号／结算单位／账单号／流水号'],
    ['对冲', FinanceNettingsPage, '对冲单号／结算单位'],
  ] as const)(
    '%s移除筛选字段会清除已生效的查询并刷新',
    async (_name, Page, placeholder) => {
      renderWithApp(<Page />, {
        user: {
          id: 'filter-user',
          currentOrganization: {
            id: 'company-a',
            kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
          },
        },
      });
      fireEvent.change(screen.getByPlaceholderText(placeholder), {
        target: { value: '待移除条件' },
      });
      fireEvent.click(screen.getByRole('button', { name: /查询/ }));
      await waitFor(() => expect(state.reload).toHaveBeenCalledOnce());
      await act(async () => {
        await state.ledger?.request({ current: 1, pageSize: 40 });
      });
      expect(state.list).toHaveBeenLastCalledWith(
        expect.objectContaining({
          keyword: '待移除条件',
          organizationId: 'company-a',
        }),
        expect.anything(),
      );
      fireEvent.click(screen.getByRole('button', { name: '筛选设置' }));
      fireEvent.click(
        await screen.findByRole('checkbox', { name: '综合搜索' }),
      );
      await waitFor(() =>
        expect(
          screen.queryByPlaceholderText(placeholder),
        ).not.toBeInTheDocument(),
      );
      await act(async () => {
        await state.ledger?.request({ current: 1, pageSize: 40 });
      });
      expect(state.list).toHaveBeenLastCalledWith(
        expect.objectContaining({
          keyword: undefined,
          organizationId: 'company-a',
        }),
        expect.anything(),
      );
      expect(state.reload).toHaveBeenCalledWith(true);
    },
  );
});
