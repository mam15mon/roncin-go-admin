import { createTestQueryClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, ConfigProvider, Form } from 'antd';
import React, { useState } from 'react';
import { afterEach, beforeEach, expect, it, vi } from 'vitest';
import BillEditModal from './BillEditModal';
import type { BillFormValues } from './billConstants';

const api = vi.hoisted(() => ({ getBill: vi.fn(), refresh: vi.fn() }));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canCreatePartnerAccounts: false,
    canOperateOrganization: () => false,
  }),
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceGetBill: api.getBill,
  settlementServiceListBillSettlementAccountUpdateCandidates: vi
    .fn()
    .mockResolvedValue({ success: true, data: [] }),
}));
vi.mock('@/features/master-data/currencies', () => ({
  currencyOptionsQueryOptions: () => ({
    queryKey: ['test-currencies'],
    queryFn: async () => [],
  }),
}));

function Harness() {
  const [form] = Form.useForm<BillFormValues>();
  const [version, setVersion] = useState('3');
  const [submitting, setSubmitting] = useState(false);
  return (
    <>
      <BillEditModal
        open
        editing={{ id: 'bill-1', version, billNo: 'BILL-1' }}
        form={form}
        submitting={submitting}
        onCancel={vi.fn()}
        onOk={async () => {}}
        onRefresh={async () => {
          const accepted = await api.refresh();
          if (accepted) {
            form.setFieldsValue({ statementTitle: '服务端新抬头' });
            setVersion('4');
          }
          return accepted;
        }}
      />
      <button type="button" onClick={() => setSubmitting(true)}>
        开始本人保存
      </button>
      <button
        type="button"
        onClick={() => {
          setVersion('4');
          setSubmitting(false);
        }}
      >
        采纳本人保存
      </button>
    </>
  );
}

function mount() {
  const client = createTestQueryClient();
  const view = render(
    <QueryClientProvider client={client}>
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <Harness />
        </App>
      </ConfigProvider>
    </QueryClientProvider>,
  );
  return { ...view, client };
}
async function detectChange(client: ReturnType<typeof createTestQueryClient>) {
  await act(async () => {
    client.setQueryData(
      ['finance', 'bill-change-probe', { billId: 'bill-1' }],
      '4',
    );
  });
  await screen.findByText('这个账单已经发生变动，请重新刷新。');
}

beforeEach(() => {
  window.history.replaceState({}, '', '/finance/bills');
  vi.clearAllMocks();
  vi.spyOn(document, 'hasFocus').mockReturnValue(true);
  vi.spyOn(document, 'visibilityState', 'get').mockReturnValue('visible');
  api.getBill.mockResolvedValue({
    success: true,
    data: { id: 'bill-1', version: '3' },
  });
  api.refresh.mockResolvedValue(true);
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});

it('外部变化只提示，取消刷新保留输入和原基线', async () => {
  const { client } = mount();
  await waitFor(() => expect(api.getBill).toHaveBeenCalledOnce());
  const title = screen.getByLabelText('对账抬头');
  fireEvent.change(title, { target: { value: '人工新输入' } });
  await detectChange(client);
  expect(title).toHaveValue('人工新输入');
  fireEvent.click(screen.getByRole('button', { name: '刷新账单' }));
  fireEvent.click(await screen.findByRole('button', { name: '继续编辑' }));
  await waitFor(() =>
    expect(screen.getByRole('button', { name: '刷新账单' })).not.toHaveClass(
      'ant-btn-loading',
    ),
  );
  expect(api.refresh).not.toHaveBeenCalled();
  expect(title).toHaveValue('人工新输入');
  expect(
    screen.getByText('这个账单已经发生变动，请重新刷新。'),
  ).toBeInTheDocument();
});

it.each([false, true])(
  '确认刷新结果为%s，只有成功才接受完整快照',
  async (accepted) => {
    api.refresh.mockResolvedValue(accepted);
    const { client } = mount();
    await waitFor(() => expect(api.getBill).toHaveBeenCalledOnce());
    const title = screen.getByLabelText('对账抬头');
    fireEvent.change(title, { target: { value: '人工新输入' } });
    await detectChange(client);
    fireEvent.click(screen.getByRole('button', { name: '刷新账单' }));
    fireEvent.click(await screen.findByRole('button', { name: '放弃并刷新' }));
    await waitFor(() => expect(api.refresh).toHaveBeenCalledOnce());
    await waitFor(() =>
      expect(title).toHaveValue(accepted ? '服务端新抬头' : '人工新输入'),
    );
    if (accepted)
      expect(
        screen.queryByText('这个账单已经发生变动，请重新刷新。'),
      ).not.toBeInTheDocument();
    else
      expect(
        screen.getByText('这个账单已经发生变动，请重新刷新。'),
      ).toBeInTheDocument();
  },
);

it('本人写入在途与采纳响应后均不误报外部变化', async () => {
  const { client } = mount();
  await waitFor(() => expect(api.getBill).toHaveBeenCalledOnce());
  fireEvent.click(screen.getByRole('button', { name: '开始本人保存' }));
  await act(async () => {
    client.setQueryData(
      ['finance', 'bill-change-probe', { billId: 'bill-1' }],
      '4',
    );
  });
  expect(
    screen.queryByText('这个账单已经发生变动，请重新刷新。'),
  ).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: '采纳本人保存' }));
  await act(async () => {});
  expect(
    screen.queryByText('这个账单已经发生变动，请重新刷新。'),
  ).not.toBeInTheDocument();
});
