import {
  authenticated,
  deferred,
  mocks,
  mount,
  replace,
} from '@root/tests/login-embedded-support';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { DingTalkLoginStatus } from '@/enums.generated';

describe('钉钉客户端免登', () => {
  it('StrictMode 及父层重渲染仅取一次码换一次会话，回到完整目标', async () => {
    const pending = deferred<string>();
    mocks.code.mockReturnValue(pending.promise);
    const view = mount();
    await waitFor(() => expect(mocks.code).toHaveBeenCalledOnce());
    await act(async () => pending.resolve('code-a'));
    await waitFor(() =>
      expect(replace).toHaveBeenCalledWith(
        '/orders/sea-export/123?tab=fees#cost',
      ),
    );
    expect(mocks.config).toHaveBeenCalledOnce();
    expect(mocks.code).toHaveBeenCalledWith('corp-a', expect.any(Function));
    expect(mocks.login).toHaveBeenCalledExactlyOnceWith(
      { authCode: 'code-a', state: 'state-a' },
      { skipErrorHandler: true },
    );
    expect(mocks.setInitialState).toHaveBeenCalledOnce();
    view.unmount();
  });
  it('已有本地有效会话不再获取配置或授权码', async () => {
    mocks.initialState = { currentUser: authenticated().currentUser };
    mount();
    await waitFor(() => expect(replace).toHaveBeenCalledOnce());
    expect(mocks.config).not.toHaveBeenCalled();
    expect(mocks.code).not.toHaveBeenCalled();
    expect(mocks.login).not.toHaveBeenCalled();
  });
  it.each([
    [DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_NOT_SYNCED, '人员尚未同步'],
    [
      DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_PENDING_ASSIGNMENT,
      '尚未分配组织',
    ],
    [DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_DISABLED, '账号已停用'],
  ])('人员状态 %s 显示处理提示并保持账号入口', async (status, title) => {
    mocks.login.mockResolvedValue({ data: { status } });
    mount();
    expect(await screen.findByText(title)).toBeVisible();
    expect(screen.getByRole('button', { name: '使用账号登录' })).toBeVisible();
    expect(replace).not.toHaveBeenCalled();
    expect(mocks.setInitialState).not.toHaveBeenCalled();
  });
  it('多组织明确选择其他工作台后轮转，保留完整目标', async () => {
    mocks.login.mockResolvedValue({ data: authenticated(true) });
    mocks.switchOrganization.mockResolvedValue({ data: { id: 'user-a' } });
    mount();
    expect(await screen.findByText('选择进入的组织')).toBeVisible();
    expect(replace).not.toHaveBeenCalled();
    fireEvent.click(screen.getByRole('button', { name: '进入北京公司' }));
    await waitFor(() =>
      expect(replace).toHaveBeenCalledWith(
        '/orders/sea-export/123?tab=fees#cost',
      ),
    );
    expect(mocks.switchOrganization).toHaveBeenCalledExactlyOnceWith(
      { organizationId: 'org-b' },
      { skipErrorHandler: true },
    );
  });
  it('选择当前组织直接进入；组织信息来自本次认证', async () => {
    const data = authenticated(true);
    if (!data.currentUser) throw new Error('测试需要已认证用户');
    data.currentUser.currentOrganization = data.currentUser.organizations?.[1];
    mocks.login.mockResolvedValue({ data });
    mount();
    fireEvent.click(
      await screen.findByRole('button', { name: '进入北京公司' }),
    );
    expect(replace).toHaveBeenCalledWith(
      '/orders/sea-export/123?tab=fees#cost',
    );
    expect(mocks.switchOrganization).not.toHaveBeenCalled();
  });
  it('组织轮转失败保持选择页且不导航', async () => {
    mocks.login.mockResolvedValue({ data: authenticated(true) });
    mocks.switchOrganization.mockRejectedValue(new Error('工作台不可用'));
    mount();
    fireEvent.click(
      await screen.findByRole('button', { name: '进入北京公司' }),
    );
    await waitFor(() =>
      expect(mocks.error).toHaveBeenCalledWith('工作台不可用'),
    );
    expect(screen.getByText('选择进入的组织')).toBeVisible();
    expect(replace).not.toHaveBeenCalled();
  });
});
