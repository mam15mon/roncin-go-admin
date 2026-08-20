import { ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Descriptions, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import React, { useRef } from 'react';
import { adminServiceListAuditLogs } from '@/services/roncin/adminService';

export default function AuditPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const columns: ProColumns<API.AdminAuditLog>[] = [
    { title: '时间范围', dataIndex: 'timeRange', valueType: 'dateRange', hideInTable: true },
    { title: '时间', dataIndex: 'createdAt', valueType: 'dateTime', width: 180, search: false },
    { title: '动作', dataIndex: 'action', width: 220 },
    { title: '用户', dataIndex: 'userId', width: 240, copyable: true },
    {
      title: '结果',
      dataIndex: 'result',
      width: 90,
      valueEnum: { success: { text: '成功' }, failure: { text: '失败' } },
      render: (_, record) => record.result === 'success' ? <Tag color="success">成功</Tag> : <Tag color="error">失败</Tag>,
    },
    { title: '资源', dataIndex: 'resourceId', width: 220, copyable: true },
    { title: '请求编号', dataIndex: 'requestId', width: 220, copyable: true },
    { title: '追踪编号', dataIndex: 'traceId', width: 220, copyable: true },
    { title: 'IP 地址', dataIndex: 'ipAddress', width: 150 },
    {
      title: '详情',
      dataIndex: 'details',
      width: 80,
      search: false,
      render: (_, record) => (
        <Typography.Paragraph
          ellipsis={{ rows: 2, expandable: true }}
          style={{ marginBottom: 0, maxWidth: 280 }}
        >
          <Descriptions size="small" column={1} bordered>
            {Object.entries(record.details ?? {}).map(([key, value]) => (
              <Descriptions.Item key={key} label={key}>{String(value)}</Descriptions.Item>
            ))}
          </Descriptions>
        </Typography.Paragraph>
      ),
    },
  ];

  return (
    <ProTable<API.AdminAuditLog>
      rowKey="id"
      actionRef={actionRef}
      columns={columns}
      pagination={{ defaultPageSize: 20, showSizeChanger: true }}
      request={async (params) => {
        const range = params.timeRange as [string, string] | undefined;
        const response = await adminServiceListAuditLogs({
          page: params.current,
          pageSize: params.pageSize,
          action: params.action,
          startTime: range?.[0] ? dayjs(range[0]).startOf('day').toISOString() : undefined,
          endTime: range?.[1] ? dayjs(range[1]).add(1, 'day').startOf('day').toISOString() : undefined,
        });
        return { data: response.data ?? [], success: response.success ?? true, total: response.total ?? 0 };
      }}
      toolBarRender={() => [
        <Button key="refresh" icon={<ReloadOutlined />} onClick={() => actionRef.current?.reload()}>
          刷新
        </Button>,
      ]}
      search={{ labelWidth: 'auto', defaultCollapsed: false }}
    />
  );
}
