import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import type { AxiosAdapter } from 'axios';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import ResetPasswordModal from './ResetPasswordModal';

const { transport, notify } = vi.hoisted(() => ({
  transport: vi.fn<AxiosAdapter>(),
  notify: vi.fn(),
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

beforeEach(() => vi.clearAllMocks());
describe('重置密码真实请求失败反馈', () => {
  it('409及业务失败消费为undefined后保留密码，不关闭不刷新；成功报文才完成', async () => {
    transport.mockImplementationOnce(async (config) => {
      throw {
        config,
        response: { status: 409, data: { message: '并发冲突' } },
      };
    });
    transport.mockImplementationOnce(async (config) => ({
      config,
      status: 200,
      statusText: 'OK',
      headers: {},
      data: { success: false, message: '重置被拒绝' },
    }));
    transport.mockImplementationOnce(async (config) => ({
      config,
      status: 200,
      statusText: 'OK',
      headers: {},
      data: { success: true },
    }));
    const onClose = vi.fn();
    const onReload = vi.fn();
    renderWithApp(
      <ResetPasswordModal
        user={{ id: 'u-1', username: 'operator', hasPassword: true }}
        onClose={onClose}
        onReload={onReload}
      />,
    );
    const password = screen.getByPlaceholderText('请输入至少 12 位新密码');
    fireEvent.change(password, { target: { value: 'safe-retry-password' } });
    for (let n = 1; n <= 2; n++) {
      fireEvent.click(screen.getByRole('button', { name: /确.*定/ }));
      await waitFor(() => expect(notify).toHaveBeenCalledTimes(n));
      expect(password).toHaveValue('safe-retry-password');
      expect(onClose).not.toHaveBeenCalled();
      expect(onReload).not.toHaveBeenCalled();
      expect(
        screen.queryByText('密码已重置，该用户现有在线会话已全部失效'),
      ).not.toBeInTheDocument();
      await waitFor(() =>
        expect(screen.getByRole('button', { name: /确.*定/ })).not.toHaveClass(
          'ant-btn-loading',
        ),
      );
    }
    fireEvent.click(screen.getByRole('button', { name: /确.*定/ }));
    await waitFor(() => expect(onReload).toHaveBeenCalledOnce());
    expect(onClose).toHaveBeenCalled();
    expect(notify).toHaveBeenCalledTimes(2);
  });
});
