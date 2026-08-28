import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import DingTalkCallback from './dingtalk-callback';

const { dingTalkLogin, registerDingTalkUser } = vi.hoisted(() => ({
  dingTalkLogin: vi.fn(),
  registerDingTalkUser: vi.fn(),
}));

vi.mock('@umijs/max', () => ({
  Helmet: ({ children }: { children: React.ReactNode }) => children,
  useModel: () => ({ setInitialState: vi.fn() }),
}));

vi.mock('@/services/roncin/authService', () => ({
  authServiceDingTalkLogin: dingTalkLogin,
  authServiceRegisterDingTalkUser: registerDingTalkUser,
}));

describe('DingTalkCallback', () => {
  beforeEach(() => {
    sessionStorage.clear();
    window.history.replaceState(
      {},
      '',
      '/user/login/dingtalk/callback?authCode=code&state=state',
    );
  });

  afterEach(() => {
    cleanup();
    vi.clearAllMocks();
  });

  it('一次身份验证后由人员确认注册，不再重复扫码', async () => {
    dingTalkLogin.mockResolvedValueOnce({
      data: {
        status: 2,
        displayName: '张三',
      },
    });
    registerDingTalkUser.mockResolvedValueOnce({
      data: { displayName: '张三', status: 'PENDING' },
    });

    render(
      <App>
        <DingTalkCallback />
      </App>,
    );

    expect(await screen.findByText('确认注册')).toBeInTheDocument();
    expect(screen.getByText(/已确认 张三 属于本企业/)).toBeInTheDocument();

    fireEvent.click(screen.getByText('确认注册'));

    expect(await screen.findByText('入职或返聘申请已提交')).toBeInTheDocument();
    expect(registerDingTalkUser).toHaveBeenCalledWith(
      {},
      { skipErrorHandler: true },
    );
  });
});
