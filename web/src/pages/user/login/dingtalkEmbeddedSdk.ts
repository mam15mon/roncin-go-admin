const SDK_TIMEOUT_MS = 15_000;

type RequestAuthCode =
  typeof import('dingtalk-jsapi/api/runtime/permission/requestAuthCode').default;
let sdk: Promise<RequestAuthCode> | undefined;

function loadSdk(): Promise<RequestAuthCode> {
  if (!sdk) {
    sdk = import('dingtalk-jsapi/entry/union')
      .then(
        () => import('dingtalk-jsapi/api/runtime/permission/requestAuthCode'),
      )
      .then((module) => module.default)
      .catch(() => {
        sdk = undefined;
        throw new Error('钉钉客户端组件加载失败，请重试或使用账号登录');
      });
  }
  return sdk;
}

/** 仅使用旧版 runtime.permission 协议，与企业 getuserinfo 换码配对。 */
export function requestEmbeddedAuthCode(
  corpId: string,
  isActive: () => boolean,
): Promise<string> {
  return new Promise((resolve, reject) => {
    let finished = false;
    const timer = setTimeout(() => {
      finished = true;
      reject(new Error('钉钉客户端认证超时，请重试或使用账号登录'));
    }, SDK_TIMEOUT_MS);
    void (async () => {
      try {
        const requestAuthCode = await loadSdk();
        if (finished || !isActive()) {
          resolve('');
          return;
        }
        const result = await requestAuthCode({ corpId });
        if (!result?.code?.trim())
          throw new Error('钉钉客户端未返回有效授权码，请重试');
        resolve(result.code);
      } catch (error) {
        reject(
          error instanceof Error && error.message.startsWith('钉钉客户端')
            ? error
            : new Error('钉钉客户端认证失败，请重试或使用账号登录'),
        );
      } finally {
        finished = true;
        clearTimeout(timer);
      }
    })();
  });
}
