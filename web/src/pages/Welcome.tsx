import {
  ApartmentOutlined,
  ContactsOutlined,
  DatabaseOutlined,
  KeyOutlined,
  OrderedListOutlined,
  RightOutlined,
  SafetyCertificateOutlined,
  SettingOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { PageContainer, ProCard } from '@ant-design/pro-components';
import { history, useModel } from '@umijs/max';
import { Avatar, Button, Descriptions, Empty, Space, Tag, Typography } from 'antd';
import React from 'react';

const { Text, Title, Paragraph } = Typography;

const quickLinks = [
  {
    title: '订单管理',
    desc: '全流程跟踪海运、空运及多式联运货代订单，协同单证、集装箱与异常处理',
    icon: <OrderedListOutlined style={{ fontSize: 24, color: '#1677ff' }} />,
    path: '/orders',
  },
  {
    title: '往来单位',
    desc: '统一维护客户、供应商、车队、船东及国外代理档案，管理银行账户与合同条款',
    icon: <ContactsOutlined style={{ fontSize: 24, color: '#52c41a' }} />,
    path: '/partners/customers',
  },
  {
    title: '主数据管理',
    desc: '配置港口、机场、箱型字典，定制单据自动编号序列与业务状态履约流程',
    icon: <DatabaseOutlined style={{ fontSize: 24, color: '#fa8c16' }} />,
    path: '/master-data',
  },
  {
    title: '系统与安全',
    desc: '维护多级组织架构、分配人员角色权限、审计追踪操作日志与后台异步任务',
    icon: <SettingOutlined style={{ fontSize: 24, color: '#722ed1' }} />,
    path: '/admin',
  },
];

export default function Welcome() {
  const { initialState } = useModel('@@initialState');
  const user = initialState?.currentUser;

  const displayName = user?.displayName || user?.username || '用户';
  const orgName = user?.currentOrganization?.name || '默认组织';
  const orgCode = user?.currentOrganization?.code || '-';

  return (
    <PageContainer
      title={
        <Space size={12} align="center">
          <Avatar
            size={44}
            style={{
              backgroundColor: '#1677ff',
              fontSize: 18,
              fontWeight: 600,
            }}
          >
            {displayName.charAt(0).toUpperCase()}
          </Avatar>
          <div>
            <Title level={4} style={{ margin: 0 }}>
              您好，{displayName}
            </Title>
            <Text type="secondary" style={{ fontSize: 13 }}>
              欢迎登录 Roncin 国际货代协同管理平台
            </Text>
          </div>
        </Space>
      }
      extra={
        <Space size={8}>
          <Tag icon={<ApartmentOutlined />} color="blue" style={{ padding: '4px 10px', fontSize: 12 }}>
            当前组织：{orgName}
          </Tag>
        </Space>
      }
    >
      <Space direction="vertical" size={16} style={{ width: '100%' }}>
        {/* Profile and System Boundary Overview */}
        <ProCard gutter={[16, 16]} wrap ghost>
          <ProCard
            colSpan={{ xs: 24, lg: 16 }}
            title={
              <Space size={8}>
                <UserOutlined style={{ color: '#1677ff' }} />
                <span>当前账号与组织上下文</span>
              </Space>
            }
            headerBordered
            variant="outlined"
          >
            <Descriptions column={{ xs: 1, sm: 2 }} size="middle">
              <Descriptions.Item label="用户登录名">
                <Text copyable style={{ fontFamily: 'monospace' }}>
                  {user?.username}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item label="显示名称">
                <Text strong>{user?.displayName || '-'}</Text>
              </Descriptions.Item>
              <Descriptions.Item label="所属组织机构">
                <Space size={6}>
                  <ApartmentOutlined style={{ color: '#1677ff' }} />
                  <Text strong>{orgName}</Text>
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="组织统一编码">
                <Text copyable style={{ fontFamily: 'monospace' }}>
                  {orgCode}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item label="数据授权范围" span={2}>
                {user?.roleScopes && user.roleScopes.length > 0 ? (
                  <Space wrap size={[6, 6]}>
                    {user.roleScopes.map((scope) => (
                      <Tag
                        key={`${scope.roleCode}:${scope.dataScope}`}
                        color="cyan"
                        bordered={false}
                        style={{ padding: '2px 8px', fontSize: 12 }}
                      >
                        <SafetyCertificateOutlined style={{ marginRight: 4 }} />
                        {scope.roleCode} · 数据范围: {scope.dataScope}
                      </Tag>
                    ))}
                  </Space>
                ) : (
                  <Text type="secondary">按系统默认规则隔离</Text>
                )}
              </Descriptions.Item>
            </Descriptions>
          </ProCard>

          <ProCard
            colSpan={{ xs: 24, lg: 8 }}
            title={
              <Space size={8}>
                <SafetyCertificateOutlined style={{ color: '#52c41a' }} />
                <span>系统运行与安全边界</span>
              </Space>
            }
            headerBordered
            variant="outlined"
          >
            <Paragraph style={{ fontSize: 13, color: '#64748b', lineHeight: 1.6, marginBottom: 12 }}>
              当前平台基于严格的组织树数据隔离与基于角色的功能访问控制（RBAC）。
              所有的单据创建、流转、核销均受当前组织边界和授权策略约束。
            </Paragraph>
            <Space direction="vertical" size={8} style={{ width: '100%' }}>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
                <Text type="secondary">功能权限授权数</Text>
                <Text strong>{user?.permissions?.length ?? 0} 项</Text>
              </div>
              <div style={{ display: 'flex', justifyContent: 'space-between', fontSize: 12 }}>
                <Text type="secondary">角色作用域数</Text>
                <Text strong>{user?.roleScopes?.length ?? 0} 个</Text>
              </div>
            </Space>
          </ProCard>
        </ProCard>

        {/* Quick Navigation Cards */}
        <ProCard
          title={
            <Space size={8}>
              <OrderedListOutlined style={{ color: '#1677ff' }} />
              <span>业务模块快速导航</span>
            </Space>
          }
          headerBordered
          variant="outlined"
        >
          <ProCard gutter={[16, 16]} wrap ghost>
            {quickLinks.map((item) => (
              <ProCard
                key={item.path}
                colSpan={{ xs: 24, sm: 12, lg: 6 }}
                variant="outlined"
                hoverable
                style={{ height: '100%' }}
                onClick={() => history.push(item.path)}
              >
                <Space direction="vertical" size={12} style={{ width: '100%' }}>
                  <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
                    <div style={{ padding: 8, borderRadius: 8, background: '#f8fafc' }}>
                      {item.icon}
                    </div>
                    <Button
                      type="link"
                      size="small"
                      style={{ padding: 0 }}
                      onClick={(e) => {
                        e.stopPropagation();
                        history.push(item.path);
                      }}
                    >
                      进入 <RightOutlined style={{ fontSize: 10 }} />
                    </Button>
                  </div>
                  <div>
                    <Text strong style={{ fontSize: 15, display: 'block', marginBottom: 4 }}>
                      {item.title}
                    </Text>
                    <Paragraph
                      type="secondary"
                      ellipsis={{ rows: 2 }}
                      style={{ fontSize: 12, marginBottom: 0, minHeight: 36 }}
                    >
                      {item.desc}
                    </Paragraph>
                  </div>
                </Space>
              </ProCard>
            ))}
          </ProCard>
        </ProCard>

        {/* Permissions Matrix */}
        <ProCard
          title={
            <Space size={8}>
              <KeyOutlined style={{ color: '#fa8c16' }} />
              <span>当前已授权功能权限</span>
              {user?.permissions?.length ? (
                <Tag color="blue" bordered={false}>
                  共 {user.permissions.length} 项
                </Tag>
              ) : null}
            </Space>
          }
          headerBordered
          variant="outlined"
        >
          {user?.permissions && user.permissions.length > 0 ? (
            <Space wrap size={[6, 8]}>
              {user.permissions.map((permission) => (
                <Tag
                  key={permission}
                  bordered={false}
                  style={{
                    margin: 0,
                    fontSize: 12,
                    padding: '2px 8px',
                    fontFamily: 'monospace',
                    backgroundColor: '#eff6ff',
                    color: '#1d4ed8',
                    border: '1px solid #dbeafe',
                  }}
                >
                  <KeyOutlined style={{ marginRight: 4, fontSize: 11 }} />
                  {permission}
                </Tag>
              ))}
            </Space>
          ) : (
            <Empty
              image={Empty.PRESENTED_IMAGE_SIMPLE}
              description="当前账号或所属角色尚未配置具体功能权限"
              style={{ margin: '24px 0' }}
            />
          )}
          <Paragraph type="secondary" style={{ marginTop: 16, marginBottom: 0, fontSize: 12 }}>
            权限清单由后端安全策略与当前组织角色矩阵动态计算生成，决定各菜单入口及按钮操作的可见与可用性。
          </Paragraph>
        </ProCard>
      </Space>
    </PageContainer>
  );
}
