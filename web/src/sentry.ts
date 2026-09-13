import * as Sentry from '@sentry/react';

// Sentry 轻量接入：仅在部署环境注入 SENTRY_DSN 时初始化，未配置时完全跳过
// （与未接入时行为一致，零网络开销）。只启用异常捕获（全局 JS 异常与
// Promise 拒绝由默认 handler 覆盖），不开启性能追踪、会话回放与 PII。
const sentryDsn = process.env.SENTRY_DSN;
if (sentryDsn) {
  Sentry.init({
    dsn: sentryDsn,
    release: process.env.COMMIT_HASH,
    environment:
      process.env.UMI_ENV === 'prod'
        ? 'production'
        : process.env.UMI_ENV || 'development',
    beforeSend(event) {
      // 双保险：事件请求头不得携带令牌或 Cookie；默认配置本就不采集请求体。
      const headers = event.request?.headers;
      if (headers) {
        delete headers.Authorization;
        delete headers.authorization;
        delete headers.Cookie;
        delete headers.cookie;
      }
      return event;
    },
  });
}
