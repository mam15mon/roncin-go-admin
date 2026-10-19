/** 动作仅来自预览接口；站点 origin 由服务端配置校验，前端校验协议与固定业务路径。 */
export function notificationActionUrl(
  actionTitle?: string,
  actionUrl?: string,
) {
  if (!actionTitle?.trim() || !actionUrl || /[\s\\?#]/.test(actionUrl))
    return '';
  try {
    const url = new URL(actionUrl);
    const uuid = '[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}';
    const allowedPath =
      ['/welcome', '/orders', '/finance/exchange-rates'].includes(
        url.pathname,
      ) ||
      new RegExp(`^/orders/sea-export/${uuid}/fees$`, 'i').test(url.pathname) ||
      new RegExp(
        `^/commission-adjustments/${uuid}/my-supplement-source$`,
        'i',
      ).test(url.pathname);
    return /^https?:$/.test(url.protocol) &&
      url.hostname &&
      allowedPath &&
      !url.username &&
      !url.password &&
      !url.search &&
      !url.hash &&
      url.href === actionUrl
      ? actionUrl
      : '';
  } catch {
    return '';
  }
}
