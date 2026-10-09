import {
  QuestionCircleOutlined,
  ReloadOutlined,
  SyncOutlined,
} from '@ant-design/icons';
import type { ActionType } from '@ant-design/pro-components';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Pagination,
  Space,
  Spin,
  Table,
  Tag,
  Tooltip,
  Typography,
} from 'antd';
import React, { useEffect, useRef, useState } from 'react';
import {
  adminServiceGetDingTalkDirectorySync,
  adminServiceGetLatestDingTalkDirectorySync,
  adminServiceStartDingTalkDirectorySync,
} from '@/services/roncin/adminService';
import { formatDate } from '@/utils/format';

const fieldNames: Record<string, string> = {
  display_name: '姓名',
  avatar_url: '头像',
  mobile: '联系方式',
  email: '邮箱',
  departments: '钉钉部门',
};
const statusNames: Record<string, string> = {
  pending: '等待同步',
  running: '正在同步',
  succeeded: '同步完成',
  failed: '等待重试',
  dead_letter: '同步已停止',
};
export function isDirectorySyncRunning(status?: string): boolean {
  return ['pending', 'running', 'failed'].includes(status?.toLowerCase() ?? '');
}

function renderFailureCell(value: React.ReactNode) {
  return (
    <div style={{ whiteSpace: 'normal', overflowWrap: 'anywhere' }}>
      {value}
    </div>
  );
}

