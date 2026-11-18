import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import type { MutableRefObject, ReactNode } from 'react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import FinanceInvoicesPage from './index';

type LedgerProps = {
  actionRef: MutableRefObject<
    { reload: (reset?: boolean) => void } | undefined
  >;
  customSearch: ReactNode;
  request: (params: {
    current: number;
    pageSize: number;
    status?: string;
  }) => Promise<unknown>;
  selectionScope: string;
};
const mocks = vi.hoisted(() => ({
  reload: vi.fn(),
  list: vi.fn(),
  ledger: undefined as LedgerProps | undefined,
}));
vi.mock('@/components/ui', async (original) => ({
  ...(await original<typeof import('@/components/ui')>()),
  FinanceLedgerTemplate: (props: LedgerProps) => {
    mocks.ledger = props;
    props.actionRef.current = { reload: mocks.reload };
    return props.customSearch;
  },
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListInvoices: mocks.list,
}));
vi.mock('@/features/finance/invoice-registration', () => ({
  InvoiceRegistrationModal: () => null,
}));
vi.mock('./components/InvoiceDetailModal', () => ({ default: () => null }));
vi.mock('./components/InvoiceCancelModal', () => ({ default: () => null }));
vi.mock('./components/InvoiceIssueAndRedFlushModals', () => ({
  InvoiceRedFlushModal: () => null,
}));

const user: API.CurrentUser = {
  id: 'invoice-user',
  currentOrganization: {
    id: 'company-a',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
};
const storageKey =
  'roncin:filter-settings:v1:finance:invoices:invoice-user:company-a';

beforeEach(() => {
  vi.clearAllMocks();
  window.localStorage.clear();
  mocks.list.mockResolvedValue({ data: [], total: '0', matchedSummary: {} });
});
afterEach(() => window.localStorage.clear());

describe('发票列表筛选设置', () => {
  it('移除已生效筛选清值、回首页刷新并变更选择范围，其他条件继续生效', async () => {
    renderWithApp(<FinanceInvoicesPage />, { user });
    fireEvent.change(
      screen.getByPlaceholderText('记录编号、发票号或结算单位'),
      { target: { value: '测试单位' } },
    );
    fireEvent.mouseDown(screen.getByLabelText('发票状态'));
    fireEvent.click(await screen.findByText('草稿'));
    fireEvent.click(screen.getByRole('button', { name: /查\s*询/ }));
    await waitFor(() => expect(mocks.reload).toHaveBeenCalledWith(true));
    await act(async () => {
      await mocks.ledger?.request({ current: 1, pageSize: 40 });
    });
    expect(mocks.list).toHaveBeenLastCalledWith(
      expect.objectContaining({
        keyword: '测试单位',
        status: 1,
        organizationId: 'company-a',
      }),
      expect.anything(),
    );
    const previousScope = mocks.ledger?.selectionScope;
    fireEvent.click(screen.getByRole('button', { name: '筛选设置' }));
    fireEvent.click(await screen.findByRole('checkbox', { name: '发票状态' }));
    await waitFor(() =>
      expect(
        screen.queryByRole('combobox', { name: '发票状态' }),
      ).not.toBeInTheDocument(),
    );
    await act(async () => {
      await mocks.ledger?.request({ current: 1, pageSize: 40, status: '1' });
    });
    expect(mocks.list).toHaveBeenLastCalledWith(
      expect.objectContaining({
        keyword: '测试单位',
        status: undefined,
        organizationId: 'company-a',
      }),
      expect.anything(),
    );
    expect(mocks.reload).toHaveBeenLastCalledWith(true);
    expect(mocks.ledger?.selectionScope).not.toBe(previousScope);
    const preference = window.localStorage.getItem(storageKey);
    expect(preference).toContain('status');
    expect(preference).not.toContain('测试单位');
  });

  it('同用户公司重开恢复字段偏好，不把公司范围列为可删除筛选', async () => {
    window.localStorage.setItem(
      storageKey,
      JSON.stringify({ visible: [], hidden: ['direction'] }),
    );
    renderWithApp(<FinanceInvoicesPage />, { user });
    expect(screen.queryByLabelText('发票方向')).not.toBeInTheDocument();
    expect(screen.getByLabelText('发票状态')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('button', { name: '筛选设置' }));
    expect(
      await screen.findByRole('checkbox', { name: '发票方向' }),
    ).not.toBeChecked();
    expect(
      screen.queryByRole('checkbox', { name: /公司/ }),
    ).not.toBeInTheDocument();
  });
});
