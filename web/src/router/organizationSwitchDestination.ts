import { generatePath, matchPath } from 'react-router';
import routes from '../../config/routes';
import type { AccessState, UmiRoute } from './routeTypes';
import { isWorkspaceTabPath, workspaceTabs } from './workspaceTabs';

function flattenRoutes(items: UmiRoute[]): UmiRoute[] {
  return items.flatMap((route) => [
    route,
    ...flattenRoutes(route.routes ?? []),
  ]);
}

const configuredRoutes = flattenRoutes(routes);
const welcome = '/welcome';

export type OrganizationSwitchDestination = {
  path: string;
  accessDenied: boolean;
};

/** 只迁移集中路由明确声明的功能位置，不携带记录、筛选、分页或锚点。 */
export function resolveOrganizationSwitchDestination(
  location: { pathname: string; search: string },
  access: AccessState,
): OrganizationSwitchDestination {
  const source = configuredRoutes.find(
    (route) =>
      route.organizationSwitch &&
      route.path &&
      matchPath({ path: route.path, end: true }, location.pathname),
  );
  if (!source?.path || !source.organizationSwitch) {
    return { path: welcome, accessDenied: false };
  }
  const match = matchPath(source.path, location.pathname);
  const targetPath =
    'keep' in source.organizationSwitch
      ? generatePath(source.path, match?.params)
      : generatePath(source.organizationSwitch.list, match?.params);
  // 动态 kind 只能回到实际存在的列表，不能以 Any 权限推测某种订单可读。
  const target = configuredRoutes.find(
    (route) =>
      route.path === targetPath &&
      route.organizationSwitch &&
      'keep' in route.organizationSwitch,
  );
  if (!target) return { path: welcome, accessDenied: false };
  if (target.access && !access[target.access]) {
    return { path: welcome, accessDenied: true };
  }
  if (!isWorkspaceTabPath(targetPath)) {
    return { path: targetPath, accessDenied: false };
  }
  const tabs = workspaceTabs[targetPath];
  const requestedKey = new URLSearchParams(location.search).get('tab');
  const requestedTab = tabs.find((tab) => tab.key === requestedKey);
  if (requestedTab && !requestedTab.visible(access)) {
    return { path: welcome, accessDenied: true };
  }
  const tab = requestedTab ?? tabs.find((item) => item.visible(access));
  if (!tab) return { path: welcome, accessDenied: true };
  return { path: `${targetPath}?tab=${tab.key}`, accessDenied: false };
}
