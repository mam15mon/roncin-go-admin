import { useModel } from '@umijs/max';
import { App, Select } from 'antd';
import React, { startTransition, useState } from 'react';
import { authServiceSwitchOrganization } from '@/services/roncin/authService';

export default function OrganizationSwitcher() {
  const { initialState, setInitialState } = useModel('@@initialState');
  const { message } = App.useApp();
  const [loading, setLoading] = useState(false);
  const currentUser = initialState?.currentUser;
  const organizations = currentUser?.organizations ?? [];

  if (!currentUser?.currentOrganization || organizations.length < 2) {
    return null;
  }

  const handleChange = async (organizationId: string) => {
    setLoading(true);
    try {
      const response = await authServiceSwitchOrganization({ organizationId });
      if (!response.data) return;
      startTransition(() => {
        setInitialState((state) => ({
          ...state,
          currentUser: response.data,
        }));
      });
      message.success('已切换当前组织');
    } finally {
      setLoading(false);
    }
  };

  return (
    <Select
      aria-label="切换当前组织"
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
