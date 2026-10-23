import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import type { AxiosAdapter } from 'axios';
import { beforeEach, expect, it, vi } from 'vitest';
import { AsyncPopconfirm } from '@/components/ui';
import * as settlementService from '@/services/roncin/settlementService';
import FinanceBillsPage from './index';

const { transport, notify, reload, confirmOpenChange } = vi.hoisted(() => ({
  transport: vi.fn<AxiosAdapter>(),
  notify: vi.fn(),
  reload: vi.fn(),
  confirmOpenChange: vi.fn(),
}));
vi.mock('axios', async (original) => {
  const actual = await original<typeof import('axios')>();
  return {
    ...actual,
    default: {
      ...actual.default,
      create: (config: Parameters<typeof actual.default.create>[0]) =>
        actual.default.create({ ...config, adapter: transport }),
    },
  };
});
vi.mock('@sentry/react', () => ({ captureException: vi.fn() }));
vi.mock('@/utils/appFeedback', () => ({
  showErrorMessage: notify,
  showErrorNotification: notify,
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => true }),
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencyOptions: vi.fn().mockResolvedValue([]),
}));
vi.mock('./components/billColumns', () => ({
  getFinanceBillColumns: (props: {
    onCancelBill: (bill: API.FinanceBill) => void;
    onConfirmBill: (bill: API.FinanceBill) => Promise<unknown>;
  }) => [
    {
      valueType: 'option',
      render: () => (
        <>
          <button
            type="button"
            onClick={() => props.onCancelBill({ id: 'n-1', version: '2' })}
          >
            取消
          </button>
          <AsyncPopconfirm
            title="确认真实账单？"
            okText="提交确认"
            onOpenChange={confirmOpenChange}
            onConfirm={() => props.onConfirmBill({ id: 'n-1', version: '2' })}
          >
            <button type="button">确认账单</button>
          </AsyncPopconfirm>
        </>
      ),
    },
  ],
}));
vi.mock('@/components/ui', async (original) => {
  const actual = await original<typeof import('@/components/ui')>();
  return {
    ...actual,
    SearchFilterTemplate: () => null,
    FinanceLedgerTemplate: (props: {
      actionRef: { current: unknown };
      columns: { valueType?: string; render?: () => React.ReactNode }[];
    }) => {
      props.actionRef.current = { reload };
      return (
        <>
          {props.columns
            .find((item) => item.valueType === 'option')
            ?.render?.()}
        </>
      );
    },
  };
});
beforeEach(() => vi.clearAllMocks());
it('账单取消真实409后保留原因且不刷新；缺ID不能完成，重试成功才关闭', async () => {
  let attempts = 0;
  transport.mockImplementation(async (config) => {
    if (config.method !== 'get') {
      attempts++;
      if (attempts === 1)
        throw {
          config,
          response: { status: 409, data: { message: '账单版本冲突' } },
        };
      return {
        config,
        status: 200,
        statusText: 'OK',
        headers: {},
        data: { success: true, data: attempts === 2 ? {} : { id: 'n-1' } },
      };
    }
    return {
      config,
      status: 200,
      statusText: 'OK',
      headers: {},
      data: {
        success: true,
        data: config.url?.endsWith('/bills/n-1')
          ? {
              id: 'n-1',
              version: '3',
              billNo: 'B-001',
              organizationName: '公司甲',
              lines: [
                { orderId: 'order-a', orderNo: 'SE-A', active: true },
                { orderId: 'order-a', orderNo: 'SE-A', active: true },
                { orderId: 'order-b', orderNo: 'SE-B', active: true },
              ],
            }
          : [],
      },
    };
  });
  renderWithApp(<FinanceBillsPage />);
  fireEvent.click(screen.getByText('取消'));
  const reason = await screen.findByPlaceholderText('请输入取消原因（必填）');
  expect(screen.getByRole('dialog')).toHaveTextContent(
    '涉及 2 个订单的 3 笔费用（SE-A、SE-B）',
  );
  expect(screen.getByRole('dialog')).toHaveTextContent(
    '合并账单内所有关联费用会一并释放',
  );
  fireEvent.change(reason, { target: { value: '保留取消原因' } });
  for (let n = 1; n <= 2; n++) {
    fireEvent.click(screen.getByRole('button', { name: /确\s*认$/ }));
    await waitFor(() => expect(attempts).toBe(n));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /确\s*认$/ })).not.toHaveClass(
        'ant-btn-loading',
      ),
    );
    expect(reason).toHaveValue('保留取消原因');
    expect(reload).not.toHaveBeenCalled();
    expect(
      screen.queryByText('账单已取消，关联明细费用已释放并可重新建单'),
    ).not.toBeInTheDocument();
  }
  fireEvent.click(screen.getByRole('button', { name: /确\s*认$/ }));
  await waitFor(() => expect(reload).toHaveBeenCalledOnce());
  await waitFor(() =>
    expect(document.querySelector('.ant-modal')).toHaveClass('ant-zoom-leave'),
  );
  expect(notify).toHaveBeenCalledOnce();
  const cancelRequests = transport.mock.calls
    .map(([config]) => config)
    .filter((config) => config.method !== 'get');
  expect(
    cancelRequests.every(
      (config) => JSON.parse(config.data).expectedVersion === '3',
    ),
  ).toBe(true);
});

