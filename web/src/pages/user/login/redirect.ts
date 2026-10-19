/** 认证完成后只允许返回同源站内页面，保留查询条件和锚点。 */
export function safeLoginRedirect(value: string | null): string {
  if (!value?.startsWith('/') || value.startsWith('//') || value.includes('\\'))
    return '/';
  try {
    const parsed = new URL(value, window.location.origin);
    let pathname = decodeURIComponent(parsed.pathname);
    pathname = pathname.replace(/\/+$/, '').toLowerCase();
    if (
      parsed.origin !== window.location.origin ||
      pathname.includes('\\') ||
      pathname === '/login' ||
      pathname === '/user/login' ||
      pathname.startsWith('/user/login/')
    )
      return '/';
    return `${parsed.pathname}${parsed.search}${parsed.hash}`;
  } catch {
    return '/';
  }
}
