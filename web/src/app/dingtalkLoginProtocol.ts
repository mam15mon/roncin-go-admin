export const DINGTALK_ORIGIN = 'https://login.dingtalk.com';
export const CALLBACK_PATH = '/user/login/dingtalk/callback';
export const PRELOADED_FRAME_ID = 'roncin-dingtalk-login-frame';

export type FrameConfig = {
  src: string;
  state: string;
  expiresAt: number;
  callback: URL;
};

export type LoginConfig =
  | { enabled: false }
  | { enabled: true; frame: FrameConfig };

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

// HTML 开发入口与浏览器共用同一配置校验，不让预载绕过扫码组件的安全边界。
export function parseDingTalkLoginResponse(
  response: unknown,
  pageOrigin: string,
): LoginConfig {
  if (!isRecord(response) || !response.data || response.success === false) {
    throw new Error('获取钉钉登录配置失败，请重试');
  }
  const config = response.data;
  if (
    !isRecord(config) ||
    (config.enabled !== undefined && typeof config.enabled !== 'boolean')
  ) {
    throw new Error('钉钉登录配置不完整，请联系管理员');
  }
  if (!config.enabled) return { enabled: false };
  if (
    typeof config.authorizeUrl !== 'string' ||
    !config.authorizeUrl ||
    typeof config.expiresAt !== 'string' ||
    !config.expiresAt
  ) {
    throw new Error('钉钉登录配置不完整，请联系管理员');
  }
  const authorize = new URL(config.authorizeUrl);
  const callback = new URL(authorize.searchParams.get('redirect_uri') ?? '');
  const state = authorize.searchParams.get('state');
  const expiresAt = Date.parse(config.expiresAt);
  if (
    authorize.origin !== DINGTALK_ORIGIN ||
    authorize.username ||
    authorize.password ||
    authorize.pathname !== '/oauth2/auth' ||
    callback.origin !== pageOrigin ||
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
