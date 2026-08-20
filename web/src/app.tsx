import type { Settings as LayoutSettings } from '@ant-design/pro-components';
import { SettingDrawer } from '@ant-design/pro-components';
import type { RequestConfig, RunTimeLayoutConfig } from '@umijs/max';
import { history, Link } from '@umijs/max';
import { Result } from 'antd';
import React from 'react';
import OrganizationSwitcher from '@/components/OrganizationSwitcher';
import { AvatarDropdown } from '@/components/RightContent/AvatarDropdown';
import { authServiceMe } from '@/services/roncin/authService';
import defaultSettings from '../config/defaultSettings';
import { errorConfig } from './requestErrorConfig';

const loginPath = '/user/login';
const layoutSettings = defaultSettings as Partial<LayoutSettings>;

export interface InitialState {
  settings?: Partial<LayoutSettings>;
  currentUser?: API.CurrentUser;
  fetchUserInfo?: () => Promise<API.CurrentUser | undefined>;
  settingDrawerOpen?: boolean;
}

export async function getInitialState(): Promise<InitialState> {
  const fetchUserInfo = async () => {
    const response = await authServiceMe({ skipErrorHandler: true });
    return response.data;
  };

  if (history.location.pathname === loginPath) {
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

export const layout: RunTimeLayoutConfig = ({
  initialState,
  setInitialState,
}) => ({
  menuItemRender: (item, dom) =>
    item.path ? (
      <Link to={item.path} prefetch>
        {dom}
      </Link>
    ) : (
      dom
    ),
  actionsRender: () => [<OrganizationSwitcher key="organization" />],
  avatarProps: {
    title:
      initialState?.currentUser?.displayName ??
      initialState?.currentUser?.username,
    render: (_, avatarChildren) => (
      <AvatarDropdown>{avatarChildren}</AvatarDropdown>
    ),
  },
  onPageChange: () => {
    if (!initialState?.currentUser && history.location.pathname !== loginPath) {
      history.replace(loginPath);
    }
  },
  unAccessible: <Result status="403" title="403" subTitle="无权访问此页面" />,
  childrenRender: (children) => (
    <>
      {children}
      <SettingDrawer
        disableUrlParams
        enableDarkTheme
        collapse={initialState?.settingDrawerOpen}
        onCollapseChange={(open) =>
          setInitialState((state) => ({
            ...state,
            settingDrawerOpen: open,
          }))
        }
        settings={initialState?.settings}
        onSettingChange={(settings) =>
          setInitialState((state) => ({ ...state, settings }))
        }
      />
    </>
  ),
  ...initialState?.settings,
});

export const request: RequestConfig = {
  baseURL: '',
  withCredentials: true,
  ...errorConfig,
};
