import { renderWithClient } from '@root/tests/queryClientTestUtils';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';

const serviceMocks = vi.hoisted(() => ({
  createVerification: vi.fn(),
  organizationId: 'organization-a' as string | undefined,
  canOperate: true,
  listFinanceOrganizationOptions: vi.fn(),
  listFinanceSettlementPartyOptions: vi.fn(),
  listVerificationCreationCandidates: vi.fn(),
  getCreditLimitControlPolicy: vi.fn(),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user',
        currentOrganization: { id: serviceMocks.organizationId },
      },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canCreateFinanceVerifications: serviceMocks.canOperate,
    canOperateOrganization: (id?: string) =>
      serviceMocks.canOperate &&
      Boolean(id) &&
      id === serviceMocks.organizationId,
  }),
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceCreateVerification: serviceMocks.createVerification,
  settlementServiceListFinanceOrganizationOptions:
    serviceMocks.listFinanceOrganizationOptions,
  settlementServiceListFinanceSettlementPartyOptions:
    serviceMocks.listFinanceSettlementPartyOptions,
  settlementServiceListVerificationCreationCandidates:
    serviceMocks.listVerificationCreationCandidates,
  // 默认仅提醒模式（超额后仍允许选择），不触发超额禁用；干预模式用例单独覆写。
  settlementServiceGetCreditLimitControlPolicy:
    serviceMocks.getCreditLimitControlPolicy,
}));

vi.mock('antd', async () => {
  const React = await import('react');
  const passthrough = ({ children }: { children?: React.ReactNode }) =>
    React.createElement(React.Fragment, null, children);
  const message = {
    error: vi.fn(),
    success: vi.fn(),
    warning: vi.fn(),
  };
  const App = Object.assign(passthrough, {
    useApp: () => ({ message }),
  });
  const Select = ({
    options = [],
    value,
    onChange,
    disabled,
    'aria-label': ariaLabel,
  }: any) =>
    React.createElement(
      'select',
      {
        'aria-label': ariaLabel,
        value: value ?? '',
        disabled,
        onChange: (event: React.ChangeEvent<HTMLSelectElement>) =>
          onChange?.(event.target.value),
      },
      React.createElement('option', { value: '' }),
      ...options.map(
        (option: { label: string; value: string; disabled?: boolean }) =>
          React.createElement(
            'option',
            {
              key: option.value,
              value: option.value,
              disabled: option.disabled,
            },
            option.label,
          ),
      ),
    );
  const Table = ({ dataSource = [], columns = [], rowSelection }: any) =>
    React.createElement(
      'table',
      null,
      React.createElement(
        'tbody',
        null,
        ...dataSource.map((record: Record<string, unknown>, index: number) =>
          React.createElement(
            'tr',
            { key: String(record.id ?? index) },
            rowSelection
              ? React.createElement(
                  'td',
                  null,
                  React.createElement('input', {
                    type: 'checkbox',
                    checked: rowSelection.selectedRowKeys.includes(record.id),
                    onChange: () => {
                      const selected = new Set(rowSelection.selectedRowKeys);
                      if (selected.has(record.id)) selected.delete(record.id);
                      else selected.add(record.id);
                      rowSelection.onChange([...selected]);
                    },
                  }),
                )
              : null,
            ...columns.map((column: any) =>
              React.createElement(
                'td',
                { key: column.dataIndex ?? column.title },
                column.render
                  ? column.render(record[column.dataIndex], record, index)
                  : String(record[column.dataIndex] ?? ''),
              ),
            ),
          ),
        ),
      ),
    );
  const Input = ({ suffix: _suffix, ...props }: any) =>
    React.createElement('input', props);
  Input.TextArea = (props: any) => React.createElement('textarea', props);
  return {
    Alert: passthrough,
    App,
    Button: ({
      children,
      icon: _icon,
      loading: _loading,
      type: _type,
      ...props
    }: any) => React.createElement('button', props, children),
    Card: ({ children, extra }: any) =>
      React.createElement(React.Fragment, null, extra, children),
    Col: passthrough,
    ConfigProvider: passthrough,
    DatePicker: (props: any) => React.createElement('input', props),
    Descriptions: passthrough,
    Empty: ({ description }: { description?: React.ReactNode }) =>
      React.createElement('div', null, description),
    Input,
    Modal: ({
      children,
      footer,
      open,
    }: {
      children?: React.ReactNode;
      footer?: React.ReactNode;
      open?: boolean;
    }) =>
      open ? React.createElement(React.Fragment, null, children, footer) : null,
    Row: passthrough,
    Select,
    Space: passthrough,
    Table,
    Tag: passthrough,
    Tooltip: passthrough,
    Typography: { Text: passthrough },
    // 统一列设置浮层内容在组件渲染期即会取用以下导出（浮层关闭时不实际展示）。
    Checkbox: passthrough,
    Radio: Object.assign(passthrough, { Group: passthrough }),
    Tabs: () => null,
    Popover: () => null,
  };
});

