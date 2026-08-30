import { KeyOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Space, Tag, Typography } from 'antd';
import React, { useRef } from 'react';
import { adminServiceListPermissions } from '@/services/roncin/adminService';
import { toTableRequest } from '@/utils/api';

const { Text } = Typography;

export default function PermissionsPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const columns: ProColumns<API.AdminPermission>[] = [
    {
      title: '权限标识码',
      dataIndex: 'key',
      width: 280,
      copyable: true,
      render: (_, record) => (
        <Text style={{ fontFamily: 'monospace', fontSize: 12, fontWeight: 500 }}>
          {record.key}
        </Text>
      ),
    },
    {
      title: '功能名称',
      dataIndex: 'name',
      width: 200,
      render: (_, record) => <Text strong>{record.name}</Text>,
    },
    {
      title: '业务分组',
      dataIndex: 'group',
      width: 160,
      render: (_, record) => (
        <Tag color="blue" variant="filled" style={{ padding: '2px 8px' }}>
          {record.group || '通用'}
        </Tag>
      ),
    },
    {
      title: '权限说明',
      dataIndex: 'description',
      ellipsis: true,
      render: (_, record) =>
        record.description ? (
          <span>{record.description}</span>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
  ];

  return (
    <ProTable<API.AdminPermission>
      headerTitle={
        <Space size={8}>
          <KeyOutlined style={{ color: '#1677ff' }} />
          <span>系统功能权限字典清单</span>
        </Space>
      }
      rowKey="key"
      actionRef={actionRef}
      columns={columns}
      bordered
      search={false}
      pagination={false}
      request={async () => {
        const response = await adminServiceListPermissions();
        return toTableRequest(response);
      }}
      toolBarRender={() => [
        <Button key="refresh" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>
          刷新
        </Button>,
      ]}
    />
  );
}
