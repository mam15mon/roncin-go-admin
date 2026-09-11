import { fireEvent, render, screen } from '@testing-library/react';
import { Form } from 'antd';
import React from 'react';
import { describe, expect, it, vi } from 'vitest';
import BillGroupCard from './BillGroupCard';

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListBillSettlementAccountCandidates: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));
vi.mock('@/utils/options', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([]),
}));
vi.mock('@ant-design/pro-components', () => ({
  ProTable: () => <div data-testid="mock-pro-table" />,
}));

describe('BillGroupCard', () => {
  it('散客应收组展示散客Tag，账期>0时展示黄色警示且不阻断', async () => {
    const casualGroup = {
      groupKey: 'casual-1',
      direction: 'RECEIVABLE',
      settlementPartyId: 'party-casual',
      settlementPartyName: '散客货主',
      currency: 'CNY',
      isCasual: true,
      defaultPaymentTermsDays: 0,
      totalAmount: '100.00000000',
      configurationComplete: true,
    };

    function TestWrapper() {
      const [form] = Form.useForm();
      return (
        <Form
          form={form}
          initialValues={{
            groups: {
              'casual-1': {
                paymentTermsDays: 0,
              },
            },
          }}
        >
          <BillGroupCard
            group={casualGroup}
            organizationId="org-1"
            sessionIdentity="session-1"
            feeColumns={[]}
            directionText={(dir) => (dir === 'RECEIVABLE' ? '应收' : '应付')}
            onConfigurationChange={vi.fn()}
          />
        </Form>
      );
    }

    render(<TestWrapper />);

    // 散客Tag渲染
    expect(screen.getByText('散客')).toBeInTheDocument();
    // 账期为0时无警告
    expect(
      screen.queryByText(/该客户为单次合作散客，建议现结/),
    ).not.toBeInTheDocument();

    // 修改账期为 30 天
    const input = screen.getByLabelText('账期（天）');
    fireEvent.change(input, { target: { value: '30' } });

    // 出现黄色预警
    expect(
      await screen.findByText(
        '该客户为单次合作散客，建议现结；当前已设置 30 天账期，请注意资金回款风险',
      ),
    ).toBeInTheDocument();
  });

  it('正式客户不展示散客Tag，账期>0时不显示散客警告', async () => {
    const regularGroup = {
      groupKey: 'reg-1',
      direction: 'RECEIVABLE',
      settlementPartyId: 'party-reg',
      settlementPartyName: '正式客户企业',
      currency: 'CNY',
      isCasual: false,
      totalAmount: '200.00000000',
      configurationComplete: true,
    };

    function TestWrapper() {
      const [form] = Form.useForm();
      return (
        <Form
          form={form}
          initialValues={{
            groups: {
              'reg-1': {
                paymentTermsDays: 30,
              },
            },
          }}
        >
          <BillGroupCard
            group={regularGroup}
            organizationId="org-1"
            sessionIdentity="session-1"
            feeColumns={[]}
            directionText={(dir) => (dir === 'RECEIVABLE' ? '应收' : '应付')}
            onConfigurationChange={vi.fn()}
          />
        </Form>
      );
    }

    render(<TestWrapper />);

    expect(screen.queryByText('散客')).not.toBeInTheDocument();
    expect(
      screen.queryByText(/该客户为单次合作散客，建议现结/),
    ).not.toBeInTheDocument();
  });
});
