import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import type { AxiosAdapter } from 'axios';
import { beforeEach, expect, it, vi } from 'vitest';
import { FinanceNettingStatus } from '@/enums.generated';
import FinanceNettingsPage from './index';

const { transport, notify, reload } = vi.hoisted(() => ({
  transport: vi.fn<AxiosAdapter>(),
  notify: vi.fn(),
  reload: vi.fn(),
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
  useAccess: () => ({
    canOperateOrganization: () => true,
    canReverseFinanceNettings: true,
  }),
}));
vi.mock('@/components/ui', async (original) => {
  const actual = await original<typeof import('@/components/ui')>();
  return {
    ...actual,
    FinanceLedgerTemplate: (props: {
      actionRef: { current: unknown };
      columns: {
        valueType?: string;
        render?: (
          value: unknown,
          record: API.FinanceNetting,
        ) => React.ReactNode;
      }[];
    }) => {
      props.actionRef.current = { reload };
      const column = props.columns.find((item) => item.valueType === 'option');
      return (
        <>
          {column?.render?.(null, {
            id: 'n-1',
            version: '2',
            status: FinanceNettingStatus.FINANCE_NETTING_STATUS_DRAFT,
          })}
        </>
      );
    },
  };
});
beforeEach(() => vi.clearAllMocks());
it('对冲取消真实409后原因仍在且不刷新；缺ID不能完成，重试成功才关闭', async () => {
  let attempts = 0;
  transport.mockImplementation(async (config) => {
    if (config.method !== 'get') {
      attempts++;
      if (attempts === 1)
        throw {
          config,
          response: { status: 409, data: { message: '对冲版本冲突' } },
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
      data: { success: true, data: [] },
    };
  });
  renderWithApp(<FinanceNettingsPage />);
  fireEvent.click(screen.getByText('取消'));
  const reason = screen.getByPlaceholderText('请输入取消原因（必填）');
  fireEvent.change(reason, { target: { value: '保留取消原因' } });
  for (let n = 1; n <= 2; n++) {
    fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
    await waitFor(() => expect(attempts).toBe(n));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /确.*认/ })).not.toHaveClass(
        'ant-btn-loading',
      ),
    );
    expect(reason).toHaveValue('保留取消原因');
    expect(reload).not.toHaveBeenCalled();
    expect(
      screen.queryByText('对冲单已取消；草稿未占用账单余额'),
    ).not.toBeInTheDocument();
  }
  fireEvent.click(screen.getByRole('button', { name: /确.*认/ }));
  await waitFor(() => expect(reload).toHaveBeenCalledOnce());
  await waitFor(() => expect(document.querySelector('.ant-modal')).toHaveClass('ant-zoom-leave'));
  expect(notify).toHaveBeenCalledOnce();
});
