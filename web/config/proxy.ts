type ProxyConfig = Record<
  string,
  {
    target: string;
    changeOrigin: boolean;
  }
>;

const localProxy: ProxyConfig = {
  '/api/': {
    target: 'http://127.0.0.1:8000',
    changeOrigin: false,
  },
};

function remoteProxy(environment: 'test' | 'pre', rawTarget?: string): ProxyConfig {
  const target = rawTarget?.trim();
  if (!target) {
    throw new Error(
      `--mode ${environment} 启动前必须设置 RONCIN_API_PROXY_TARGET`,
    );
  }

  let url: URL;
  try {
    url = new URL(target);
  } catch {
    throw new Error('RONCIN_API_PROXY_TARGET 必须是合法的绝对 URL');
  }
  if (url.protocol !== 'http:' && url.protocol !== 'https:') {
    throw new Error('RONCIN_API_PROXY_TARGET 仅支持 http 或 https 协议');
  }

  return {
    '/api/': {
      target: url.toString().replace(/\/$/, ''),
      changeOrigin: true,
    },
  };
}

export function getProxyConfig(
  environment: string,
  remoteTarget = process.env.RONCIN_API_PROXY_TARGET,
): ProxyConfig {
  // 裸 `vite`（未显式 --mode）的默认模式为 development，视同本地 dev。
  const flavor = environment === 'development' ? 'dev' : environment;
  switch (flavor) {
    case 'dev':
      return localProxy;
    case 'test':
    case 'pre':
      return remoteProxy(flavor, remoteTarget);
    default:
      throw new Error(`不支持的启动模式（vite --mode）：${environment}`);
  }
}
