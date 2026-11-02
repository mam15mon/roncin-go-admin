export const DINGTALK_ORIGIN = 'https://login.dingtalk.com';
export const CALLBACK_PATH = '/user/login/dingtalk/callback';

export type QRConfig = {
  qrCodeUrl: string;
  state: string;
  expiresAt: number;
};

export type LoginConfig = { enabled: false } | { enabled: true; qr: QRConfig };

export function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

// 仅绘制后端本次扫码尝试生成的钉钉确认链接，不把普通 OAuth 授权地址当二维码。
export function parseDingTalkLoginResponse(response: unknown): LoginConfig {
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
    typeof config.qrCodeUrl !== 'string' ||
    typeof config.state !== 'string' ||
    !config.state.trim() ||
    typeof config.expiresAt !== 'string'
  ) {
    throw new Error('钉钉登录配置不完整，请联系管理员');
  }
  const qrCode = new URL(config.qrCodeUrl);
  const expiresAt = Date.parse(config.expiresAt);
  if (
    qrCode.origin !== DINGTALK_ORIGIN ||
    qrCode.username ||
    qrCode.password ||
    qrCode.hash ||
    qrCode.pathname !== '/oauth2/qr_confirm.htm' ||
    qrCode.searchParams.getAll('code').length !== 1 ||
    !qrCode.searchParams.get('code')?.trim() ||
    !Number.isFinite(expiresAt)
  ) {
    throw new Error('钉钉二维码配置不正确，请联系管理员');
  }
  return {
    enabled: true,
    qr: { qrCodeUrl: qrCode.href, state: config.state, expiresAt },
  };
}
