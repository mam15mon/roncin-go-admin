import { cleanup, render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';
import DingTalkCallback from './dingtalk-callback';

const { dingTalkLogin } = vi.hoisted(() => ({
  dingTalkLogin: vi.fn(),
}));

vi.mock('@umijs/max', () => ({
  Helmet: ({ children }: { children: React.ReactNode }) => children,
  useModel: () => ({ setInitialState: vi.fn() }),
}));

vi.mock('@/services/roncin/authService', () => ({
  authServiceDingTalkLogin: dingTalkLogin,
  authServiceRegisterDingTalkUser: vi.fn(),
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

  it('人员未注册时提供注册入口和返回登录操作', async () => {
    dingTalkLogin.mockRejectedValueOnce({
      data: {
        message: '当前人员尚未注册，请先完成钉钉扫码注册',
        reason: 'AUTH_DINGTALK_NOT_REGISTERED',
      },
    });

    render(
      <App>
        <DingTalkCallback />
      </App>,
    );

    expect(await screen.findByText('前往注册')).toBeInTheDocument();
    expect(screen.getByText('返回登录')).toBeInTheDocument();
    expect(
      screen.getByText('当前人员尚未注册，请先完成钉钉扫码注册'),
    ).toBeInTheDocument();
  });
});
