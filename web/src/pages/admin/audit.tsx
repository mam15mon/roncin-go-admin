import { HistoryOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Descriptions, Popover, Space, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import React, { useRef } from 'react';
import { adminServiceListAuditLogs } from '@/services/roncin/adminService';

const { Text } = Typography;

export default function AuditPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);

  const columns: ProColumns<API.AdminAuditLog>[] = [
    {
      title: '时间范围',
      dataIndex: 'timeRange',
      valueType: 'dateRange',
      hideInTable: true,
      fieldProps: {
        placeholder: ['开始时间', '结束时间'],
      },
    },
    {
      title: '操作时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 170,
      search: false,
    },
    {
      title: '操作动作',
      dataIndex: 'action',
      width: 200,
      render: (_, record) => (
        <Text strong style={{ fontSize: 13 }}>
          {record.action}
        </Text>
      ),
    },
    {
      title: '操作用户',
      dataIndex: 'userId',
      width: 200,
      copyable: true,
      render: (_, record) => (
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {record.userId || '-'}
        </Text>
      ),
    },
    {
      title: '执行结果',
      dataIndex: 'result',
      width: 100,
      valueEnum: { success: { text: '成功' }, failure: { text: '失败' } },
      render: (_, record) =>
        record.result === 'success' ? (
          <Tag color="success">成功</Tag>
        ) : (
          <Tag color="error">失败</Tag>
        ),
    },
    {
      title: '操作资源',
      dataIndex: 'resourceId',
      width: 180,
      copyable: true,
      render: (_, record) =>
        record.resourceId ? (
          <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
            {record.resourceId}
          </Text>
        ) : (
          '-'
        ),
    },
    {
      title: 'IP 地址',
      dataIndex: 'ipAddress',
      width: 140,
      search: false,
      render: (_, record) => record.ipAddress || '-',
    },
    {
      title: '请求 / 追踪编号',
      dataIndex: 'requestId',
      width: 180,
      search: false,
      render: (_, record) => (
        <Space vertical size={0}>
          {record.requestId && (
            <Text copyable style={{ fontSize: 11, fontFamily: 'monospace' }}>
              Req: {record.requestId}
            </Text>
          )}
          {record.traceId && (
            <Text copyable type="secondary" style={{ fontSize: 11, fontFamily: 'monospace' }}>
              Trace: {record.traceId}
            </Text>
          )}
        </Space>
      ),
    },
    {
      title: '变更详情',
      dataIndex: 'details',
      width: 110,
      search: false,
      render: (_, record) => {
        const entries = Object.entries(record.details ?? {});
        if (entries.length === 0) return <Text type="secondary">-</Text>;

        return (
          <Popover
            title="操作详细参数"
            trigger="click"
            content={
              <Descriptions size="small" column={1} bordered style={{ maxWidth: 360 }}>
                {entries.map(([key, value]) => (
                  <Descriptions.Item key={key} label={key}>
                    <span style={{ wordBreak: 'break-all', fontFamily: 'monospace', fontSize: 12 }}>
                      {String(value)}
                    </span>
                  </Descriptions.Item>
                ))}
              </Descriptions>
            }
          >
            <Button type="link" size="small" style={{ padding: 0 }}>
              查看详情 ({entries.length})
            </Button>
          </Popover>
        );
      },
    },
  ];

  return (
    <ProTable<API.AdminAuditLog>
      headerTitle={
        <Space size={8}>
          <HistoryOutlined style={{ color: '#1677ff' }} />
          <span>系统操作与安全审计日志</span>
        </Space>
      }
      rowKey="id"
      actionRef={actionRef}
      columns={columns}
      bordered
      pagination={{
        defaultPageSize: 20,
        showSizeChanger: true,
        showQuickJumper: true,
      }}
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
      search={{ labelWidth: 80, defaultCollapsed: false }}
    />
  );
}
