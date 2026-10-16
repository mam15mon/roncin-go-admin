import { KeyOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { Button, Space, Tag, Typography } from 'antd';
import React, { useRef, useState } from 'react';
import { ResponsiveProTable as ProTable } from '@/components/ui';
import { useColumnSettings } from '@/components/ui/column-settings';
import { adminServiceListPermissions } from '@/services/roncin/adminService';
import { toTableRequest } from '@/utils/api';
import { getPermissionGroupSegments } from './components/permissionGrouping';

const { Text } = Typography;

type PermissionSearch = { keyword?: string; module?: string };

export default function PermissionsPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const [moduleOptions, setModuleOptions] = useState<
    { label: string; value: string }[]
  >([]);
  const columns: ProColumns<API.AdminPermission>[] = [
    {
      title: '关键词',
      dataIndex: 'keyword',
      hideInTable: true,
      fieldProps: {
        placeholder: '搜索功能名称、权限码、业务分组或说明',
        allowClear: true,
      },
    },
    {
      title: '业务模块',
      dataIndex: 'module',
      hideInTable: true,
      valueType: 'select',
      fieldProps: {
        placeholder: '全部模块',
        allowClear: true,
        options: moduleOptions,
      },
    },
    {
      title: '权限标识码',
      dataIndex: 'key',
      search: false,
      width: 280,
      copyable: true,
      render: (_, record) => (
        <Text
          style={{ fontFamily: 'monospace', fontSize: 12, fontWeight: 500 }}
        >
          {record.key}
        </Text>
      ),
    },
    {
      title: '功能名称',
      dataIndex: 'name',
      search: false,
      width: 200,
      render: (_, record) => <Text strong>{record.name}</Text>,
    },
    {
      title: '业务分组',
      dataIndex: 'group',
      search: false,
      width: 260,
      render: (_, record) => (
        <Tag color="blue" variant="filled" style={{ padding: '2px 8px' }}>
          {getPermissionGroupSegments(record.group).join(' → ')}
        </Tag>
      ),
    },
    {
      title: '权限说明',
      dataIndex: 'description',
      search: false,
      ellipsis: true,
      render: (_, record) =>
        record.description ? (
          <span>{record.description}</span>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
  ];

  const columnSettings = useColumnSettings<ProColumns<API.AdminPermission>>({
    tableKey: 'admin:permissions',
    columns,
  });

  return (
    <ProTable<API.AdminPermission, PermissionSearch>
      headerTitle={
        <Space size={8}>
          <KeyOutlined style={{ color: '#1677ff' }} />
          <span>系统功能权限字典清单</span>
        </Space>
      }
      rowKey="key"
      actionRef={actionRef}
      columns={columnSettings.columns}
      bordered
      search={{ labelWidth: 'auto', defaultCollapsed: false }}
      pagination={false}
      locale={{ emptyText: '未找到匹配的权限，请调整关键词或业务模块' }}
      request={async (params) => {
        const response = await adminServiceListPermissions();
        if (!response) {
          return { data: [], success: false };
        }
        const result = toTableRequest(response);
        const modules = [
          ...new Set(
            result.data.map(
              (permission) => getPermissionGroupSegments(permission.group)[0],
            ),
          ),
        ];
        setModuleOptions(modules.map((name) => ({ label: name, value: name })));
        const keyword = (params.keyword ?? '').trim().toLowerCase();
        const data = result.data.filter((permission) => {
          const segments = getPermissionGroupSegments(permission.group);
          return (
            (!params.module || segments[0] === params.module) &&
            [
              permission.name,
              permission.key,
              permission.group,
              segments.join(' → '),
              permission.description,
            ].some((value) => value?.toLowerCase().includes(keyword))
          );
        });
        return { ...result, data, total: data.length };
      }}
      options={{ reload: true, density: true, setting: false }}
      toolBarRender={() => [
        columnSettings.entry,
        <Button
          key="refresh"
          icon={<ReloadOutlined />}
          onClick={() => actionRef.current?.reload()}
        >
          刷新
        </Button>,
      ]}
    />
  );
}
