import { mocks, mount, replace } from '@root/tests/login-embedded-support';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

describe('钉钉免登失败与迟到结果', () => {
  it.each(['disabled', 'missing-config', 'expired', 'config-error'])(
    '%s 明确失败，不调用 SDK 或自动重试',
    async (kind) => {
      if (kind === 'disabled') mocks.config.mockResolvedValue({ data: {} });
      if (kind === 'missing-config')
        mocks.config.mockResolvedValue({ data: { enabled: true } });
      if (kind === 'expired')
        mocks.config.mockResolvedValue({
          data: {
            enabled: true,
            corpId: 'a',
            state: 'a',
            expiresAt: '2000-01-01T00:00:00Z',
          },
        });
      if (kind === 'config-error')
        mocks.config.mockRejectedValue(new Error('配置接口失败'));
      mount();
      expect(await screen.findByText('钉钉免登未完成')).toBeVisible();
      expect(mocks.config).toHaveBeenCalledOnce();
      expect(mocks.code).not.toHaveBeenCalled();
      expect(mocks.login).not.toHaveBeenCalled();
      expect(replace).not.toHaveBeenCalled();
    },
  );
  it.each([undefined, {}, { status: 0 }, { status: 999 }, { status: 1 }])(
    '换码结果 %j 无效时明确失败',
    async (data) => {
      mocks.login.mockResolvedValue({ data });
      mount();
      expect(
        await screen.findByText('钉钉认证未返回有效结果，请重试'),
      ).toBeVisible();
      expect(replace).not.toHaveBeenCalled();
    },
  );
  it('显式重试重新获取配置和新码，失败不自动循环', async () => {
    mocks.code
      .mockRejectedValueOnce(new Error('钉钉客户端认证失败，请重试'))
      .mockResolvedValueOnce('new-code');
    mount();
    fireEvent.click(await screen.findByRole('button', { name: '重试免登' }));
    await waitFor(() => expect(replace).toHaveBeenCalledOnce());
    expect(mocks.config).toHaveBeenCalledTimes(2);
    expect(mocks.code).toHaveBeenCalledTimes(2);
    expect(mocks.login).toHaveBeenCalledExactlyOnceWith(
      { authCode: 'new-code', state: 'state-a' },
      { skipErrorHandler: true },
    );
  });
});
