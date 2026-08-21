import { ClockCircleOutlined, LockOutlined, SettingOutlined, TeamOutlined, UserOutlined } from '@ant-design/icons';
import { PageContainer } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { Alert, Card, Tabs } from 'antd';
import React from 'react';
import OrganizationsPanel from './organizations';
import AuditPanel from './audit';
import BackgroundTasksPanel from './background-tasks';
import PermissionsPanel from './permissions';
import RolesPanel from './roles';
import UsersPanel from './users';

export default function Admin() {
  const access = useAccess();
  const items = [
    access.canManageOrganizations
      ? {
          key: 'organizations',
          label: '组织管理',
          icon: <TeamOutlined />,
          children: <OrganizationsPanel />,
        }
      : null,
    access.canManageUsers
      ? {
          key: 'users',
          label: '用户管理',
          icon: <UserOutlined />,
          children: <UsersPanel />,
        }
      : null,
    access.canManageRoles
      ? {
          key: 'roles',
          label: '角色管理',
          icon: <SettingOutlined />,
          children: <RolesPanel />,
        }
      : null,
    access.canReadAudit
      ? {
          key: 'audit',
          label: '审计日志',
          icon: <LockOutlined />,
          children: <AuditPanel />,
        }
      : null,
    access.canReadTasks
      ? {
          key: 'background-tasks',
          label: '后台任务',
          icon: <ClockCircleOutlined />,
          children: <BackgroundTasksPanel />,
        }
      : null,
    access.canManageRoles
      ? {
          key: 'permissions',
          label: '权限目录',
          icon: <LockOutlined />,
          children: <PermissionsPanel />,
        }
      : null,
  ].filter(Boolean) as { key: string; label: string; icon: React.ReactNode; children: React.ReactNode }[];

  return (
    <PageContainer
      title="系统管理"
      subTitle="维护当前组织的访问边界和管理员配置"
    >
      {items.length > 0 ? (
        <Card>
          <Tabs items={items} />
        </Card>
      ) : (
        <Alert
          showIcon
          type="warning"
          message="暂无可用的管理权限"
          description="请联系系统管理员为当前账号分配组织、用户或角色管理权限。"
        />
      )}
    </PageContainer>
  );
}
