import { describe, expect, it } from 'vitest';
import { getProxyConfig } from '../config/proxy';

describe('前端开发代理配置', () => {
  it('开发环境固定代理到本机服务端', () => {
    expect(getProxyConfig('dev')).toEqual({
      '/api/': {
        target: 'http://127.0.0.1:8000',
        changeOrigin: false,
      },
    });
  });

  it('测试和预发布环境必须显式提供合法目标', () => {
    expect(() => getProxyConfig('test', '')).toThrow(
      'RONCIN_API_PROXY_TARGET',
    );
    expect(() => getProxyConfig('pre', 'redis://127.0.0.1')).toThrow(
      '仅支持 http 或 https',
    );
    expect(() => getProxyConfig('production')).toThrow('不支持的 UMI_ENV');
  });

  it('远程环境开启跨主机代理且规范化末尾斜杠', () => {
    expect(getProxyConfig('test', ' https://test-api.example.com/ ')).toEqual({
      '/api/': {
        target: 'https://test-api.example.com',
        changeOrigin: true,
      },
    });
  });
});
