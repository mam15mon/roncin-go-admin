import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';

const serviceMocks = vi.hoisted(() => ({
  createVerification: vi.fn(),
  listFinanceOrganizationOptions: vi.fn(),
  listFinanceSettlementPartyOptions: vi.fn(),
  listVerificationCreationCandidates: vi.fn(),
  getCreditLimitControlPolicy: vi.fn(),
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
    DatePicker: (props: any) => React.createElement('input', props),
    Descriptions: passthrough,
    Empty: ({ description }: { description?: React.ReactNode }) =>
      React.createElement('div', null, description),
    Input,
    Modal: ({
      children,
      footer,
    }: {
      children?: React.ReactNode;
      footer?: React.ReactNode;
    }) => React.createElement(React.Fragment, null, children, footer),
    Row: passthrough,
    Select,
    Space: passthrough,
    Table,
    Tag: passthrough,
    Typography: { Text: passthrough },
  };
});

vi.mock('@/utils/options', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@/utils/options')>();
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
          { id: 'cashflow-a', flowNo: 'FLOW-A', unverifiedAmount: '10' },
        ],
        bills: [{ id: 'bill-a', billNo: 'BILL-A', unverifiedAmount: '10' }],
      },
    });
    serviceMocks.createVerification.mockResolvedValue({ data: {} });
  });

  const selectOrganizationAndParty = async (organizationName: string) => {
    await screen.findByRole('option', { name: organizationName });
    await act(async () => {
      fireEvent.change(screen.getByLabelText('所属公司'), {
        target: {
          value:
            organizationName === '公司 A' ? 'organization-a' : 'organization-b',
        },
      });
      await Promise.resolve();
    });
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
    await act(async () => {
      fireEvent.change(screen.getByLabelText('结算单位'), {
        target: {
          value: `party-${organizationName === '公司 A' ? 'organization-a' : 'organization-b'}`,
        },
      });
      await Promise.resolve();
    });
  };

  it('先选择核销创建可写公司，再以该组织请求候选并能自动分配提交', async () => {
    const onCreated = vi.fn();
    render(
      <App>
        <VerificationWorkbench open onClose={vi.fn()} onCreated={onCreated} />
      </App>,
    );

    await waitFor(() =>
      expect(serviceMocks.listFinanceOrganizationOptions).toHaveBeenCalledWith({
        purpose: 9,
      }),
    );
    expect(
      serviceMocks.listVerificationCreationCandidates,
    ).not.toHaveBeenCalled();

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

    const cashflowRow = screen.getByText('FLOW-A').closest('tr');
    const billRow = screen.getByText('BILL-A').closest('tr');
    if (!cashflowRow || !billRow) throw new Error('候选表格行不存在');
    fireEvent.click(within(cashflowRow).getByRole('checkbox'));
    fireEvent.click(within(billRow).getByRole('checkbox'));
    await waitFor(() => {
      expect(within(cashflowRow).getByRole('checkbox')).toBeChecked();
      expect(within(billRow).getByRole('checkbox')).toBeChecked();
    });
    fireEvent.click(screen.getByRole('button', { name: /按余额自动分配/ }));
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
  });

  it('切换公司清空上一组织候选与选择，并只向新组织请求结算单位', async () => {
    render(
      <App>
        <VerificationWorkbench open onClose={vi.fn()} onCreated={vi.fn()} />
      </App>,
    );
    await selectOrganizationAndParty('公司 A');
    expect(await screen.findByText('FLOW-A')).toBeInTheDocument();

    const cashflowRow = screen.getByText('FLOW-A').closest('tr');
    if (!cashflowRow) throw new Error('资金流水候选行不存在');
    fireEvent.click(within(cashflowRow).getByRole('checkbox'));

    await act(async () => {
      fireEvent.change(screen.getByLabelText('所属公司'), {
        target: { value: 'organization-b' },
      });
      await Promise.resolve();
    });

    await waitFor(() => {
      expect(screen.queryByText('FLOW-A')).not.toBeInTheDocument();
      expect(screen.queryByText('BILL-A')).not.toBeInTheDocument();
    });
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

  it('直接干预模式下超额客户置灰禁用，仅提醒模式可选', async () => {
    // 服务端省略 false 布尔：直接以「缺省 = 干预模式」覆写策略响应。
    serviceMocks.getCreditLimitControlPolicy.mockResolvedValue({
      success: true,
      data: {},
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
            {
              id: 'party-exceeded',
              code: 'EXCEED',
              name: '超额客户',
              creditExceeded: true,
            },
          ],
        }),
    );

    render(
      <App>
        <VerificationWorkbench open onClose={vi.fn()} onCreated={vi.fn()} />
      </App>,
    );
    await selectOrganizationAndParty('公司 A');

    const exceededOption = await screen.findByRole('option', {
      name: '超额客户 (EXCEED)',
    });
    // 直接干预模式：超额候选置灰禁用，无法被选择；label 保持纯净文本。
    expect(exceededOption).toBeDisabled();
    const normalOption = screen.getByRole('option', {
      name: '结算单位 organization-a (PARTY)',
    });
    expect(normalOption).toBeEnabled();
  });
});
