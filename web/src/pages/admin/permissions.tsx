import { ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Tag } from 'antd';
import React, { useRef } from 'react';
import { adminServiceListPermissions } from '@/services/roncin/adminService';

export default function PermissionsPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const columns: ProColumns<API.AdminPermission>[] = [
    { title: '权限码', dataIndex: 'key', width: 260, copyable: true },
    { title: '名称', dataIndex: 'name', width: 180 },
    { title: '分组', dataIndex: 'group', width: 140, render: (_, record) => <Tag>{record.group}</Tag> },
    { title: '说明', dataIndex: 'description', ellipsis: true },
  ];

  return (
    <ProTable<API.AdminPermission>
      rowKey="key"
      actionRef={actionRef}
      columns={columns}
      search={false}
      pagination={false}
      request={async () => {
        const response = await adminServiceListPermissions();
        return { data: response.data ?? [], success: response.success ?? true };
      }}
      toolBarRender={() => [
        <Button key="refresh" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>
          刷新
        </Button>,
      ]}
    />
  );
}
