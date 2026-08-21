import { ReloadOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Popconfirm, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import React, { useRef } from 'react';
import {
  backgroundTaskServiceListBackgroundTasks,
  backgroundTaskServiceRequeueBackgroundTask,
} from '@/services/roncin/backgroundTaskService';

const statusTagMap: Record<number, { color: string; label: string }> = {
  1: { color: 'default', label: '待执行' },
  2: { color: 'processing', label: '执行中' },
  3: { color: 'success', label: '已成功' },
  4: { color: 'warning', label: '已失败' },
  5: { color: 'error', label: '死信' },
};

export default function BackgroundTasksPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const access = useAccess();
  const { message } = App.useApp();

  const columns: ProColumns<API.BackgroundTask>[] = [
    { title: '时间范围', dataIndex: 'timeRange', valueType: 'dateRange', hideInTable: true },
    { title: '创建时间', dataIndex: 'createdAt', valueType: 'dateTime', width: 180, search: false },
    {
      title: '类型',
      dataIndex: 'kind',
      width: 140,
      valueEnum: {
        1: { text: '主数据导入' },
        2: { text: 'UNLOCODE 导入' },
        3: { text: '订单提醒' },
        4: { text: '集成任务' },
      },
    },
    {
      title: '状态',
      dataIndex: 'status',
      width: 110,
      valueEnum: {
        1: { text: '待执行' },
        2: { text: '执行中' },
        3: { text: '已成功' },
        4: { text: '已失败' },
        5: { text: '死信' },
      },
      render: (_, record) => {
        const config = record.status ? statusTagMap[record.status] : undefined;
        return config ? <Tag color={config.color}>{config.label}</Tag> : '-';
      },
    },
    { title: '幂等键', dataIndex: 'idempotencyKey', width: 220, copyable: true },
    {
      title: '重试次数',
      dataIndex: 'attempts',
      width: 100,
      search: false,
      render: (_, record) => `${record.attempts ?? 0}/${record.maxAttempts ?? 0}`,
    },
    { title: '下次执行', dataIndex: 'nextRunAt', valueType: 'dateTime', width: 180, search: false },
    {
      title: '最近错误',
      dataIndex: 'lastError',
      search: false,
      render: (_, record) =>
        record.lastError ? (
          <Typography.Paragraph
            ellipsis={{ rows: 2, expandable: true }}
            style={{ marginBottom: 0, maxWidth: 300 }}
          >
            {record.lastError}
          </Typography.Paragraph>
        ) : (
          '-'
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 80,
      search: false,
      render: (_, record) => {
        if (!access.canManageTasks) return null;
        if (record.status !== 4 && record.status !== 5) return null;
        return (
          <Popconfirm
            title="确定回放此任务？"
            description="任务将重置为待执行并清空租约"
            onConfirm={async () => {
              if (!record.id) return;
              try {
                await backgroundTaskServiceRequeueBackgroundTask(
                  { id: record.id },
                  { id: record.id },
                );
                message.success('任务已回放');
                actionRef.current?.reload();
              } catch (err: any) {
                message.error(err?.message || '回放失败');
              }
            }}
          >
            <Button type="link" size="small">
              回放
            </Button>
          </Popconfirm>
        );
      },
    },
  ];

  return (
    <ProTable<API.BackgroundTask>
      rowKey="id"
      actionRef={actionRef}
      columns={columns}
      pagination={{ defaultPageSize: 20, showSizeChanger: true }}
      request={async (params) => {
        const range = params.timeRange as [string, string] | undefined;
        const response = await backgroundTaskServiceListBackgroundTasks({
          page: params.current,
          pageSize: params.pageSize,
          status:
            params.status !== undefined && params.status !== ''
              ? Number(params.status)
              : undefined,
          kind:
            params.kind !== undefined && params.kind !== ''
              ? Number(params.kind)
              : undefined,
          startTime: range?.[0] ? dayjs(range[0]).startOf('day').toISOString() : undefined,
          endTime: range?.[1] ? dayjs(range[1]).add(1, 'day').startOf('day').toISOString() : undefined,
        });
        return {
          data: response.data ?? [],
          success: response.success ?? true,
          total: response.total ?? 0,
        };
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