vi.mock('@/features/master-data/currencies', async (importOriginal) => {
  const actual =
    await importOriginal<typeof import('@/features/master-data/currencies')>();
  return {
    ...actual,
    getCurrencyOptions: vi
      .fn()
      .mockResolvedValue([{ label: '美元', value: 'USD' }]),
  };
});

import VerificationWorkbench from './VerificationWorkbench';

describe('核销创建工作台组织候选', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', '/finance/verifications');
    serviceMocks.organizationId = 'organization-a';
    serviceMocks.canOperate = true;
    serviceMocks.createVerification.mockReset();
    serviceMocks.listFinanceOrganizationOptions.mockReset();
    serviceMocks.listFinanceSettlementPartyOptions.mockReset();
    serviceMocks.listVerificationCreationCandidates.mockReset();
    serviceMocks.getCreditLimitControlPolicy.mockReset();
    // 默认「超额后允许选择」为 true：true 会正常序列化，仅提醒模式生效。
    serviceMocks.getCreditLimitControlPolicy.mockResolvedValue({
      success: true,
      data: { allowSelectionWhenCreditExceeded: true },
    });
    serviceMocks.listFinanceOrganizationOptions.mockResolvedValue({
      data: [
        { id: 'organization-a', code: 'A', name: '公司 A' },
        { id: 'organization-b', code: 'B', name: '公司 B' },
      ],
    });
    serviceMocks.listFinanceSettlementPartyOptions.mockImplementation(
      ({ organizationId }) =>
        Promise.resolve({
          data: [
            {
              id: `party-${organizationId}`,
              code: 'PARTY',
              name: `结算单位 ${organizationId}`,
            },
          ],
        }),
    );
    serviceMocks.listVerificationCreationCandidates.mockResolvedValue({
      data: {
        cashflows: [
          {
            id: 'cashflow-a',
            flowNo: 'FLOW-A',
            unverifiedAmount: '10',
            organizationId: 'organization-a',
          },
        ],
        bills: [
          {
            id: 'bill-a',
            billNo: 'BILL-A',
            unverifiedAmount: '10',
            organizationId: 'organization-a',
          },
        ],
      },
    });
    serviceMocks.createVerification.mockResolvedValue({
      data: { id: 'verification-new' },
    });
  });

  const selectOrganizationAndParty = async (organizationName: string) => {
    await waitFor(() =>
      expect(
        serviceMocks.listFinanceSettlementPartyOptions,
      ).toHaveBeenCalledWith(
        expect.objectContaining({
          organizationId:
            organizationName === '公司 A' ? 'organization-a' : 'organization-b',
          purpose: 9,
        }),
      ),
    );
    await screen.findByRole('option', {
      name: `结算单位 ${organizationName === '公司 A' ? 'organization-a' : 'organization-b'} (PARTY)`,
    });
    await act(async () => {
      fireEvent.change(screen.getByLabelText('结算单位'), {
        target: {
          value: `party-${organizationName === '公司 A' ? 'organization-a' : 'organization-b'}`,
        },
      });
      await Promise.resolve();
    });
  };

  it('默认当前公司请求候选并能自动分配提交', async () => {
    const onCreated = vi.fn();
    renderWithClient(
      <App>
        <VerificationWorkbench open onClose={vi.fn()} onCreated={onCreated} />
      </App>,
    );

    expect(serviceMocks.listFinanceOrganizationOptions).not.toHaveBeenCalled();
    expect(screen.queryByLabelText('所属公司')).not.toBeInTheDocument();

    await selectOrganizationAndParty('公司 A');

    await waitFor(() =>
      expect(
        serviceMocks.listVerificationCreationCandidates,
      ).toHaveBeenCalledWith(
        expect.objectContaining({
          organizationId: 'organization-a',
          direction: 'RECEIVABLE',
          settlementPartyId: 'party-organization-a',
          currency: 'CNY',
        }),
      ),
    );
    expect(await screen.findByText('FLOW-A')).toBeInTheDocument();
    expect(screen.getByText('BILL-A')).toBeInTheDocument();

    expect(getAnyDirtyTabKey()).toBeUndefined();
    const cashflowRow = screen.getByText('FLOW-A').closest('tr');
    const billRow = screen.getByText('BILL-A').closest('tr');
    if (!cashflowRow || !billRow) throw new Error('候选表格行不存在');
    fireEvent.click(within(cashflowRow).getByRole('checkbox'));
    fireEvent.click(within(billRow).getByRole('checkbox'));
    await waitFor(() => {
      expect(within(cashflowRow).getByRole('checkbox')).toBeChecked();
      expect(within(billRow).getByRole('checkbox')).toBeChecked();
    });
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.click(screen.getByRole('button', { name: /按余额自动分配/ }));
    expect(getAnyDirtyTabKey()).toBe('/finance/verifications');
    fireEvent.change(screen.getByLabelText('第 1 行核销金额'), {
      target: { value: '9' },
    });
    fireEvent.change(screen.getByLabelText('第 1 行核销金额'), {
      target: { value: '10.00000000' },
    });
    expect(getAnyDirtyTabKey()).toBe('/finance/verifications');
    expect(await screen.findByLabelText('第 1 行核销金额')).toHaveValue(
      '10.00000000',
    );

    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /提交核销/ }));
      await Promise.resolve();
    });
    await waitFor(() =>
      expect(serviceMocks.createVerification).toHaveBeenCalledWith(
        expect.objectContaining({
          allocations: [
            {
              cashflowId: 'cashflow-a',
              billId: 'bill-a',
              amount: '10.00000000',
            },
          ],
        }),
      ),
    );
    await waitFor(() => expect(onCreated).toHaveBeenCalledOnce());
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.change(screen.getByLabelText('第 1 行核销金额'), {
      target: { value: '9' },
    });
    expect(getAnyDirtyTabKey()).toBe('/finance/verifications');
  });

  it('切换工作区重建后只请求新公司结算单位并清空选择', async () => {
    const view = renderWithClient(
      <App>
        <VerificationWorkbench open onClose={vi.fn()} onCreated={vi.fn()} />
      </App>,
    );
    await selectOrganizationAndParty('公司 A');
    expect(await screen.findByText('FLOW-A')).toBeInTheDocument();
    view.unmount();
    serviceMocks.organizationId = 'organization-b';
    renderWithClient(
      <App>
        <VerificationWorkbench open onClose={vi.fn()} onCreated={vi.fn()} />
      </App>,
    );
    expect(screen.queryByText('FLOW-A')).not.toBeInTheDocument();
    await waitFor(() =>
      expect(
        serviceMocks.listFinanceSettlementPartyOptions,
      ).toHaveBeenLastCalledWith(
        expect.objectContaining({
          organizationId: 'organization-b',
          purpose: 9,
        }),
      ),
    );
  });

  it.each([undefined, 'system'])(
    '身份%s不在公司工作区时不请求经营候选',
    async (id) => {
      serviceMocks.organizationId = id;
      serviceMocks.canOperate = false;
      renderWithClient(
        <App>
          <VerificationWorkbench open onClose={vi.fn()} onCreated={vi.fn()} />
        </App>,
      );
      expect(
        serviceMocks.listFinanceSettlementPartyOptions,
      ).not.toHaveBeenCalled();
      expect(
        serviceMocks.listVerificationCreationCandidates,
      ).not.toHaveBeenCalled();
      expect(serviceMocks.createVerification).not.toHaveBeenCalled();
    },
  );

  it('直接干预模式下超额客户置灰禁用，仅提醒模式可选', async () => {
    // 服务端省略 false 布尔：直接以「缺省 = 干预模式」覆写策略响应。
    let resolvePolicy: ((response: unknown) => void) | undefined;
    serviceMocks.getCreditLimitControlPolicy.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolvePolicy = resolve;
        }),
    );
    serviceMocks.listFinanceSettlementPartyOptions.mockImplementation(
      ({ organizationId }) =>
        Promise.resolve({
          data: [
            {
              id: `party-${organizationId}`,
              code: 'PARTY',
              name: `结算单位 ${organizationId}`,
            },
            {
              id: 'party-exceeded',
              code: 'EXCEED',
              name: '超额客户',
              creditExceeded: true,
            },
          ],
        }),
    );

    renderWithClient(
      <App>
        <VerificationWorkbench open onClose={vi.fn()} onCreated={vi.fn()} />
      </App>,
    );
    await selectOrganizationAndParty('公司 A');

    const exceededOption = await screen.findByRole('option', {
      name: '超额客户 (EXCEED)',
    });
    expect(exceededOption).toBeEnabled();
    await act(async () => resolvePolicy?.({ success: true, data: {} }));
    // 直接干预模式：策略迟到仍更新超额候选禁用，label 保持纯净文本。
    await waitFor(() => expect(exceededOption).toBeDisabled());
    const normalOption = screen.getByRole('option', {
      name: '结算单位 organization-a (PARTY)',
    });
    expect(normalOption).toBeEnabled();
  });
  it.each(['cashflows', 'bills'])(
    '候选混入外公司%s时拒绝核销分配和提交',
    async (source) => {
      serviceMocks.listVerificationCreationCandidates.mockResolvedValue({
        data: {
          cashflows: [
            {
              id: 'cashflow-a',
              flowNo: 'FLOW-A',
              unverifiedAmount: '10',
              organizationId:
                source === 'cashflows' ? 'foreign' : 'organization-a',
            },
          ],
          bills: [
            {
              id: 'bill-a',
              billNo: 'BILL-A',
              unverifiedAmount: '10',
              organizationId: source === 'bills' ? 'foreign' : 'organization-a',
            },
          ],
        },
      });
      renderWithClient(
        <App>
          <VerificationWorkbench open onClose={vi.fn()} onCreated={vi.fn()} />
        </App>,
      );
      await selectOrganizationAndParty('公司 A');
      const cashflowRow = (await screen.findByText('FLOW-A')).closest('tr');
      const billRow = screen.getByText('BILL-A').closest('tr');
      if (!cashflowRow || !billRow) throw new Error('候选行不存在');
      fireEvent.click(within(cashflowRow).getByRole('checkbox'));
      fireEvent.click(within(billRow).getByRole('checkbox'));
      fireEvent.click(screen.getByRole('button', { name: /按余额自动分配/ }));
      expect(
        screen.queryByLabelText('第 1 行核销金额'),
      ).not.toBeInTheDocument();
      expect(serviceMocks.createVerification).not.toHaveBeenCalled();
    },
  );

  it('纯 state 备注输入会保护，关闭后清理', async () => {
    const view = renderWithClient(
      <App>
        <VerificationWorkbench open onClose={vi.fn()} onCreated={vi.fn()} />
      </App>,
    );
    expect(getAnyDirtyTabKey()).toBeUndefined();
    fireEvent.change(screen.getByLabelText('核销备注'), {
      target: { value: '待保存备注' },
    });
    expect(getAnyDirtyTabKey()).toBe('/finance/verifications');
    view.unmount();
    expect(getAnyDirtyTabKey()).toBeUndefined();
  });
});
