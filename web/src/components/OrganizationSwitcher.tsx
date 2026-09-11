import { history, useModel } from '@umijs/max';
import { App, Select } from 'antd';
import React, { useRef, useState } from 'react';
import { authServiceSwitchOrganization } from '@/services/roncin/authService';
import { confirmIfAnyTabDirty } from './layout/tabCloseGuard';

export default function OrganizationSwitcher() {
  const { initialState, setInitialState } = useModel('@@initialState');
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const switchInProgressRef = useRef(false);
  const currentUser = initialState?.currentUser;
  const organizations = currentUser?.organizations ?? [];

  if (!currentUser?.currentOrganization || organizations.length < 2) {
    return null;
  }

  const switchOrganization = async (organizationId: string) => {
    setLoading(true);
    try {
      const response = await authServiceSwitchOrganization({ organizationId });
      if (!response.data) {
        message.error('组织切换失败，请稍后重试');
        return;
      }
      // 组织身份决定 workspace key；不得把这次更新降为 transition 后又先跳转，
      // 否则工作台可能短暂以旧组织身份挂载并发起旧组织请求。
      setInitialState((state) => ({
        ...state,
        currentUser: response.data,
      }));
      history.replace('/welcome');
      message.success('已切换当前组织');
    } catch {
      // 统一请求错误处理已负责展示失败原因；这里仅保证失败不改变当前工作区。
    } finally {
      setLoading(false);
      switchInProgressRef.current = false;
    }
  };

  const handleChange = (organizationId: string) => {
    if (
      switchInProgressRef.current ||
      loading ||
      organizationId === currentUser.currentOrganization?.id
    ) {
      return;
    }

    switchInProgressRef.current = true;
    confirmIfAnyTabDirty(
      () => {
        void switchOrganization(organizationId);
      },
      undefined,
      () => {
        switchInProgressRef.current = false;
      },
    );
  };

  return (
    <Select
      aria-label="切换当前组织"
      disabled={loading}
      loading={loading}
      value={currentUser.currentOrganization.id}
      options={organizations.map((organization) => ({
        value: organization.id,
        label: `${organization.code} · ${organization.name}`,
      }))}
      onChange={handleChange}
      showSearch={{ optionFilterProp: 'label' }}
      style={{ minWidth: 180 }}
    />
  );
}
