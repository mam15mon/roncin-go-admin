import { type QueryClient, queryOptions } from '@tanstack/react-query';
import { authServiceGetDingTalkLoginConfig } from '@/services/roncin/authService';
import {
  type LoginConfig,
  parseDingTalkLoginResponse,
} from './dingtalkLoginProtocol';
import { discardPreloadedDingTalkFrame } from './dingtalkPreloadedFrame';

export {
  CALLBACK_PATH,
  DINGTALK_ORIGIN,
  type FrameConfig,
} from './dingtalkLoginProtocol';

const CONFIG_QUERY_KEY = ['auth', 'dingtalk-login-config'] as const;

async function loadConfig(): Promise<LoginConfig> {
  try {
    let response: unknown;
    const bootstrap = document.getElementById('roncin-dingtalk-login-config');
    if (bootstrap) {
      // 一次性配置与本次 HTML 的 Cookie 对应，先移除，后续刷新必须请求新 state。
      bootstrap.remove();
      if (
        bootstrap.tagName !== 'SCRIPT' ||
        bootstrap.getAttribute('type') !== 'application/json'
      ) {
        throw new Error('钉钉登录配置格式不正确，请刷新页面');
      }
      response = JSON.parse(bootstrap.textContent ?? '');
    } else {
      // 不消费 AbortSignal：已发出的配置响应会写 Cookie，必须由同一 Query 串行收齐。
      response = await authServiceGetDingTalkLoginConfig({
        skipErrorHandler: true,
      });
    }
    const config = parseDingTalkLoginResponse(response, window.location.origin);
    if (!config.enabled || config.frame.expiresAt <= Date.now()) {
      discardPreloadedDingTalkFrame();
    }
    return config;
  } catch (error) {
    discardPreloadedDingTalkFrame();
    if (error instanceof SyntaxError) {
      throw new Error('钉钉登录配置格式不正确，请刷新页面');
    }
    if (error instanceof TypeError) {
      throw new Error('钉钉登录配置或回调地址不正确，请联系管理员');
    }
    throw error;
  }
}

// 预热完成到扫码组件挂载之间保留结果；模式切回保留有效实例，失效后才按需刷新。
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
  discardPreloadedDingTalkFrame();
  // fetchQuery 复用同 key 在途请求，不取消仍可能写 Cookie 的响应。
  // 失败留在查询中由扫码区展示；显式返回扫码允许重新获取配置。
  void queryClient
    .fetchQuery({ ...dingTalkLoginConfigOptions, staleTime: 0 })
    .catch(() => {});
}
