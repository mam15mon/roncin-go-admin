import { EditOutlined, PlusOutlined } from '@ant-design/icons';
import {
  Button,
  Card,
  Descriptions,
  Drawer,
  Empty,
  Space,
  Table,
  Tag,
  Typography,
} from 'antd';
import React from 'react';
import { getChildOrganizationKind, getOrganizationKindMeta } from './types';

const { Text } = Typography;

type OrgDetailDrawerProps = {
  open: boolean;
  onClose: () => void;
  selectedOrg: API.AdminOrganization | null;
  parentOrg: API.AdminOrganization | null;
  directChildren: API.AdminOrganization[];
  totalDescendantCount: number;
  canCreate: boolean;
  canUpdate: boolean;
  onOpenCreateChild: (org: API.AdminOrganization) => void;
  onOpenEdit: (org: API.AdminOrganization) => void;
  onSelectNode: (id: string) => void;
};

export default function OrgDetailDrawer({
  open,
  onClose,
  selectedOrg,
  parentOrg,
  directChildren,
  totalDescendantCount,
  canCreate,
  canUpdate,
  onOpenCreateChild,
  onOpenEdit,
  onSelectNode,
}: OrgDetailDrawerProps) {
  return (
    <Drawer
      title={
        selectedOrg ? (
          <Space>
            <span>{selectedOrg.name}</span>
            <Tag
              color={selectedOrg.enabled ? 'success' : 'default'}
              variant="filled"
            >
              {selectedOrg.enabled ? '正常启用' : '已停用'}
            </Tag>
          </Space>
        ) : (
          '组织详情'
        )
      }
      open={open}
      onClose={onClose}
      size={420}
      extra={
        selectedOrg && (
          <Space>
            {canCreate && getChildOrganizationKind(selectedOrg.kind) && (
              <Button
                size="small"
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
                size="small"
                icon={<EditOutlined />}
                onClick={() => onOpenEdit(selectedOrg)}
              >
                编辑
              </Button>
            )}
          </Space>
        )
      }
    >
      {selectedOrg ? (
        <Space vertical size={16} style={{ width: '100%' }}>
          <Descriptions column={1} bordered size="small">
            <Descriptions.Item label="节点类型">
              {getOrganizationKindMeta(selectedOrg.kind)?.label}
            </Descriptions.Item>
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
            <Descriptions.Item label="所属上级">
              {parentOrg ? (
                <span>
                  {parentOrg.name} ({parentOrg.code})
                </span>
              ) : (
                <Text type="secondary">根组织（无上级）</Text>
              )}
            </Descriptions.Item>
            <Descriptions.Item label="直属下级">
              <Text strong style={{ color: '#1677ff' }}>
                {directChildren.length}
              </Text>{' '}
              个组织
            </Descriptions.Item>
            <Descriptions.Item label="全部后代">
              <Text strong style={{ color: '#1677ff' }}>
                {totalDescendantCount}
              </Text>{' '}
              个组织
            </Descriptions.Item>
          </Descriptions>

          <Card
            size="small"
            title={`直属下级组织 (${directChildren.length})`}
            styles={{ body: { padding: 0 } }}
          >
            <Table<API.AdminOrganization>
              rowKey="id"
              size="small"
              pagination={false}
              dataSource={directChildren}
              columns={[
                {
                  title: '名称',
                  dataIndex: 'name',
                  render: (name, record) => (
                    <Button
                      type="link"
                      size="small"
                      style={{ padding: 0 }}
                      onClick={() => onSelectNode(record.id ?? '')}
                    >
                      {name}
                    </Button>
                  ),
                },
                {
                  title: '编码',
                  dataIndex: 'code',
                  render: (code) => (
                    <Text style={{ fontFamily: 'monospace', fontSize: 11 }}>
                      {code}
                    </Text>
                  ),
                },
              ]}
              locale={{
                emptyText: (
                  <Empty
                    image={Empty.PRESENTED_IMAGE_SIMPLE}
                    description="无直属下级"
                    style={{ margin: '12px 0' }}
                  />
                ),
              }}
            />
          </Card>
        </Space>
      ) : null}
    </Drawer>
  );
}
