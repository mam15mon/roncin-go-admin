import {
  ApartmentOutlined,
  SafetyCertificateOutlined,
  UserOutlined,
} from '@ant-design/icons';
import { PageContainer, ProCard } from '@ant-design/pro-components';
import { useModel } from '@umijs/max';
import { Avatar, Descriptions, Space, Tag, Typography } from 'antd';
import React from 'react';

const { Text, Title, Paragraph } = Typography;

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
            src={user?.avatarUrl}
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
      <Space vertical size={16} style={{ width: '100%' }}>
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
                        variant="filled"
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
            <Space vertical size={8} style={{ width: '100%' }}>
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
      </Space>
    </PageContainer>
  );
}
