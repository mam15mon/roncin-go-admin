import {
  EditOutlined,
  PlusOutlined,
  TeamOutlined,
} from '@ant-design/icons';
import { ProCard } from '@ant-design/pro-components';
import {
  Button,
  Card,
  Descriptions,
  Empty,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import React from 'react';
import {
  getChildOrganizationKind,
  getOrganizationKindMeta,
} from './types';

const { Text, Title } = Typography;

interface OrgDetailCardProps {
  selectedOrg?: API.AdminOrganization | null;
  parentOrgOfSelected?: API.AdminOrganization | null;
  directChildren: API.AdminOrganization[];
  totalDescendantCount: number;
  canCreate: boolean;
  canUpdate: boolean;
  onOpenCreateChild: (parent?: API.AdminOrganization | null) => void;
  onOpenEdit: (org?: API.AdminOrganization | null) => void;
  onSelectOrg: (id: string) => void;
}

export default function OrgDetailCard({
  selectedOrg,
  parentOrgOfSelected,
  directChildren,
  totalDescendantCount,
  canCreate,
  canUpdate,
  onOpenCreateChild,
  onOpenEdit,
  onSelectOrg,
}: OrgDetailCardProps) {
  return (
    <ProCard
      title={
        selectedOrg ? (
          <Space align="center" size={10}>
            <Title
              level={4}
              style={{ margin: 0, color: 'rgba(0, 0, 0, 0.88)' }}
            >
              {selectedOrg.name}
            </Title>
            <Tag
              variant="filled"
              style={{
                margin: 0,
                fontFamily: 'monospace',
                fontSize: 12,
                padding: '2px 8px',
                backgroundColor: '#fafafa',
                color: 'rgba(0, 0, 0, 0.65)',
              }}
            >
              {selectedOrg.code}
            </Tag>
            {selectedOrg.enabled ? (
              <Tag color="success">正常启用</Tag>
            ) : (
              <Tag color="default">已停用</Tag>
            )}
          </Space>
        ) : (
          '组织详细信息'
        )
      }
      extra={
        selectedOrg ? (
          <Space size={8}>
            {canCreate && getChildOrganizationKind(selectedOrg.kind) && (
              <Button
                icon={<PlusOutlined />}
                onClick={() => onOpenCreateChild(selectedOrg)}
              >
                新增
                {
                  getOrganizationKindMeta(
                    getChildOrganizationKind(selectedOrg.kind),
                  )?.label
                }
              </Button>
            )}
            {canUpdate && (
              <Button
                type="primary"
                icon={<EditOutlined />}
                onClick={() => onOpenEdit(selectedOrg)}
              >
                编辑组织
              </Button>
            )}
          </Space>
        ) : null
      }
      headerBordered
    >
      {selectedOrg ? (
        <Space vertical size={20} style={{ width: '100%' }}>
          <Descriptions
            bordered
            size="middle"
            column={{ xs: 1, sm: 2, lg: 3 }}
          >
            <Descriptions.Item label="组织编码">
              <Text copyable style={{ fontFamily: 'monospace' }}>
                {selectedOrg.code}
              </Text>
            </Descriptions.Item>
            <Descriptions.Item label="组织名称">
              <Text strong>{selectedOrg.name}</Text>
            </Descriptions.Item>
            <Descriptions.Item label="本币">
              <Tag color="blue">{selectedOrg.baseCurrency}</Tag>
              {selectedOrg.kind === 3 || selectedOrg.kind === 4 ? (
                <Text type="secondary">从所属公司继承</Text>
              ) : null}
            </Descriptions.Item>
            <Descriptions.Item label="上级组织">
              {parentOrgOfSelected ? (
                <Button
                  type="link"
                  size="small"
                  style={{ padding: 0, height: 'auto' }}
                  onClick={() => onSelectOrg(parentOrgOfSelected.id ?? '')}
                >
                  {parentOrgOfSelected.name} ({parentOrgOfSelected.code})
                </Button>
              ) : (
                <Text type="secondary">根组织（无上级）</Text>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="运营状态">
              {selectedOrg.enabled ? (
                <Tag color="success">正常启用</Tag>
              ) : (
                <Tag color="default">已停用</Tag>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="直属下级机构">
              <Text strong style={{ color: '#1677ff' }}>
                {directChildren.length}
              </Text>{' '}
              个
            </Descriptions.Item>
            <Descriptions.Item label="全部后代机构">
              <Text strong style={{ color: '#1677ff' }}>
                {totalDescendantCount}
              </Text>{' '}
              个
            </Descriptions.Item>
          </Descriptions>

          <Card
            title={
              <Space size={6}>
                <TeamOutlined style={{ color: '#1677ff' }} />
                <span style={{ fontSize: 14, fontWeight: 600 }}>
                  直属下级组织列表
                </span>
                <Tag
                  color="blue"
                  variant="filled"
                  style={{ margin: 0, fontSize: 11 }}
                >
                  {directChildren.length}
                </Tag>
              </Space>
            }
            size="small"
            extra={
              canCreate ? (
                <Button
                  type="link"
                  size="small"
                  icon={<PlusOutlined />}
                  onClick={() => onOpenCreateChild(selectedOrg)}
                >
                  添加下级组织
                </Button>
              ) : null
            }
          >
            <Table<API.AdminOrganization>
              rowKey="id"
              size="middle"
              pagination={
                directChildren.length > 5
                  ? { pageSize: 5, showSizeChanger: false }
                  : false
              }
              dataSource={directChildren}
              columns={[
                {
                  title: '组织名称',
                  dataIndex: 'name',
                  render: (name, record) => (
                    <Button
                      type="link"
                      size="small"
                      style={{
                        padding: 0,
                        height: 'auto',
                        fontWeight: 500,
                      }}
                      onClick={() => onSelectOrg(record.id ?? '')}
                    >
                      {name}
                    </Button>
                  ),
                },
                {
                  title: '组织编码',
                  dataIndex: 'code',
                  render: (code) => (
                    <Text
                      copyable
                      style={{ fontFamily: 'monospace', fontSize: 12 }}
                    >
                      {code}
                    </Text>
                  ),
                },
                {
                  title: '组织类型',
                  dataIndex: 'kind',
                  render: (kind) => {
                    const meta = getOrganizationKindMeta(kind);
                    return meta ? (
                      <Tag color={meta.color}>{meta.label}</Tag>
                    ) : (
                      '-'
                    );
                  },
                },
                {
                  title: '本币',
                  dataIndex: 'baseCurrency',
                  render: (cur) => <Tag color="blue">{cur}</Tag>,
                },
                {
                  title: '状态',
                  dataIndex: 'enabled',
                  render: (enabled) =>
                    enabled ? (
                      <Tag color="success">启用</Tag>
                    ) : (
                      <Tag color="default">停用</Tag>
                    ),
                },
                {
                  title: '操作',
                  key: 'action',
                  width: 140,
                  render: (_, record) => (
                    <Space size={8}>
                      {canCreate &&
                        getChildOrganizationKind(record.kind) && (
                          <Button
                            type="link"
                            size="small"
                            style={{ padding: 0 }}
                            onClick={() => onOpenCreateChild(record)}
                          >
                            新增下级
                          </Button>
                        )}
                      {canUpdate && (
                        <Button
                          type="link"
                          size="small"
                          style={{ padding: 0 }}
                          onClick={() => onOpenEdit(record)}
                        >
                          编辑
                        </Button>
                      )}
                    </Space>
                  ),
                },
              ]}
              locale={{
                emptyText: (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description="暂无直属下级组织"
                    style={{ margin: '16px 0' }}
                  />
                ),
              }}
            />
          </Card>
        </Space>
      ) : (
        <Empty
          image={Empty.PRESENTED_IMAGE_SIMPLE}
          description="请在左侧树中选择组织机构"
          style={{ margin: '80px 0' }}
        />
      )}
    </ProCard>
  );
}
