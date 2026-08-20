import { PageContainer } from '@ant-design/pro-components';
import { useModel } from '@umijs/max';
import { Card, Descriptions, Empty, Space, Tag, Typography } from 'antd';
import React from 'react';

export default function Welcome() {
  const { initialState } = useModel('@@initialState');
  const user = initialState?.currentUser;

  return (
    <PageContainer title="工作台" subTitle="货代业务模块将从这里逐步迁入">
      <Card title={`欢迎，${user?.displayName ?? user?.username ?? ''}`}>
        <Descriptions column={{ xs: 1, sm: 2 }}>
          <Descriptions.Item label="当前组织">
            {user?.currentOrganization?.name}
          </Descriptions.Item>
          <Descriptions.Item label="组织编码">
            {user?.currentOrganization?.code}
          </Descriptions.Item>
          <Descriptions.Item label="用户名">{user?.username}</Descriptions.Item>
          <Descriptions.Item label="数据范围">
            <Space wrap>
              {user?.roleScopes?.map((scope) => (
                <Tag key={`${scope.roleCode}:${scope.dataScope}`}>
                  {scope.roleCode} · {scope.dataScope}
                </Tag>
              ))}
            </Space>
          </Descriptions.Item>
        </Descriptions>
      </Card>
      <Card title="已授权能力" style={{ marginTop: 16 }}>
        {user?.permissions?.length ? (
          <Space wrap>
            {user.permissions.map((permission) => (
              <Tag color="blue" key={permission}>
                {permission}
              </Tag>
            ))}
          </Space>
        ) : (
          <Empty description="当前角色尚未配置权限" />
        )}
        <Typography.Paragraph type="secondary" style={{ marginTop: 16 }}>
          权限来自后端 Manifest 与当前组织角色，前端只负责页面和按钮显隐。
        </Typography.Paragraph>
      </Card>
    </PageContainer>
  );
}
