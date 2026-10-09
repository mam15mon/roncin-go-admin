import { ReloadOutlined, SyncOutlined } from '@ant-design/icons';
import type { ActionType } from '@ant-design/pro-components';
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import {
  Alert,
  Button,
  Card,
  Descriptions,
  Pagination,
  Space,
  Spin,
  Table,
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
      title="钉钉通讯录"
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
      <Typography.Paragraph type="secondary">
        新同步人员先进入待分配。管理员分配组织和角色后开通；重复同步只更新钉钉资料。本次未返回的人员不会自动离职。
      </Typography.Paragraph>
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
          <Alert
            showIcon
            type={
              running
                ? 'info'
                : sync.lastError || sync.failed || !sync.fullScope
                  ? 'warning'
                  : 'success'
            }
            title={statusTitle}
            description={
              <>
                {sync.fullScope
                  ? '通讯录授权范围：全企业。此范围仅说明可以访问哪些人员，不代表每条人员资料均已同步成功。'
                  : sync.scopeSummary || '通讯录范围尚未核验'}
                {!sync.fullScope &&
                  '；尚未确认全员授权范围，结果可能只包含部分人员。'}
                {sync.lastError && <div>{sync.lastError}</div>}
              </>
            }
            style={{ marginBottom: 12 }}
          />
          <Descriptions
            size="small"
            column={{ xs: 1, sm: 2, lg: 4 }}
            items={[
              {
                key: 'total',
                label: '可处理人员（去重）',
                children: sync.total ?? 0,
              },
              {
                key: 'created',
                label: '新增待分配',
                children: sync.created ?? 0,
              },
              {
                key: 'updated',
                label: '资料更新',
                children: sync.updated ?? 0,
              },
              {
                key: 'unchanged',
                label: '未变化',
                children: sync.unchanged ?? 0,
              },
              { key: 'failed', label: '失败记录', children: sync.failed ?? 0 },
              {
                key: 'time',
                label: '完成时间',
                children: formatDate(sync.completedAt, 'minute'),
              },
              {
                key: 'fields',
                label: '本次同步资料',
                span: 2,
                children:
                  sync.availableFields
                    ?.map((field) => fieldNames[field] || '其他资料')
                    .join('、') || '尚未核验',
              },
            ]}
          />
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
    </Card>
  );
}
