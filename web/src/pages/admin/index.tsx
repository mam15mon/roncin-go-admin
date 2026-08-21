import {
  ApartmentOutlined,
  ClockCircleOutlined,
  HistoryOutlined,
  KeyOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { Alert, Space } from 'antd';
import React, { useMemo, useState } from 'react';
import AuditPanel from './audit';
import BackgroundTasksPanel from './background-tasks';
import OrganizationsPanel from './organizations';
import PermissionsPanel from './permissions';
import RolesPanel from './roles';
import UsersPanel from './users';

export default function Admin() {
  const access = useAccess();

  const tabItems = useMemo(
    () =>
      [
        access.canManageOrganizations
          ? {
              key: 'organizations',
              tab: (
                <Space size={6}>
                  <ApartmentOutlined />
                  <span>组织架构</span>
                </Space>
              ),
              children: <OrganizationsPanel />,
            }
          : null,
        access.canManageUsers
          ? {
              key: 'users',
              tab: (
                <Space size={6}>
                  <UserOutlined />
                  <span>用户管理</span>
                </Space>
              ),
              children: <UsersPanel />,
            }
          : null,
        access.canManageRoles
          ? {
              key: 'roles',
              tab: (
                <Space size={6}>
                  <SafetyCertificateOutlined />
                  <span>角色权限</span>
                </Space>
              ),
              children: <RolesPanel />,
            }
          : null,
        access.canReadAudit
          ? {
              key: 'audit',
              tab: (
                <Space size={6}>
                  <HistoryOutlined />
                  <span>审计日志</span>
                </Space>
              ),
              children: <AuditPanel />,
            }
          : null,
        access.canReadTasks
          ? {
              key: 'background-tasks',
              tab: (
                <Space size={6}>
                  <ClockCircleOutlined />
                  <span>后台任务</span>
                </Space>
              ),
              children: <BackgroundTasksPanel />,
            }
          : null,
        access.canManageRoles
          ? {
              key: 'permissions',
              tab: (
                <Space size={6}>
                  <KeyOutlined />
                  <span>权限字典</span>
                </Space>
              ),
              children: <PermissionsPanel />,
            }
          : null,
      ].filter(Boolean) as {
        key: string;
        tab: React.ReactNode;
        children: React.ReactNode;
      }[],
    [access],
  );

  const [activeTab, setActiveTab] = useState<string>(
    () => tabItems[0]?.key || 'organizations',
  );

  const currentTabKey = tabItems.some((item) => item.key === activeTab)
    ? activeTab
    : (tabItems[0]?.key ?? '');

  const activeContent = tabItems.find((item) => item.key === currentTabKey)
    ?.children;

  return (
    <PageContainer
      title="系统管理"
      subTitle="维护多级组织架构、成员账号体系、角色权限策略、安全审计日志与异步任务"
      tabList={tabItems.map((item) => ({
        key: item.key,
        tab: item.tab,
      }))}
      tabActiveKey={currentTabKey}
      onTabChange={(key) => setActiveTab(key)}
    >
      {tabItems.length > 0 ? (
        activeContent
      ) : (
        <Alert
          showIcon
          type="warning"
          message="暂无可用的管理权限"
          description="请联系系统管理员为当前账号分配组织、用户或角色管理等相应权限。"
        />
      )}
    </PageContainer>
  );
}
