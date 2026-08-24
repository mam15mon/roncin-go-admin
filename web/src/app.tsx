import type { Settings as LayoutSettings } from '@ant-design/pro-components';
import type { RequestConfig, RunTimeLayoutConfig } from '@umijs/max';
import { history, Link } from '@umijs/max';
import { Result } from 'antd';
import React from 'react';
import { HeaderMenus } from '@/components/layout/HeaderMenus';
import { HeaderTitle } from '@/components/layout/HeaderTitle';
import { TagsView } from '@/components/layout/TagsView';
import OrganizationSwitcher from '@/components/OrganizationSwitcher';
import { AvatarDropdown } from '@/components/RightContent/AvatarDropdown';
import { authServiceMe } from '@/services/roncin/authService';
import defaultSettings from '../config/defaultSettings';
import { errorConfig } from './requestErrorConfig';

const loginPath = '/user/login';
const publicAuthPaths = new Set([
  loginPath,
  '/user/login/wecom/callback',
  '/user/login/dingtalk/callback',
]);
const layoutSettings = defaultSettings as Partial<LayoutSettings>;

export interface InitialState {
  settings?: Partial<LayoutSettings>;
  currentUser?: API.CurrentUser;
  fetchUserInfo?: () => Promise<API.CurrentUser | undefined>;
}

export async function getInitialState(): Promise<InitialState> {
  const fetchUserInfo = async () => {
    const response = await authServiceMe({ skipErrorHandler: true });
    return response.data;
  };

  if (publicAuthPaths.has(history.location.pathname)) {
    return { fetchUserInfo, settings: layoutSettings };
  }

  try {
    return {
      fetchUserInfo,
      currentUser: await fetchUserInfo(),
      settings: layoutSettings,
    };
  } catch {
    const { pathname, search, hash } = history.location;
    history.replace(
      `${loginPath}?redirect=${encodeURIComponent(pathname + search + hash)}`,
    );
    return { fetchUserInfo, settings: layoutSettings };
  }
}

export const layout: RunTimeLayoutConfig = ({ initialState }) => ({
  menuHeaderRender: (logo, title) => (
    <Link to="/welcome" style={{ display: 'flex', alignItems: 'center', gap: 10 }}>
      {logo}
      {title}
    </Link>
  ),
  collapsedButtonRender: (_collapsed, defaultDom) => (
    <div className="roncin-sider-bottom-trigger" title={_collapsed ? '展开侧栏' : '收起侧栏'}>
      {defaultDom}
    </div>
  ),
  menuItemRender: (item, dom) =>
    item.path ? (
      <Link to={item.path} className="roncin-menu-item-link" prefetch>
        {dom}
      </Link>
    ) : (
      dom
    ),
  headerContentRender: () => (
    <div className="roncin-header-content">
      <HeaderTitle />
    </div>
  ),
  actionsRender: () => [
    <HeaderMenus key="header-menus" />,
    <OrganizationSwitcher key="organization" />,
  ],
  avatarProps: {
    src: initialState?.currentUser?.avatarUrl,
    title:
      initialState?.currentUser?.displayName ??
      initialState?.currentUser?.username,
    render: (_, avatarChildren) => (
      <AvatarDropdown>{avatarChildren}</AvatarDropdown>
    ),
  },
  onPageChange: () => {
    if (
      !initialState?.currentUser &&
      !publicAuthPaths.has(history.location.pathname)
    ) {
      history.replace(loginPath);
    }
  },
  unAccessible: <Result status="403" title="403" subTitle="无权访问此页面" />,
  childrenRender: (children) => (
    <div className="roncin-layout-wrapper">
      <TagsView />
      <div className="roncin-layout-main">{children}</div>
    </div>
  ),
  ...initialState?.settings,
});

export const request: RequestConfig = {
  baseURL: '',
  withCredentials: true,
  ...errorConfig,
};
