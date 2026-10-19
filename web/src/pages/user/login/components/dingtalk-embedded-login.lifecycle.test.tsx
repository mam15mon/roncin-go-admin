import {
  authenticated,
  deferred,
  mocks,
  mount,
  replace,
} from '@root/tests/login-embedded-support';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';

describe('钉钉免登生命周期隔离', () => {
  it('换码前同步通知父级锁定，真实卸载清理父级锁', async () => {
    const pending = deferred<{ data: API.DingTalkLoginResult }>();
    mocks.login.mockImplementationOnce(() => {
      expect(mocks.switchLock).toHaveBeenLastCalledWith(true);
      return pending.promise;
    });
    const view = mount();
    await waitFor(() => expect(mocks.login).toHaveBeenCalledOnce());
    expect(screen.getByRole('button', { name: '使用账号登录' })).toBeDisabled();
    view.unmount();
    expect(mocks.switchLock).toHaveBeenLastCalledWith(false);
    await act(async () => pending.resolve({ data: authenticated() }));
    expect(replace).not.toHaveBeenCalled();
    expect(mocks.switchLock).toHaveBeenLastCalledWith(false);
  });
  it('切账号后迟到 SDK 结果不换码，回调也被同步隔离', async () => {
    const pending = deferred<string>();
    mocks.code.mockReturnValue(pending.promise);
    const view = mount();
    await waitFor(() => expect(mocks.code).toHaveBeenCalledOnce());
    const isActive = mocks.code.mock.calls[0][1];
    fireEvent.click(screen.getByRole('button', { name: '使用账号登录' }));
    expect(view.onAccountLogin).toHaveBeenCalledOnce();
    expect(isActive()).toBe(false);
    await act(async () => pending.resolve('late-code'));
    expect(mocks.login).not.toHaveBeenCalled();
    expect(replace).not.toHaveBeenCalled();
  });
  it.each(['code', 'login', 'organization'])(
    '离页后迟到 %s 结果无状态写入和导航',
    async (stage) => {
      const pendingCode = deferred<string>();
      const pendingLogin = deferred<{ data: API.DingTalkLoginResult }>();
      const pendingOrganization = deferred<{ data: { id: string } }>();
      if (stage === 'code') mocks.code.mockReturnValue(pendingCode.promise);
      if (stage === 'login') mocks.login.mockReturnValue(pendingLogin.promise);
      if (stage === 'organization') {
        mocks.login.mockResolvedValue({ data: authenticated(true) });
        mocks.switchOrganization.mockReturnValue(pendingOrganization.promise);
      }
      const view = mount();
      if (stage === 'code')
        await waitFor(() => expect(mocks.code).toHaveBeenCalledOnce());
      if (stage === 'login')
        await waitFor(() => expect(mocks.login).toHaveBeenCalledOnce());
      if (stage === 'organization') {
        fireEvent.click(
          await screen.findByRole('button', { name: '进入北京公司' }),
        );
        await waitFor(() =>
          expect(mocks.switchOrganization).toHaveBeenCalledOnce(),
        );
        mocks.setInitialState.mockClear();
      }
      view.unmount();
      await act(async () => {
        pendingCode.resolve('late-code');
        pendingLogin.resolve({ data: authenticated() });
        pendingOrganization.resolve({ data: { id: 'user-a' } });
      });
      if (stage === 'code') expect(mocks.login).not.toHaveBeenCalled();
      expect(mocks.setInitialState).not.toHaveBeenCalled();
      expect(mocks.success).not.toHaveBeenCalled();
      expect(replace).not.toHaveBeenCalled();
    },
  );
});
