import { ClockCircleOutlined, ReloadOutlined, RedoOutlined } from '@ant-design/icons';
import type { ActionType, ProColumns } from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import { App, Button, Popconfirm, Space, Tag, Typography } from 'antd';
import dayjs from 'dayjs';
import React, { useRef } from 'react';
import {
  backgroundTaskServiceListBackgroundTasks,
  backgroundTaskServiceRequeueBackgroundTask,
} from '@/services/roncin/backgroundTaskService';

const { Text } = Typography;

const statusTagMap: Record<number, { color: string; label: string }> = {
  1: { color: 'default', label: '待执行' },
  2: { color: 'processing', label: '执行中' },
  3: { color: 'success', label: '已成功' },
  4: { color: 'warning', label: '已失败' },
  5: { color: 'error', label: '死信' },
};

const kindTagMap: Record<number, { label: string; color: string }> = {
  1: { label: '主数据导入', color: 'blue' },
  2: { label: 'UNLOCODE 导入', color: 'cyan' },
  3: { label: '订单提醒', color: 'orange' },
  4: { label: '外部系统集成', color: 'purple' },
};

export default function BackgroundTasksPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const access = useAccess();
  const { message } = App.useApp();

  const columns: ProColumns<API.BackgroundTask>[] = [
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
      title: '创建时间',
      dataIndex: 'createdAt',
      valueType: 'dateTime',
      width: 170,
      search: false,
    },
    {
      title: '任务类型',
      dataIndex: 'kind',
      width: 150,
      valueEnum: {
        1: { text: '主数据导入' },
        2: { text: 'UNLOCODE 导入' },
        3: { text: '订单提醒' },
        4: { text: '外部系统集成' },
      },
      render: (_, record) => {
        const config = record.kind ? kindTagMap[record.kind] : undefined;
        return config ? (
          <Tag color={config.color} variant="filled">
            {config.label}
          </Tag>
        ) : (
          '-'
        );
      },
    },
    {
      title: '执行状态',
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
    {
      title: '幂等键',
      dataIndex: 'idempotencyKey',
      width: 200,
      copyable: true,
      render: (_, record) => (
        <Text style={{ fontFamily: 'monospace', fontSize: 12 }}>
          {record.idempotencyKey || '-'}
        </Text>
      ),
    },
    {
      title: '重试次数',
      dataIndex: 'attempts',
      width: 100,
      search: false,
      render: (_, record) => {
        const attempts = record.attempts ?? 0;
        const maxAttempts = record.maxAttempts ?? 0;
        const hasFailed = attempts >= maxAttempts && maxAttempts > 0;
        return (
          <Tag color={hasFailed ? 'error' : attempts > 0 ? 'warning' : 'default'} variant="filled">
            {attempts} / {maxAttempts}
          </Tag>
        );
      },
    },
    {
      title: '下次调度时间',
      dataIndex: 'nextRunAt',
      valueType: 'dateTime',
      width: 170,
      search: false,
    },
    {
      title: '最近错误信息',
      dataIndex: 'lastError',
      search: false,
      render: (_, record) =>
        record.lastError ? (
          <Typography.Paragraph
            type="danger"
            ellipsis={{ rows: 2, expandable: true }}
            style={{ marginBottom: 0, maxWidth: 300, fontSize: 12 }}
          >
            {record.lastError}
          </Typography.Paragraph>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '操作',
      valueType: 'option',
      width: 90,
      fixed: 'right',
      search: false,
      render: (_, record) => {
        if (!access.canRequeueTasks) return null;
        if (record.status !== 4 && record.status !== 5) return null;
        return (
          <Popconfirm
            title="确定重新回放此任务？"
            description="任务将重置为「待执行」状态并释放当前租约"
            onConfirm={async () => {
              if (!record.id) return;
              try {
                await backgroundTaskServiceRequeueBackgroundTask(
                  { id: record.id },
                  { id: record.id },
                );
                message.success('任务已成功加入回放队列');
                actionRef.current?.reload();
              } catch (err: any) {
                message.error(err?.message || '回放任务失败');
              }
            }}
          >
            <Button type="link" size="small" icon={<RedoOutlined />}>
              回放
            </Button>
          </Popconfirm>
        );
      },
    },
  ];

  return (
    <ProTable<API.BackgroundTask>
      headerTitle={
        <Space size={8}>
          <ClockCircleOutlined style={{ color: '#1677ff' }} />
          <span>后台异步任务队列</span>
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
      search={{ labelWidth: 80, defaultCollapsed: false }}
    />
  );
}
