import type { ProLayoutProps } from '@ant-design/pro-components';
import { ProLayout } from '@ant-design/pro-components';
import { useEffect, useMemo } from 'react';
import { Link, Outlet, useLocation } from 'react-router';
import { AvatarDropdown } from '@/components/RightContent/AvatarDropdown';
import { HeaderMenus } from '@/components/layout/HeaderMenus';
import { HeaderTitle } from '@/components/layout/HeaderTitle';
import {
  getOrganizationWorkspaceKey,
  OrganizationWorkspace,
} from '@/components/layout/OrganizationWorkspace';
import OrganizationSwitcher from '@/components/OrganizationSwitcher';
import { buildMenuData } from '@/router/adaptRoutes';
import { history } from '@/router/history';
import { LOGIN_PATH, PUBLIC_AUTH_PATHS, useInitialState } from './AppProvider';
import { useAccess } from './access';

// ProLayout 壳：插槽与守卫逻辑自 src/app.tsx 的 RunTimeLayoutConfig 原样平移。
// menuItemRender 的 prefetch 属性随 Umi routePrefetch 一并放弃（RR Link 无此能力）。
export function AppLayout() {
  const { currentUser, settings } = useInitialState();
  const accessState = useAccess();
  const location = useLocation();
  const menuData = useMemo(
    () => buildMenuData(accessState),
    [accessState],
  );

  // 平移 onPageChange 未登录守卫。
  useEffect(() => {
    if (!currentUser && !PUBLIC_AUTH_PATHS.has(location.pathname)) {
      history.replace(LOGIN_PATH);
    }
  }, [currentUser, location.pathname]);

  const layoutSettings = settings as Partial<ProLayoutProps>;

  return (
    <ProLayout
      {...layoutSettings}
      location={{ pathname: location.pathname }}
      route={menuData}
      menu={{ locale: false }}
      menuHeaderRender={(logo, title) => (
        <Link
          to="/welcome"
          style={{ display: 'flex', alignItems: 'center', gap: 10 }}
        >
          {logo}
          {title}
        </Link>
      )}
      collapsedButtonRender={(collapsed, defaultDom) => (
        <div
          className="roncin-sider-bottom-trigger"
          title={collapsed ? '展开侧栏' : '收起侧栏'}
        >
          {defaultDom}
        </div>
      )}
      menuItemRender={(item, dom) =>
        item.path ? (
          <Link to={item.path} className="roncin-menu-item-link">
            {dom}
          </Link>
        ) : (
          dom
        )
      }
      headerContentRender={() => (
        <div className="roncin-header-content">
          <HeaderTitle />
        </div>
      )}
      actionsRender={() => [
        <HeaderMenus key="header-menus" />,
        <OrganizationSwitcher key="organization" />,
      ]}
      avatarProps={{
        src: currentUser?.avatarUrl,
        title:
          currentUser?.displayName ?? currentUser?.username,
        render: (_, avatarChildren) => (
          <AvatarDropdown>{avatarChildren}</AvatarDropdown>
        ),
      }}
    >
      <OrganizationWorkspace key={getOrganizationWorkspaceKey(currentUser)}>
        <Outlet />
      </OrganizationWorkspace>
    </ProLayout>
  );
}
