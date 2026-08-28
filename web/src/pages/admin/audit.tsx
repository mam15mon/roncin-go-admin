import { HistoryOutlined, ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { Button, Descriptions, Space, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import React, { useRef } from 'react';
import { adminServiceListAuditLogs } from '@/services/roncin/adminService';
import {
  auditActionPresentation,
  auditActorName,
  auditBusinessObject,
  auditDetailLabel,
} from './audit-presentation';

const { Text } = Typography;

function technicalText(value?: string) {
  return value ? (
    <Text
      copyable
      style={{ fontFamily: 'monospace', fontSize: 12, wordBreak: 'break-all' }}
    >
      {value}
    </Text>
  ) : (
    '—'
  );
}

export default function AuditPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);

  const columns: ProColumns<API.AdminAuditLog>[] = [
    {
      title: '时间范围',
      dataIndex: 'timeRange',
      valueType: 'dateRange',
      hideInTable: true,
      fieldProps: { placeholder: ['开始时间', '结束时间'] },
    },
    {
      title: '操作时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 170,
      search: false,
    },
    {
      title: '操作人',
      dataIndex: 'actorDisplayName',
      width: 140,
      search: false,
      render: (_, record) => <Text strong>{auditActorName(record)}</Text>,
    },
    {
      title: '操作内容',
      dataIndex: 'action',
      width: 260,
      search: false,
      render: (_, record) => {
        const presentation = auditActionPresentation(record.action);
        return (
          <Space size={6}>
            <Tag color={presentation.color}>{presentation.category}</Tag>
            <Text>{presentation.title}</Text>
          </Space>
        );
      },
    },
    {
      title: '业务对象',
      dataIndex: 'targetDisplayName',
      width: 190,
      search: false,
      render: (_, record) => {
        const target = auditBusinessObject(record);
        return (
          <Space orientation="vertical" size={0}>
            <Text>{target.name}</Text>
            {target.type && <Text type="secondary">{target.type}</Text>}
          </Space>
        );
      },
    },
    {
      title: '执行结果',
      dataIndex: 'result',
      width: 100,
      search: false,
      render: (_, record) =>
        record.result === 'success' ? (
          <Tag color="success">成功</Tag>
        ) : (
          <Tag color="error">失败</Tag>
        ),
    },
    {
      title: '来源地址',
      dataIndex: 'ipAddress',
      width: 140,
      search: false,
      render: (_, record) => record.ipAddress || '—',
    },
  ];

  return (
    <ProTable<API.AdminAuditLog>
      headerTitle={
        <Space size={10}>
          <HistoryOutlined style={{ color: '#1677ff' }} />
          <Space orientation="vertical" size={0}>
            <Text strong>操作与安全记录</Text>
            <Text type="secondary" style={{ fontSize: 12, fontWeight: 400 }}>
              查看人员登录、权限和业务资料变更
            </Text>
          </Space>
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
          startTime: range?.[0]
            ? dayjs(range[0]).startOf('day').toISOString()
            : undefined,
          endTime: range?.[1]
            ? dayjs(range[1]).add(1, 'day').startOf('day').toISOString()
            : undefined,
        });
        return {
          data: response.data ?? [],
          success: response.success ?? true,
          total: response.total ?? 0,
        };
      }}
      expandable={{
        expandedRowRender: (record) => {
          const presentation = auditActionPresentation(record.action);
          const target = auditBusinessObject(record);
          const details = Object.entries(record.details ?? {});
          return (
            <Descriptions
              title="技术详情"
              size="small"
              bordered
              column={{ xs: 1, sm: 1, md: 2, lg: 2, xl: 2, xxl: 2 }}
              items={[
                {
                  key: 'summary',
                  label: '业务说明',
                  children: `${auditActorName(record)} · ${presentation.title} · ${target.name}`,
                  span: 'filled',
                },
                {
                  key: 'action',
                  label: '原始动作码',
                  children: technicalText(record.action),
                },
                {
                  key: 'actor',
                  label: '操作人用户 ID',
                  children: technicalText(record.userId),
                },
                {
                  key: 'audit',
                  label: '审计记录 ID',
                  children: technicalText(record.id),
                },
                {
                  key: 'resourceType',
                  label: '资源类型',
                  children: technicalText(record.resourceType),
                },
                {
                  key: 'resourceId',
                  label: '资源 ID',
                  children: technicalText(record.resourceId),
                },
                {
                  key: 'request',
                  label: '请求编号',
                  children: technicalText(record.requestId),
                },
                {
                  key: 'trace',
                  label: '追踪编号',
                  children: technicalText(record.traceId),
                },
                ...details.map(([key, value]) => ({
                  key: `detail-${key}`,
                  label: auditDetailLabel(key),
                  children: technicalText(String(value)),
                })),
              ]}
            />
          );
        },
      }}
      toolBarRender={() => [
        <Button
          key="refresh"
          icon={<ReloadOutlined />}
          onClick={() => actionRef.current?.reload()}
        >
          刷新
        </Button>,
      ]}
      search={{ labelWidth: 80, defaultCollapsed: false }}
    />
  );
}
