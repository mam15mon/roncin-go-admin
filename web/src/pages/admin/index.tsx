import {
  AlertOutlined,
  ApartmentOutlined,
  ClockCircleOutlined,
  HistoryOutlined,
  KeyOutlined,
  NumberOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from '@ant-design/icons';
import React, { useMemo } from 'react';
import { useAccess } from '@/app/access';
import {
  type MultiTabCenterTabItem,
  MultiTabCenterTemplate,
} from '@/components/ui';
import { isWorkspaceTabVisible } from '@/router/workspaceTabs';
import AuditPanel from './audit';
import BackgroundTasksPanel from './background-tasks';
import AbnormalCasesPanel from './components/AbnormalCasesPanel';
import NumberRulesPanel from './components/NumberRulesPanel';
import OrganizationsPanel from './organizations';
import PermissionsPanel from './permissions';
import RolesPanel from './roles';
import UsersPanel from './users';

export default function Admin() {
  const access = useAccess();
  const tabItems: MultiTabCenterTabItem[] = useMemo(
    () => [
      {
        key: 'organizations',
        label: '组织架构',
        icon: <ApartmentOutlined />,
        visible: isWorkspaceTabVisible('/admin', 'organizations', access),
        tooltip: '维护组织层级、多法人公司/网点架构及部门归属',
        children: <OrganizationsPanel />,
      },
      {
        key: 'users',
        label: '用户管理',
        icon: <UserOutlined />,
        visible: isWorkspaceTabVisible('/admin', 'users', access),
        tooltip: '同步钉钉人员、分配组织角色并管理成员账号',
        children: <UsersPanel />,
      },
      {
        key: 'roles',
        label: '角色权限',
        icon: <SafetyCertificateOutlined />,
        visible: isWorkspaceTabVisible('/admin', 'roles', access),
        tooltip: '配置角色功能权限及角色成员绑定',
        children: <RolesPanel />,
      },
      {
        key: 'number-rules',
        label: '单据规则',
        icon: <NumberOutlined />,
        visible: isWorkspaceTabVisible('/admin', 'number-rules', access),
        tooltip: '配置业务订单号、账单号、结算单等核心单据编号生成规则',
        children: <NumberRulesPanel />,
      },
      {
        key: 'abnormal-cases',
        label: '业务异常',
        icon: <AlertOutlined />,
        visible: isWorkspaceTabVisible('/admin', 'abnormal-cases', access),
        tooltip: '监控与维护业务运行中的异常定义与告警配置',
        children: <AbnormalCasesPanel />,
      },
      {
        key: 'audit',
        label: '审计日志',
        icon: <HistoryOutlined />,
        visible: isWorkspaceTabVisible('/admin', 'audit', access),
        tooltip: '查看全平台业务操作轨迹、敏感数据变更与安全审计日志',
        children: <AuditPanel />,
      },
      {
        key: 'background-tasks',
        label: '后台任务',
        icon: <ClockCircleOutlined />,
        visible: isWorkspaceTabVisible('/admin', 'background-tasks', access),
        tooltip: '查看与管理异步批处理、报表导出及后台定时运行任务',
        children: <BackgroundTasksPanel />,
      },
      {
        key: 'permissions',
        label: '权限字典',
        icon: <KeyOutlined />,
        visible: isWorkspaceTabVisible('/admin', 'permissions', access),
        tooltip: '只读查阅后端 Manifest 权限字典清单与依赖拓扑',
        children: <PermissionsPanel />,
      },
    ],
    [access],
  );

  return (
    <MultiTabCenterTemplate
      title="系统管理"
      subTitle="统一管理组织架构、用户账号、角色权限、单据规则及审计日志"
      items={tabItems}
      defaultActiveKey="organizations"
      syncUrlQuery
    />
  );
}
