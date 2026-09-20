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
import React, { useEffect, useMemo } from 'react';
import { useLocation } from 'react-router';
import { useAccess } from '@/app/access';
import {
  type MultiTabCenterTabItem,
  MultiTabCenterTemplate,
} from '@/components/ui';
import { history } from '@/router/history';
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
  const location = useLocation();

  // 兼容旧路径重定向（原顶级 dingtalk-registrations / dingtalk-invitations 平滑收敛至用户管理子页签）
  useEffect(() => {
    const params = new URLSearchParams(location.search);
    const queryTab = params.get('tab');
    if (queryTab === 'dingtalk-registrations') {
      params.set('tab', 'users');
      params.set('subTab', 'registrations');
      history.replace(`${location.pathname}?${params.toString()}`);
    } else if (queryTab === 'dingtalk-invitations') {
      params.set('tab', 'users');
      params.set('subTab', 'invitations');
      history.replace(`${location.pathname}?${params.toString()}`);
    }
  }, [location.search, location.pathname]);

  const tabItems: MultiTabCenterTabItem[] = useMemo(
    () => [
      {
        key: 'organizations',
        label: '组织架构',
        icon: <ApartmentOutlined />,
        visible: access.canReadOrganizations,
        tooltip: '维护组织层级、多法人公司/网点架构及部门归属',
        children: <OrganizationsPanel />,
      },
      {
        key: 'users',
        label: '用户管理',
        icon: <UserOutlined />,
        visible: access.canReadUsers || access.canManageDingTalkInvitations,
        tooltip: '管理系统成员账号、审批钉钉注册申请及生成入职邀请码',
        children: <UsersPanel />,
      },
      {
        key: 'roles',
        label: '角色权限',
        icon: <SafetyCertificateOutlined />,
        visible: access.canReadRoles,
        tooltip: '配置角色功能权限树、数据范围权限及角色成员绑定',
        children: <RolesPanel />,
      },
      {
        key: 'number-rules',
        label: '单据规则',
        icon: <NumberOutlined />,
        visible: access.canReadMasterDataNumberRules,
        tooltip: '配置业务订单号、账单号、结算单等核心单据编号生成规则',
        children: <NumberRulesPanel />,
      },
      {
        key: 'abnormal-cases',
        label: '业务异常',
        icon: <AlertOutlined />,
        visible: access.canReadMasterDataItems,
        tooltip: '监控与维护业务运行中的异常定义与告警配置',
        children: <AbnormalCasesPanel />,
      },
      {
        key: 'audit',
        label: '审计日志',
        icon: <HistoryOutlined />,
        visible: access.canReadAudit,
        tooltip: '查看全平台业务操作轨迹、敏感数据变更与安全审计日志',
        children: <AuditPanel />,
      },
      {
        key: 'background-tasks',
        label: '后台任务',
        icon: <ClockCircleOutlined />,
        visible: access.canReadTasks,
        tooltip: '查看与管理异步批处理、报表导出及后台定时运行任务',
        children: <BackgroundTasksPanel />,
      },
      {
        key: 'permissions',
        label: '权限字典',
        icon: <KeyOutlined />,
        visible: access.canReadPermissions,
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
