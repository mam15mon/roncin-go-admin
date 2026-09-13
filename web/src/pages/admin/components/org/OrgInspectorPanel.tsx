import {
  ApartmentOutlined,
  CloseOutlined,
  EditOutlined,
  PlusOutlined,
} from '@ant-design/icons';
import {
  Button,
  Card,
  Descriptions,
  Empty,
  Space,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import React from 'react';
import { getChildOrganizationKind, getOrganizationKindMeta } from './types';

const { Text } = Typography;

export type OrgInspectorPanelProps = {
  open: boolean;
  onClose: () => void;
  selectedOrg?: API.AdminOrganization | null;
  parentOrg?: API.AdminOrganization | null;
  directChildren: API.AdminOrganization[];
  totalDescendantCount: number;
  canCreate: boolean;
  canUpdate: boolean;
  onOpenCreateChild: (org: API.AdminOrganization) => void;
  onOpenEdit: (org: API.AdminOrganization) => void;
  onSelectNode: (id: string) => void;
};

export default function OrgInspectorPanel({
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
}: OrgInspectorPanelProps) {
  if (!open) return null;

  return (
    <div
      data-testid="org-inspector-panel"
      onMouseDown={(e) => e.stopPropagation()}
      onWheel={(e) => e.stopPropagation()}
      style={{
        position: 'absolute',
        top: 12,
        right: 12,
        bottom: 12,
        width: 380,
        backgroundColor: '#ffffff',
        borderRadius: 10,
        boxShadow:
          '0 6px 20px rgba(0, 0, 0, 0.08), 0 1px 4px rgba(0, 0, 0, 0.04)',
        border: '1px solid #e2e8f0',
        display: 'flex',
        flexDirection: 'column',
        zIndex: 20,
        overflow: 'hidden',
      }}
    >
      {/* Header */}
      <div
        style={{
          padding: '12px 14px',
          borderBottom: '1px solid #f1f5f9',
          display: 'flex',
          alignItems: 'center',
          justifyContent: 'space-between',
          backgroundColor: '#fafbfc',
        }}
      >
        <Space size={8} style={{ minWidth: 0, flex: 1 }}>
          <ApartmentOutlined style={{ color: '#1677ff', fontSize: 16 }} />
          <div
            style={{
              fontWeight: 600,
              fontSize: 14,
              color: 'rgba(0, 0, 0, 0.88)',
              overflow: 'hidden',
              textOverflow: 'ellipsis',
              whiteSpace: 'nowrap',
              maxWidth: 150,
            }}
            title={selectedOrg?.name}
          >
            {selectedOrg?.name || '组织详情'}
          </div>
          {selectedOrg && (
            <Tag
              color={selectedOrg.enabled ? 'success' : 'default'}
              variant="filled"
              style={{ margin: 0, fontSize: 11 }}
            >
              {selectedOrg.enabled ? '启用' : '停用'}
            </Tag>
          )}
        </Space>

        <Space size={4}>
          {selectedOrg &&
            canCreate &&
            getChildOrganizationKind(selectedOrg.kind) && (
              <Tooltip
                title={`新增${
                  getOrganizationKindMeta(
                    getChildOrganizationKind(selectedOrg.kind),
                  )?.label
                }`}
              >
                <Button
                  size="small"
                  type="text"
                  aria-label={`新增${
                    getOrganizationKindMeta(
                      getChildOrganizationKind(selectedOrg.kind),
                    )?.label
                  }`}
                  icon={<PlusOutlined />}
                  onClick={() => onOpenCreateChild(selectedOrg)}
                />
              </Tooltip>
            )}
          {selectedOrg && canUpdate && (
            <Tooltip title="编辑组织">
              <Button
                size="small"
                type="text"
                aria-label="编辑组织"
                icon={<EditOutlined />}
                onClick={() => onOpenEdit(selectedOrg)}
              />
            </Tooltip>
          )}
          <Tooltip title="收起面板">
            <Button
              size="small"
              type="text"
              aria-label="收起面板"
              icon={<CloseOutlined />}
              onClick={onClose}
            />
          </Tooltip>
        </Space>
      </div>

      {/* Body Content */}
      <div
        style={{
          flex: 1,
          overflowY: 'auto',
          padding: '14px',
          display: 'flex',
          flexDirection: 'column',
          gap: 14,
        }}
      >
        {selectedOrg ? (
          <>
            {/* Core Info Descriptions */}
            <Descriptions
              column={1}
              size="small"
              bordered
              styles={{
                label: { width: 90, backgroundColor: '#f8fafc', fontSize: 12 },
                content: { fontSize: 12 },
              }}
            >
              <Descriptions.Item label="节点类型">
                <Space size={6}>
                  <Tag
                    color={getOrganizationKindMeta(selectedOrg.kind)?.color}
                    variant="filled"
                    style={{ margin: 0 }}
                  >
                    {getOrganizationKindMeta(selectedOrg.kind)?.label}
                  </Tag>
                </Space>
              </Descriptions.Item>
              <Descriptions.Item label="组织编码">
                <Text copyable style={{ fontFamily: 'monospace' }}>
                  {selectedOrg.code}
                </Text>
              </Descriptions.Item>
              <Descriptions.Item label="本位币种">
                <Tag color="blue">{selectedOrg.baseCurrency || '-'}</Tag>
                {(selectedOrg.kind === 3 || selectedOrg.kind === 4) && (
                  <Text type="secondary" style={{ fontSize: 11 }}>
                    （从所属公司继承）
                  </Text>
                )}
              </Descriptions.Item>
              <Descriptions.Item label="所属上级">
                {parentOrg ? (
                  <Button
                    type="link"
                    size="small"
                    style={{ padding: 0, height: 'auto', fontSize: 12 }}
                    onClick={() => onSelectNode(parentOrg.id ?? '')}
                  >
                    {parentOrg.name} ({parentOrg.code})
                  </Button>
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

            {/* Direct Children Organizations Card */}
            <Card
              size="small"
              title={
                <span style={{ fontSize: 13, fontWeight: 600 }}>
                  直属下级组织 ({directChildren.length})
                </span>
              }
              styles={{ body: { padding: 0 } }}
              style={{ border: '1px solid #f1f5f9' }}
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
                        style={{ padding: 0, height: 'auto', fontSize: 12 }}
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
                  {
                    title: '类型',
                    dataIndex: 'kind',
                    width: 70,
                    render: (kind) => {
                      const meta = getOrganizationKindMeta(kind);
                      return meta ? (
                        <Tag
                          color={meta.color}
                          variant="filled"
                          style={{ margin: 0, fontSize: 10, padding: '0 4px' }}
                        >
                          {meta.label}
                        </Tag>
                      ) : null;
                    },
                  },
                ]}
                locale={{
                  emptyText: (
                    <Empty
                      image={Empty.PRESENTED_IMAGE_SIMPLE}
                      description="无直属下级组织"
                      style={{ margin: '14px 0' }}
                    />
                  ),
                }}
              />
            </Card>

            {/* Helpful Tip */}
            <div
              style={{
                marginTop: 'auto',
                padding: '8px 10px',
                backgroundColor: '#f8fafc',
                borderRadius: 6,
                border: '1px solid #f1f5f9',
              }}
            >
              <Text type="secondary" style={{ fontSize: 11 }}>
                💡
                提示：点击左侧架构树中任意节点可无缝切换查看；支持拖拽画布与滚轮缩放。
              </Text>
            </div>
          </>
        ) : (
          <Empty
            image={Empty.PRESENTED_IMAGE_SIMPLE}
            description="请在左侧架构图中选择节点"
            style={{ margin: '60px 0' }}
          />
        )}
      </div>
    </div>
  );
}
