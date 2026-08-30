import {
  ClockCircleOutlined,
  RedoOutlined,
  ReloadOutlined,
} from '@ant-design/icons';
import type {
  ActionType,
  ProColumns,
  ProFormInstance,
} from '@ant-design/pro-components';
import { ProTable } from '@ant-design/pro-components';
import { useAccess } from '@umijs/max';
import {
  App,
  Button,
  Descriptions,
  Popconfirm,
  Space,
  Tabs,
  Tag,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React, { useRef, useState } from 'react';
import {
  backgroundTaskStatusMeta,
  makeValueEnum,
  statusTag,
} from '@/constants/statusMeta';
import { BackgroundTaskStatus } from '@/enums.generated';
import {
  backgroundTaskServiceListBackgroundTasks,
  backgroundTaskServiceRequeueBackgroundTask,
} from '@/services/roncin/backgroundTaskService';
import { toTableRequest } from '@/utils/api';
import { formatDate } from '@/utils/format';
import {
  backgroundTaskExecutionSummary,
  backgroundTaskHasNextRunAt,
  backgroundTaskPresentation,
} from './background-task-presentation';

const { Text } = Typography;

type BackgroundTaskPhase = 1 | 2;

const activeStatusValueEnum = makeValueEnum({
  [BackgroundTaskStatus.BACKGROUND_TASK_STATUS_PENDING]:
    backgroundTaskStatusMeta[BackgroundTaskStatus.BACKGROUND_TASK_STATUS_PENDING],
  [BackgroundTaskStatus.BACKGROUND_TASK_STATUS_RUNNING]:
    backgroundTaskStatusMeta[BackgroundTaskStatus.BACKGROUND_TASK_STATUS_RUNNING],
  [BackgroundTaskStatus.BACKGROUND_TASK_STATUS_FAILED]:
    backgroundTaskStatusMeta[BackgroundTaskStatus.BACKGROUND_TASK_STATUS_FAILED],
});

const historyStatusValueEnum = makeValueEnum({
  [BackgroundTaskStatus.BACKGROUND_TASK_STATUS_SUCCEEDED]:
    backgroundTaskStatusMeta[BackgroundTaskStatus.BACKGROUND_TASK_STATUS_SUCCEEDED],
  [BackgroundTaskStatus.BACKGROUND_TASK_STATUS_DEAD_LETTER]:
    backgroundTaskStatusMeta[BackgroundTaskStatus.BACKGROUND_TASK_STATUS_DEAD_LETTER],
});

export default function BackgroundTasksPanel() {
  const actionRef = useRef<ActionType | undefined>(undefined);
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [taskPhase, setTaskPhase] = useState<BackgroundTaskPhase>(1);
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
      title: '任务内容',
      dataIndex: 'kind',
      width: 220,
      valueEnum: {
        1: { text: '主数据导入' },
        2: { text: 'UNLOCODE 导入' },
        3: { text: '订单提醒' },
        4: { text: '外部系统集成' },
        5: { text: '钉钉通知' },
      },
      render: (_, record) => {
        const presentation = backgroundTaskPresentation(record);
        return (
          <div>
            <Tag color={presentation.color} variant="filled">
              {presentation.label}
            </Tag>
            <Text
              type="secondary"
              style={{ display: 'block', marginTop: 4, fontSize: 12 }}
            >
              {presentation.description}
            </Text>
          </div>
        );
      },
    },
    {
      title: '通知对象',
      dataIndex: 'recipientDisplayName',
      width: 320,
      search: false,
      render: (_, record) =>
        record.recipientDisplayName ? (
          <div>
            <Text strong>{record.recipientDisplayName}</Text>
            {record.recipientUserId ? (
              <Text
                type="secondary"
                copyable={{ text: record.recipientUserId }}
                style={{ display: 'block', marginTop: 4, fontSize: 12 }}
              >
                用户 ID：{record.recipientUserId}
              </Text>
            ) : null}
          </div>
        ) : (
          <Text type="secondary">-</Text>
        ),
    },
    {
      title: '执行状态',
      dataIndex: 'status',
      width: 110,
      valueEnum:
        taskPhase === 1 ? activeStatusValueEnum : historyStatusValueEnum,
      render: (_, record) =>
        statusTag(backgroundTaskStatusMeta, record.status ?? 0),
    },
    {
      title: '执行情况',
      dataIndex: 'attempts',
      width: 190,
      search: false,
      render: (_, record) => {
        const type: React.ComponentProps<typeof Text>['type'] =
          record.status === 3
            ? 'success'
            : record.status === 4 || record.status === 5
              ? 'warning'
              : 'secondary';
        return (
          <Text type={type}>{backgroundTaskExecutionSummary(record)}</Text>
        );
      },
    },
    {
      title: '下次执行时间',
      dataIndex: 'nextRunAt',
      width: 170,
      search: false,
      render: (_, record) =>
        backgroundTaskHasNextRunAt(record) ? formatDate(record.nextRunAt) : '-',
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
            title="确定重新执行此任务？"
            description="任务将恢复为待执行状态，并由后台服务重新处理"
            onConfirm={async () => {
              if (!record.id) return;
              try {
                await backgroundTaskServiceRequeueBackgroundTask(
                  { id: record.id },
                  { id: record.id },
                );
                message.success('任务已加入重新执行队列');
                actionRef.current?.reload();
              } catch (err: any) {
                message.error(err?.message || '重新执行任务失败');
              }
            }}
          >
            <Button type="link" size="small" icon={<RedoOutlined />}>
              重新执行
            </Button>
          </Popconfirm>
        );
      },
    },
  ];

  return (
    <ProTable<API.BackgroundTask>
      headerTitle={
        <div>
          <Space size={8}>
            <ClockCircleOutlined style={{ color: '#1677ff' }} />
            <span>后台任务</span>
            <Text type="secondary">系统自动执行的导入、通知和集成记录</Text>
          </Space>
          <Tabs
            activeKey={String(taskPhase)}
            items={[
              { key: '1', label: '正在进行' },
              { key: '2', label: '历史记录' },
            ]}
            onChange={(key) => {
              formRef.current?.setFieldValue('status', undefined);
              setTaskPhase(Number(key) as BackgroundTaskPhase);
            }}
            size="small"
            tabBarStyle={{ margin: '8px 0 0' }}
          />
        </div>
      }
      rowKey="id"
      actionRef={actionRef}
      formRef={formRef}
      params={{ phase: taskPhase }}
      columns={columns}
      bordered
      expandable={{
        expandedRowRender: (record) => (
          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2, lg: 3 }}
            items={[
              {
                key: 'id',
                label: '任务 ID',
                children: (
                  <Text copyable={{ text: record.id }}>{record.id || '-'}</Text>
                ),
              },
              {
                key: 'idempotencyKey',
                label: '幂等标识',
                children: (
                  <Text copyable={{ text: record.idempotencyKey }}>
                    {record.idempotencyKey || '-'}
                  </Text>
                ),
              },
              {
                key: 'attemptLimit',
                label: '失败次数上限',
                children: record.maxAttempts ?? '-',
              },
              {
                key: 'updatedAt',
                label: '最近更新时间',
                children: formatDate(record.updatedAt),
              },
            ]}
          />
        ),
      }}
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
          phase: taskPhase,
          status:
            params.status !== undefined && params.status !== ''
              ? Number(params.status)
              : undefined,
          kind:
            params.kind !== undefined && params.kind !== ''
              ? Number(params.kind)
              : undefined,
          startTime: range?.[0]
            ? dayjs(range[0]).startOf('day').toISOString()
            : undefined,
          endTime: range?.[1]
            ? dayjs(range[1]).add(1, 'day').startOf('day').toISOString()
            : undefined,
        });
        return toTableRequest(response);
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
