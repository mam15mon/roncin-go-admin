import { CheckOutlined, DownOutlined, SwapOutlined } from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { App, Button, Spin } from 'antd';
import React, { useRef, useState } from 'react';
import access from '@/access';
import { useInitialState } from '@/app/AppProvider';
import HeaderDropdown from '@/components/HeaderDropdown';
import { history } from '@/router/history';
import { resolveOrganizationSwitchDestination } from '@/router/organizationSwitchDestination';
import { authServiceSwitchOrganization } from '@/services/roncin/authService';
import { confirmIfAnyTabDirty } from './layout/tabCloseGuard';

export type OrganizationSwitcherProps = {
  onSwitchWorkspace: (
    user: API.CurrentUser,
    destination: string,
  ) => Promise<void>;
};

export default function OrganizationSwitcher({
  onSwitchWorkspace,
}: OrganizationSwitcherProps) {
  const { initialState } = useInitialState();
  const { message } = App.useApp();
  const [switchingOrgName, setSwitchingOrgName] = useState<string>();
  const switchInProgressRef = useRef(false);
  const currentUser = initialState?.currentUser;
  const organizations = currentUser?.organizations ?? [];
  const currentOrganization = currentUser?.currentOrganization;

  if (!currentOrganization || organizations.length < 2) {
    return null;
  }

  const switchOrganization = async (
    organizationId: string,
    organizationName: string,
  ): Promise<void> => {
    setSwitchingOrgName(organizationName);
    try {
      const response = await authServiceSwitchOrganization({ organizationId });
      if (!response || response.success === false) return;
      if (!response.data?.id) {
        message.error('组织切换失败，请稍后重试');
        return;
      }
      const destination = resolveOrganizationSwitchDestination(
        history.location,
        access({ currentUser: response.data }),
      );
      await onSwitchWorkspace(response.data, destination.path);
      if (destination.accessDenied) {
        message.info('当前组织无权访问该功能，已返回工作台');
      }
      message.success(`已切换至 ${organizationName}`);
    } catch {
      // 接口失败由统一请求层提示，原工作区保持；成功后的导航异常由布局边界处理。
    } finally {
      setSwitchingOrgName(undefined);
      switchInProgressRef.current = false;
    }
  };

  const handleMenuClick: MenuProps['onClick'] = ({ key }) => {
    if (switchInProgressRef.current || switchingOrgName) {
      return;
    }
    const target = organizations.find(
      (organization) => organization.id === key,
    );
    if (!target?.id || target.id === currentOrganization.id) {
      return;
    }

    switchInProgressRef.current = true;
    confirmIfAnyTabDirty(
      () => switchOrganization(target.id as string, target.name ?? ''),
      undefined,
      () => {
        switchInProgressRef.current = false;
      },
    );
  };

  // 当前组织置首并打勾置灰（不可重复选择），其余按候选顺序排列。
  const orderedOrganizations = [
    currentOrganization,
    ...organizations.filter(
      (organization) => organization.id !== currentOrganization.id,
    ),
  ];

  const menuItems: MenuProps['items'] = orderedOrganizations.map(
    (organization) => {
      const isCurrent = organization.id === currentOrganization.id;
      return {
        key: organization.id ?? '',
        disabled: isCurrent,
        label: (
          <span
            style={{
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'space-between',
              gap: 16,
              minWidth: 200,
            }}
          >
            <span
              style={{
                display: 'inline-flex',
                alignItems: 'center',
                gap: 8,
                minWidth: 0,
              }}
            >
              {isCurrent && (
                <CheckOutlined style={{ color: '#1677ff', fontSize: 12 }} />
              )}
              <span
                style={{
                  overflow: 'hidden',
                  textOverflow: 'ellipsis',
                  whiteSpace: 'nowrap',
                }}
              >
                {organization.name}
              </span>
            </span>
            <span style={{ color: '#8c8c8c', fontSize: 12, flexShrink: 0 }}>
              {organization.code}
            </span>
          </span>
        ),
      };
    },
  );

  return (
    <>
      <HeaderDropdown
        keyboardAccessible
        placement="bottomRight"
        menu={{ items: menuItems, onClick: handleMenuClick }}
      >
        <Button
          type="text"
          size="small"
          className="roncin-header-menu-btn"
          aria-label="切换当前组织"
          title="切换当前组织"
          disabled={Boolean(switchingOrgName)}
        >
          <SwapOutlined className="roncin-header-menu-icon" />
          <span
            style={{
              maxWidth: 120,
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
            }}
          >
            {currentOrganization.name}
          </span>
          <DownOutlined className="roncin-header-menu-icon" />
        </Button>
      </HeaderDropdown>
      <Spin
        spinning={Boolean(switchingOrgName)}
        fullscreen
        description={
          switchingOrgName ? `正在切换至 ${switchingOrgName}…` : undefined
        }
      />
    </>
  );
}
