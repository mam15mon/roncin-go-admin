import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { cleanup } from '@testing-library/react';
import { App } from 'antd';
import { StrictMode } from 'react';
import { afterEach, beforeEach, vi } from 'vitest';
import { DingTalkLoginStatus } from '@/enums.generated';
import { DingTalkEmbeddedLogin } from '@/pages/user/login/components/dingtalk-embedded-login';

const mockState = vi.hoisted(() => ({
  config: vi.fn(),
  login: vi.fn(),
  code: vi.fn(),
  switchOrganization: vi.fn(),
  setInitialState: vi.fn(),
  switchLock: vi.fn(),
  success: vi.fn(),
  error: vi.fn(),
  initialState: undefined as { currentUser?: API.CurrentUser } | undefined,
}));
export const mocks = mockState;
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: mockState.initialState,
    setInitialState: mockState.setInitialState,
  }),
}));
vi.mock('@/services/roncin/authService', () => ({
  authServiceGetDingTalkEmbeddedConfig: mockState.config,
  authServiceDingTalkEmbeddedLogin: mockState.login,
  authServiceSwitchOrganization: mockState.switchOrganization,
}));
vi.mock('@/pages/user/login/dingtalkEmbeddedSdk', () => ({
  requestEmbeddedAuthCode: mockState.code,
}));
vi.mock('antd', async (original) => {
  const actual = await original<typeof import('antd')>();
  return {
    ...actual,
    App: Object.assign(actual.App, {
      useApp: () => ({
        message: { success: mockState.success, error: mockState.error },
      }),
    }),
  };
});
export function authenticated(multiple = false): API.DingTalkLoginResult {
  const organizations = [
    { id: 'org-a', name: '天津公司', code: 'TJ' },
    ...(multiple ? [{ id: 'org-b', name: '北京公司', code: 'BJ' }] : []),
  ];
  return {
    status: DingTalkLoginStatus.DING_TALK_LOGIN_STATUS_AUTHENTICATED,
    currentUser: {
      id: 'user-a',
      currentOrganization: organizations[0],
      organizations,
    },
  };
}
export function deferred<T>() {
  let resolve!: (value: T) => void;
  let reject!: (error: unknown) => void;
  const promise = new Promise<T>((accept, fail) => {
    resolve = accept;
    reject = fail;
  });
  return { promise, resolve, reject };
}
export function mount() {
  const onAccountLogin = vi.fn();
  const view = renderWithClient(
    <StrictMode>
      <App>
        <DingTalkEmbeddedLogin
          onAccountLogin={onAccountLogin}
          onSwitchLockChange={mocks.switchLock}
        />
      </App>
    </StrictMode>,
  );
  return { ...view, onAccountLogin };
}
export let replace: ReturnType<typeof vi.spyOn>;
beforeEach(() => {
  mocks.initialState = undefined;
  Object.values(mocks).forEach((value) => {
    if (vi.isMockFunction(value)) value.mockReset();
  });
  window.history.replaceState(
    {},
    '',
    '/user/login?redirect=%2Forders%2Fsea-export%2F123%3Ftab%3Dfees%23cost',
  );
  replace = vi.spyOn(window.location, 'replace').mockImplementation(() => {});
  mocks.config.mockResolvedValue({
    data: {
      enabled: true,
      corpId: 'corp-a',
      state: 'state-a',
      expiresAt: new Date(Date.now() + 300000).toISOString(),
    },
  });
  mocks.code.mockResolvedValue('code-a');
  mocks.login.mockResolvedValue({ data: authenticated() });
});
afterEach(() => {
  cleanup();
  vi.restoreAllMocks();
});
