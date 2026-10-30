import type { ProLayoutProps } from '@ant-design/pro-components';
import { ProLayout } from '@ant-design/pro-components';
import { layoutDimensions } from '@root/config/layout';
import { Grid } from 'antd';
import { useEffect, useLayoutEffect, useMemo, useRef, useState } from 'react';
import { flushSync } from 'react-dom';
import { Link, Outlet, useLocation } from 'react-router';
import { HeaderMenus } from '@/components/layout/HeaderMenus';
import { HeaderTitle } from '@/components/layout/HeaderTitle';
import {
  getOrganizationWorkspaceKey,
  OrganizationWorkspace,
} from '@/components/layout/OrganizationWorkspace';
import {
  buildWorkspaceMenus,
  resolveWorkspaceMenuKey,
  toMobileWorkspaceMenus,
} from '@/components/layout/workspaceMenus';
import OrganizationSwitcher from '@/components/OrganizationSwitcher';
import { AvatarDropdown } from '@/components/RightContent/AvatarDropdown';
import { clearOrderMasterDataCache } from '@/features/orders/options';
import { buildMenuData } from '@/router/adaptRoutes';
import { history } from '@/router/history';
import { RouteErrorPage } from '@/router/RouteErrorPage';
import { LOGIN_PATH, PUBLIC_AUTH_PATHS, useInitialState } from './AppProvider';
import { useAccess } from './access';

// ProLayout 壳：插槽与守卫逻辑自 src/app.tsx 的 RunTimeLayoutConfig 原样平移。
// menuItemRender 的 prefetch 属性随 Umi routePrefetch 一并放弃（RR Link 无此能力）。
export function AppLayout() {
  const { initialState, setInitialState } = useInitialState();
  const currentUser = initialState?.currentUser;
  const layoutSettings = initialState?.settings as
    | Partial<ProLayoutProps>
    | undefined;
  const accessState = useAccess();
  const location = useLocation();
  const [workspaceSuspended, setWorkspaceSuspended] = useState(false);
  const [navigationErrorKey, setNavigationErrorKey] = useState<string>();
  const pendingSwitchRef = useRef<
    | {
        previousLocationKey: string;
        destination: string;
        workspaceKey: string;
        resolve: () => void;
        recover: () => void;
      }
    | undefined
  >(undefined);

  // navigate Promise 表示路由数据已就绪，RouterProvider 的 React 提交仍可能延后。
  // 只有实际观察到目标 location 与新身份后，才重新挂载业务树。
  useLayoutEffect(() => {
    const pending = pendingSwitchRef.current;
    if (
      !pending ||
      location.key === pending.previousLocationKey ||
      getOrganizationWorkspaceKey(currentUser) !== pending.workspaceKey
    )
      return;
    if (location.pathname + location.search !== pending.destination) {
      pending.recover();
      return;
    }
    pendingSwitchRef.current = undefined;
    setNavigationErrorKey(history.hasRouteError ? location.key : undefined);
    setWorkspaceSuspended(false);
    pending.resolve();
  }, [location.key, location.pathname, location.search, currentUser]);

  useEffect(
    () => () => {
      pendingSwitchRef.current?.resolve();
      pendingSwitchRef.current = undefined;
    },
    [],
  );

  const switchWorkspace = (user: API.CurrentUser, destination: string) => {
    let recovering = false;
    const committed = new Promise<void>((resolve) => {
      const pending = {
        previousLocationKey: history.location.key,
        destination,
        workspaceKey: getOrganizationWorkspaceKey(user),
        resolve,
        recover: () => {
          if (pendingSwitchRef.current !== pending) return;
          if (recovering) {
            pendingSwitchRef.current = undefined;
            setNavigationErrorKey(history.location.key);
            setWorkspaceSuspended(false);
            resolve();
            return;
          }
          // 浏览器后退/并发导航可取消尚在加载的目标；只尝试一次安全工作台。
          recovering = true;
          pending.previousLocationKey = history.location.key;
          pending.destination = '/welcome';
          navigate('/welcome');
        },
      };
      const navigate = (path: string) => {
        // 不等待被取消的 lazy Promise；目标提交、替换或卸载均能结束本次切换。
        void Promise.resolve()
          .then(() => history.replace(path))
          .then(() => {
            if (
              pendingSwitchRef.current === pending &&
              history.location.pathname + history.location.search !==
                pending.destination
            )
              pending.recover();
          }, pending.recover);
      };
      pendingSwitchRef.current = pending;
      // API 已成功轮转会话：先同步卸载旧详情及其请求/缓存，再发布新身份。
      flushSync(() => setWorkspaceSuspended(true));
      clearOrderMasterDataCache();
      flushSync(() => {
        setInitialState((state) => ({ ...state, currentUser: user }));
      });
      navigate(destination);
    });
    return committed;
  };

  const screens = Grid.useBreakpoint();
  const isMobile = screens.md === false;
  const workspaceMenus = useMemo(
    () => buildWorkspaceMenus(accessState),
    [accessState],
  );
  const supplementalMenus = useMemo(
    () => toMobileWorkspaceMenus(workspaceMenus),
    [workspaceMenus],
  );
  const workspaceSelectedKey = resolveWorkspaceMenuKey(
    location.pathname,
    location.search,
    workspaceMenus,
  );
  const menuData = useMemo(() => buildMenuData(accessState), [accessState]);

  // 平移 onPageChange 未登录守卫。
  useEffect(() => {
    if (!currentUser && !PUBLIC_AUTH_PATHS.has(location.pathname)) {
      history.replace(
        `${LOGIN_PATH}?redirect=${encodeURIComponent(location.pathname + location.search + location.hash)}`,
      );
    }
  }, [currentUser, location.pathname, location.search, location.hash]);

  return (
    <ProLayout
      {...layoutSettings}
      location={{ pathname: location.pathname }}
      route={menuData}
      menuDataRender={(items) =>
        isMobile ? [...items, ...supplementalMenus] : items
      }
      menuProps={
        isMobile && workspaceSelectedKey
          ? { selectedKeys: [workspaceSelectedKey] }
          : undefined
      }
      menu={{
        locale: false,
        collapsedWidth: layoutDimensions.collapsedSiderWidth,
      }}
      menuHeaderRender={(_logo, title) => (
        <Link
          to="/welcome"
          style={{ display: 'flex', alignItems: 'center', gap: 10 }}
        >
          {typeof layoutSettings?.logo === 'string' ? (
            <img
              src={layoutSettings.logo}
              width="auto"
              height={22}
              alt="logo"
            />
          ) : typeof layoutSettings?.logo === 'function' ? (
            layoutSettings.logo()
          ) : (
            layoutSettings?.logo
          )}
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
          <Link
            to={item.path}
            onClick={item.isMobile ? item.onClick : undefined}
            className="roncin-menu-item-link"
          >
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
        <OrganizationSwitcher
          key="organization"
          onSwitchWorkspace={switchWorkspace}
        />,
      ]}
      avatarProps={{
        src: currentUser?.avatarUrl,
        title: currentUser?.displayName ?? currentUser?.username,
        render: (_, avatarChildren) => (
          <AvatarDropdown>{avatarChildren}</AvatarDropdown>
        ),
      }}
    >
      {!workspaceSuspended && (
        <OrganizationWorkspace key={getOrganizationWorkspaceKey(currentUser)}>
          {navigationErrorKey === location.key ? (
            <RouteErrorPage />
          ) : (
            <Outlet />
          )}
        </OrganizationWorkspace>
      )}
    </ProLayout>
  );
}
