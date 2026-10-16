/** 后端分组路径是权限模块、业务线和功能归属的唯一来源。 */
export function getPermissionGroupSegments(group?: string): string[] {
  const segments = (group || '其他功能')
    .split('·')
    .map((segment) => segment.trim())
    .filter(Boolean);
  return segments.length ? segments : ['其他功能'];
}
