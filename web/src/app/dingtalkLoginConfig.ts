import { type QueryClient, queryOptions } from '@tanstack/react-query';
import { authServiceGetDingTalkLoginConfig } from '@/services/roncin/authService';

export const DINGTALK_ORIGIN = 'https://login.dingtalk.com';
export const CALLBACK_PATH = '/user/login/dingtalk/callback';
const CONFIG_QUERY_KEY = ['auth', 'dingtalk-login-config'] as const;

export type FrameConfig = {
  src: string;
  state: string;
  expiresAt: number;
  callback: URL;
};

type LoginConfig = { enabled: false } | { enabled: true; frame: FrameConfig };

function parseConfig(config: API.DingTalkLoginConfig): LoginConfig {
  if (!config.enabled) return { enabled: false };
  if (!config.authorizeUrl || !config.expiresAt) {
    throw new Error('钉钉登录配置不完整，请联系管理员');
  }
  const authorize = new URL(config.authorizeUrl);
  const callback = new URL(authorize.searchParams.get('redirect_uri') ?? '');
  const state = authorize.searchParams.get('state');
  const expiresAt = Date.parse(config.expiresAt);
  if (
    authorize.origin !== DINGTALK_ORIGIN ||
    authorize.pathname !== '/oauth2/auth' ||
    callback.origin !== window.location.origin ||
    callback.pathname !== CALLBACK_PATH ||
    callback.username ||
    callback.password ||
    !state ||
    !authorize.searchParams.get('client_id') ||
    authorize.searchParams.get('response_type') !== 'code' ||
    !authorize.searchParams.get('scope') ||
    !Number.isFinite(expiresAt)
  ) {
    throw new Error('钉钉登录配置或回调地址不正确，请联系管理员');
  }
  authorize.searchParams.set('iframe', 'true');
  return {
    enabled: true,
    frame: { src: authorize.href, state, expiresAt, callback },
  };
}

async function loadConfig(): Promise<LoginConfig> {
  // 不消费 AbortSignal：已发出的配置响应会写 Cookie，必须由同一 Query 串行收齐。
  const response = await authServiceGetDingTalkLoginConfig({
    skipErrorHandler: true,
  });
  if (!response?.data || response.success === false) {
    throw new Error('获取钉钉登录配置失败，请重试');
  }
  try {
    return parseConfig(response.data);
  } catch (error) {
    if (error instanceof TypeError) {
      throw new Error('钉钉登录配置或回调地址不正确，请联系管理员');
    }
    throw error;
  }
}

// 预热完成到扫码组件挂载之间保留结果；模式返回由显式刷新启动新一轮。
export const dingTalkLoginConfigOptions = queryOptions({
  queryKey: CONFIG_QUERY_KEY,
  queryFn: loadConfig,
  staleTime: Infinity,
  gcTime: 5 * 60 * 1000,
  retryOnMount: false,
  refetchOnMount: (query) =>
    query.state.isInvalidated ||
    (query.state.data?.enabled &&
      query.state.data.frame.expiresAt <= Date.now())
      ? 'always'
      : false,
  refetchOnWindowFocus: false,
  refetchOnReconnect: false,
  meta: { silent: true },
});

export function preloadDingTalkLoginConfig(
  queryClient: QueryClient,
  pathname: string,
): Promise<void> {
  // 回调页不能覆盖本次换码的 state Cookie，已登录业务页也不启动配置请求。
  if (pathname !== '/user/login') return Promise.resolve();
  return queryClient.prefetchQuery(dingTalkLoginConfigOptions);
}

export function refreshDingTalkLoginConfig(queryClient: QueryClient): void {
  // fetchQuery 复用同 key 在途请求，不取消仍可能写 Cookie 的响应。
  // 失败留在查询中由扫码区展示；显式返回扫码允许重新获取配置。
  void queryClient
    .fetchQuery({ ...dingTalkLoginConfigOptions, staleTime: 0 })
    .catch(() => {});
}
