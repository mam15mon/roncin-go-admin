import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const { requestCode } = vi.hoisted(() => ({ requestCode: vi.fn() }));
vi.mock('dingtalk-jsapi/entry/union', () => ({}));
vi.mock('dingtalk-jsapi/api/runtime/permission/requestAuthCode', () => ({
  default: requestCode,
}));
beforeEach(() => {
  vi.resetModules();
  requestCode.mockReset();
});
afterEach(() => {
  vi.useRealTimers();
});
describe('钉钉免登 SDK', () => {
  it('只调用 runtime.permission 的 corpId 签名并返回 code', async () => {
    requestCode.mockResolvedValue({ code: 'temporary-code' });
    const { requestEmbeddedAuthCode } = await import('./dingtalkEmbeddedSdk');
    await expect(requestEmbeddedAuthCode('corp-a', () => true)).resolves.toBe(
      'temporary-code',
    );
    expect(requestCode).toHaveBeenCalledExactlyOnceWith({ corpId: 'corp-a' });
  });
  it.each([undefined, {}, { code: '' }, { code: ' ' }])(
    '空结果 %j 失败',
    async (result) => {
      requestCode.mockResolvedValue(result);
      const { requestEmbeddedAuthCode } = await import('./dingtalkEmbeddedSdk');
      await expect(
        requestEmbeddedAuthCode('corp-a', () => true),
      ).rejects.toThrow('未返回有效授权码');
    },
  );
  it('SDK 原始失败不进入界面或日志', async () => {
    requestCode.mockRejectedValue({ message: 'raw-secret-code' });
    const { requestEmbeddedAuthCode } = await import('./dingtalkEmbeddedSdk');
    await expect(requestEmbeddedAuthCode('corp-a', () => true)).rejects.toThrow(
      '钉钉客户端认证失败',
    );
  });
  it('SDK 不返回时超时且不自动重试', async () => {
    vi.useFakeTimers();
    requestCode.mockReturnValue(new Promise(() => {}));
    const { requestEmbeddedAuthCode } = await import('./dingtalkEmbeddedSdk');
    const pending = requestEmbeddedAuthCode('corp-a', () => true);
    const assertion = expect(pending).rejects.toThrow('认证超时');
    await vi.advanceTimersByTimeAsync(15000);
    await assertion;
    expect(requestCode).toHaveBeenCalledOnce();
  });
  it('组件离开后 SDK 加载完成不再取码', async () => {
    const { requestEmbeddedAuthCode } = await import('./dingtalkEmbeddedSdk');
    void requestEmbeddedAuthCode('corp-a', () => false);
    await vi.dynamicImportSettled();
    expect(requestCode).not.toHaveBeenCalled();
  });
});