it('未完整读取账单影响范围时不打开取消确认，也不发起取消', async () => {
  transport.mockImplementation(async (config) => ({
    config,
    status: 200,
    statusText: 'OK',
    headers: {},
    data: { success: true, data: [] },
  }));
  renderWithApp(<FinanceBillsPage />);
  fireEvent.click(screen.getByText('取消'));
  expect(
    await screen.findByText('账单明细未完整加载，请刷新后重试'),
  ).toBeInTheDocument();
  expect(
    screen.queryByPlaceholderText('请输入取消原因（必填）'),
  ).not.toBeInTheDocument();
  expect(
    transport.mock.calls.every(([config]) => config.method === 'get'),
  ).toBe(true);
});

it('账单只有历史失效费用时不打开取消确认', async () => {
  transport.mockImplementation(async (config) => ({
    config,
    status: 200,
    statusText: 'OK',
    headers: {},
    data: {
      success: true,
      data: config.url?.endsWith('/bills/n-1')
        ? {
            id: 'n-1',
            version: '3',
            lines: [{ orderId: 'order-a', active: false }],
          }
        : [],
    },
  }));
  renderWithApp(<FinanceBillsPage />);
  fireEvent.click(screen.getByText('取消'));
  expect(
    await screen.findByText('账单没有可释放的有效费用，请刷新后重试'),
  ).toBeInTheDocument();
  expect(
    screen.queryByPlaceholderText('请输入取消原因（必填）'),
  ).not.toBeInTheDocument();
  expect(
    transport.mock.calls.every(([config]) => config.method === 'get'),
  ).toBe(true);
});

it('账单确认的本地catch及空响应保留真实弹层，重试成功才刷新关闭', async () => {
  const confirmBill = vi.spyOn(
    settlementService,
    'settlementServiceConfirmBill',
  );
  confirmBill
    .mockRejectedValueOnce(new Error('版本冲突'))
    .mockResolvedValueOnce(undefined as never)
    .mockResolvedValueOnce({ success: false, data: { id: 'n-1' } })
    .mockResolvedValueOnce({ success: true, data: {} })
    .mockResolvedValueOnce({ success: true, data: { id: 'n-1' } });
  transport.mockImplementation(async (config) => ({
    config,
    status: 200,
    statusText: 'OK',
    headers: {},
    data: { success: true, data: [] },
  }));
  try {
    renderWithApp(<FinanceBillsPage />);
    fireEvent.click(screen.getByText('确认账单'));
    for (let n = 1; n <= 4; n++) {
      fireEvent.click(await screen.findByRole('button', { name: /提交确认$/ }));
      await waitFor(() => expect(confirmBill).toHaveBeenCalledTimes(n));
      await waitFor(() =>
        expect(
          screen.getByRole('button', { name: /提交确认$/ }),
        ).not.toHaveClass('ant-btn-loading'),
      );
      expect(confirmOpenChange).not.toHaveBeenCalledWith(false);
      expect(screen.getByText('确认账单')).toHaveClass('ant-popover-open');
      expect(reload).not.toHaveBeenCalled();
      expect(
        screen.queryByText('账单已确认，进入待开票/待核销流'),
      ).not.toBeInTheDocument();
    }
    fireEvent.click(screen.getByRole('button', { name: /提交确认$/ }));
    await waitFor(() => expect(reload).toHaveBeenCalledOnce());
    await waitFor(() => expect(confirmOpenChange).toHaveBeenCalledWith(false));
    await waitFor(() =>
      expect(
        screen.getByText('确认真实账单？').closest('.ant-popconfirm'),
      ).toHaveClass('ant-zoom-big-leave'),
    );
  } finally {
    confirmBill.mockRestore();
  }
});
