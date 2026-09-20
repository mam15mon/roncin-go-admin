import { CheckOutlined, DownOutlined, SwapOutlined } from '@ant-design/icons';
import type { MenuProps } from 'antd';
import { App, Button, Spin } from 'antd';
import React, { useRef, useState } from 'react';
import { useInitialState } from '@/app/AppProvider';
import HeaderDropdown from '@/components/HeaderDropdown';
import { clearOrderMasterDataCache } from '@/features/orders/options';
import { history } from '@/router/history';
import { authServiceSwitchOrganization } from '@/services/roncin/authService';
import { confirmIfAnyTabDirty } from './layout/tabCloseGuard';

export default function OrganizationSwitcher() {
  const { initialState, setInitialState } = useInitialState();
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
  ) => {
    setSwitchingOrgName(organizationName);
    try {
      const response = await authServiceSwitchOrganization({ organizationId });
      if (!response.data) {
        message.error('组织切换失败，请稍后重试');
        return;
      }
      // 服务端已按新组织轮转会话凭据；响应 data 即新会话的权限主体，
      // 与 auth/me 同构。组织身份决定 workspace key；不得把这次更新降为
      // transition 后又先跳转，否则工作台可能短暂以旧组织身份挂载并发起旧组织请求。
      clearOrderMasterDataCache();
      setInitialState((state) => ({
        ...state,
        currentUser: response.data,
      }));
      history.replace('/welcome');
      message.success(`已切换至 ${organizationName}`);
    } catch {
      // 统一请求错误处理已负责展示失败原因；这里仅保证失败不改变当前工作区。
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
      () => {
        void switchOrganization(target.id as string, target.name ?? '');
      },
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