export default function DingTalkDirectorySyncPanel({
  actionRef,
}: {
  actionRef: React.RefObject<ActionType | undefined>;
}) {
  const queryClient = useQueryClient();
  const [taskId, setTaskId] = useState<string>();
  const [page, setPage] = useState(1);
  const completedTask = useRef<string | undefined>(undefined);
  const syncQuery = useQuery({
    queryKey: ['admin-users', 'directory-sync', taskId, page],
    queryFn: async () => {
      const response = taskId
        ? await adminServiceGetDingTalkDirectorySync({
            taskId,
            page,
            pageSize: 20,
          })
        : await adminServiceGetLatestDingTalkDirectorySync({
            page,
            pageSize: 20,
          });
      if (!response) throw new Error('读取同步结果失败');
      return response.data ?? null;
    },
    refetchInterval: (query) =>
      isDirectorySyncRunning(query.state.data?.status) ? 2000 : false,
  });
  const startSync = useMutation({
    mutationFn: async () => {
      const response = await adminServiceStartDingTalkDirectorySync({});
      if (!response?.data?.taskId) throw new Error('同步任务未返回有效结果');
      return response.data;
    },
    onSuccess: (sync) => {
      setTaskId(sync.taskId);
      setPage(1);
      queryClient.setQueryData(
        ['admin-users', 'directory-sync', sync.taskId, 1],
        sync,
      );
    },
  });
  const sync = syncQuery.data;
  const running = isDirectorySyncRunning(sync?.status);
  const completed = sync?.status?.toLowerCase() === 'succeeded';
  const statusTitle =
    completed && sync.failed
      ? `同步结束，存在 ${sync.failed} 条失败记录`
      : completed && !sync.fullScope
        ? '同步结束，尚未确认全员授权范围'
        : statusNames[sync?.status?.toLowerCase() ?? ''] || '同步状态未知';
  useEffect(() => {
    if (!sync?.taskId || running || completedTask.current === sync.taskId)
      return;
    completedTask.current = sync.taskId;
    actionRef.current?.reload();
  }, [sync?.taskId, running, actionRef]);
  return (
    <Card
      size="small"
      title={
        <Space size={8} wrap>
          <span>钉钉通讯录</span>
          {sync && (
            <Tag
              color={
                running
                  ? 'processing'
                  : sync.lastError || sync.failed || !sync.fullScope
                    ? 'warning'
                    : 'success'
              }
              style={{ margin: 0 }}
            >
              {completed && sync.failed
                ? '部分失败'
                : statusNames[sync.status?.toLowerCase() ?? ''] ||
                  '同步状态未知'}
            </Tag>
          )}
          <Tooltip title="新同步人员先进入待分配，分配组织和角色后开通；重复同步只更新钉钉资料，本次未返回的人员不会自动离职。">
            <Button
              type="text"
              size="small"
              aria-label="同步说明"
              icon={<QuestionCircleOutlined />}
              style={{ color: 'rgba(0, 0, 0, 0.45)' }}
            />
          </Tooltip>
        </Space>
      }
      styles={{
        header: { flexWrap: 'wrap', gap: 8, paddingBlock: 8 },
      }}
      style={{ marginBottom: 'var(--roncin-page-section-gap)' }}
      extra={
        <Space>
          <Button
            icon={<ReloadOutlined />}
            loading={syncQuery.isFetching}
            onClick={() => {
              void syncQuery.refetch();
            }}
          >
            刷新状态
          </Button>
          <Button
            type="primary"
            icon={<SyncOutlined spin={running} />}
            loading={startSync.isPending}
            disabled={running || syncQuery.isLoading || syncQuery.isError}
            onClick={() => startSync.mutate()}
          >
            同步钉钉人员
          </Button>
        </Space>
      }
    >
      <div style={{ padding: '12px 16px' }}>
        {syncQuery.isLoading ? (
          <Spin />
        ) : syncQuery.isError ? (
          <Alert
            type="error"
            showIcon
            title="读取同步状态失败"
            description="请刷新状态后再发起同步。"
          />
        ) : sync ? (
          <>
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: '8px 28px',
                marginBottom: 8,
              }}
            >
              {[
                {
                  key: 'total',
                  label: '可处理人员（去重）',
                  value: sync.total,
                },
                { key: 'created', label: '新增待分配', value: sync.created },
                { key: 'updated', label: '资料更新', value: sync.updated },
                { key: 'unchanged', label: '未变化', value: sync.unchanged },
                { key: 'failed', label: '失败记录', value: sync.failed },
              ].map((item) => (
                <Space key={item.key} size={8}>
                  <Typography.Text type="secondary">
                    {item.label}
                  </Typography.Text>
                  <Typography.Text
                    strong
                    type={
                      item.key === 'failed' && item.value ? 'danger' : undefined
                    }
                    style={{ fontSize: 16 }}
                  >
                    {item.value ?? 0}
                  </Typography.Text>
                </Space>
              ))}
            </div>
            <div
              style={{
                display: 'flex',
                flexWrap: 'wrap',
                gap: '4px 20px',
                fontSize: 12,
                color: 'rgba(0, 0, 0, 0.45)',
              }}
            >
              <span>完成时间：{formatDate(sync.completedAt, 'minute')}</span>
              <Tooltip title="授权范围说明可以访问哪些人员；具体同步结果以人员统计和失败详情为准。">
                <span>
                  授权范围：
                  {sync.fullScope ? '全企业' : sync.scopeSummary || '尚未核验'}
                </span>
              </Tooltip>
              <span>
                本次同步资料：
                <span>
                  {sync.availableFields
                    ?.map((field) => fieldNames[field] || '其他资料')
                    .join('、') || '尚未核验'}
                </span>
              </span>
            </div>
            {!running &&
              Boolean(sync.lastError || sync.failed || !sync.fullScope) && (
                <Alert
                  showIcon
                  type="warning"
                  title={statusTitle}
                  description={
                    sync.lastError ||
                    (!sync.fullScope
                      ? '尚未确认全员授权范围，结果可能只包含部分人员。'
                      : '部分记录未同步成功，请查看下方具体失败原因。')
                  }
                  style={{ marginTop: 12, marginBottom: 12 }}
                />
              )}
            {Boolean(sync.failureTotal) && (
              <>
                <Typography.Paragraph type="secondary">
                  下方列出具体失败原因。失败按读取或处理记录计数，可能包含同一人员在不同部门的记录，与去重人员数不直接相加。
                </Typography.Paragraph>
                <Table<API.DingTalkDirectorySyncFailure>
                  size="small"
                  tableLayout="fixed"
                  column={{ render: renderFailureCell }}
                  pagination={false}
                  dataSource={sync.failures ?? []}
                  rowKey={(row) =>
                    `${row.userId}:${row.departmentId}:${row.reason}`
                  }
                  columns={[
                    {
                      title: '人员标识',
                      width: '28%',
                      dataIndex: 'userId',
                      render: (value) => renderFailureCell(value || '—'),
                    },
                    {
                      title: '部门标识',
                      width: '20%',
                      dataIndex: 'departmentId',
                      render: (value) =>
                        renderFailureCell(
                          value && value !== '0' ? value : '未记录',
                        ),
                    },
                    { title: '失败原因', dataIndex: 'message' },
                  ]}
                />
                <Pagination
                  current={page}
                  pageSize={20}
                  total={sync.failureTotal}
                  showSizeChanger={false}
                  onChange={setPage}
                  style={{ marginTop: 12 }}
                />
              </>
            )}
          </>
        ) : (
          <Typography.Text type="secondary">
            尚未同步，点击“同步钉钉人员”获取通讯录名单。
          </Typography.Text>
        )}
      </div>
    </Card>
  );
}
