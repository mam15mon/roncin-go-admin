import {
  Alert,
  Descriptions,
  Drawer,
  Empty,
  List,
  Space,
  Spin,
  Tag,
  Typography,
} from 'antd';
import dayjs from 'dayjs';
import React, { useCallback, useEffect, useState } from 'react';
import { orderLockServiceListOrderUnlockRequests } from '@/services/roncin/orderLockService';

const ACTIVE_UNLOCK_STATUSES = new Set([
  'PENDING_DISPATCH',
  'PENDING_APPROVAL',
  'APPROVED_PENDING_APPLY',
]);

const STATUS_META: Record<
  string,
  { color: string; label: string; description: string }
> = {
  PENDING_DISPATCH: {
    color: 'processing',
    label: '待派发',
    description: '申请已保存，正在发起钉钉 OA 审批。',
  },
  PENDING_APPROVAL: {
    color: 'blue',
    label: '审批中',
    description: '钉钉 OA 审批已发起，等待候选审批人处理。',
  },
  APPROVED_PENDING_APPLY: {
    color: 'gold',
    label: '已同意待本地生效',
    description: '钉钉已同意，系统正在复验资格并解除订单锁。',
  },
  APPROVED: {
    color: 'success',
    label: '已解锁',
    description: '审批已通过并在本地成功解除订单锁。',
  },
  REJECTED: {
    color: 'error',
    label: '已拒绝',
    description: '钉钉审批已拒绝，订单继续保持锁定。',
  },
  CONFIGURATION_FAILED: {
    color: 'error',
    label: '配置失败',
    description: '申请未能发起，请按失败原因补全配置。',
  },
  DISPATCH_FAILED: {
    color: 'error',
    label: '派发失败',
    description: '钉钉明确未受理审批申请。',
  },
  DISPATCH_UNKNOWN: {
    color: 'purple',
    label: '派发结果未知',
    description: '无法确认钉钉是否创建实例，系统已停止自动重发以避免重复审批。',
  },
  STALE: {
    color: 'default',
    label: '已过期',
    description: '锁定代次、订单版本或解锁路径已变化，本申请不会再生效。',
  },
};

export function getUnlockRequestStatusMeta(status?: string) {
  return (
    STATUS_META[status || ''] || {
      color: 'default',
      label: status || '未知状态',
      description: '请刷新后查看最新服务端状态。',
    }
  );
}

export function shouldPollUnlockRequests(
  items: API.OrderUnlockRequestData[],
) {
  return items.some((item) => ACTIVE_UNLOCK_STATUSES.has(item.status || ''));
}

type Props = {
  open: boolean;
  orderId: string;
  onClose: () => void;
};

export default function UnlockRequestHistoryDrawer({
  open,
  orderId,
  onClose,
}: Props) {
  const [loading, setLoading] = useState(false);
  const [items, setItems] = useState<API.OrderUnlockRequestData[]>([]);
  const [error, setError] = useState<string>();

  const load = useCallback(async () => {
    if (!orderId) return;
    setLoading(true);
    try {
      const response = await orderLockServiceListOrderUnlockRequests({
        orderId,
        page: 1,
        pageSize: 50,
      });
      setItems(response.data?.items || []);
      setError(undefined);
    } catch (caught: unknown) {
      setError(
        caught instanceof Error ? caught.message : '加载解锁审批记录失败',
      );
    } finally {
      setLoading(false);
    }
  }, [orderId]);

  useEffect(() => {
    if (!open) return;
    void load();
  }, [load, open]);

  useEffect(() => {
    if (
      !open ||
      !shouldPollUnlockRequests(items)
    )
      return;
    const timer = window.setInterval(() => void load(), 5000);
    return () => window.clearInterval(timer);
  }, [items, load, open]);

  return (
    <Drawer
      title="解锁审批记录"
      width={600}
      open={open}
      onClose={onClose}
      destroyOnHidden
    >
      {error && (
        <Alert
          style={{ marginBottom: 16 }}
          type="error"
          showIcon
          message={error}
          action={
            <Typography.Link onClick={() => void load()}>重试</Typography.Link>
          }
        />
      )}
      <Spin spinning={loading && items.length === 0}>
        {items.length === 0 && !loading ? (
          <Empty description="暂无解锁申请" />
        ) : (
          <List
            dataSource={items}
            renderItem={(item) => {
              const meta = getUnlockRequestStatusMeta(item.status);
              return (
                <List.Item key={item.id} style={{ alignItems: 'stretch' }}>
                  <Space
                    direction="vertical"
                    size={10}
                    style={{ width: '100%' }}
                  >
                    <Space wrap>
                      <Tag color={meta.color}>{meta.label}</Tag>
                      <Typography.Text strong>
                        第 {item.lockGeneration || '-'} 代锁定
                      </Typography.Text>
                      <Typography.Text type="secondary">
                        {item.requestedAt
                          ? dayjs(item.requestedAt).format(
                              'YYYY-MM-DD HH:mm:ss',
                            )
                          : '-'}
                      </Typography.Text>
                    </Space>
                    <Typography.Text type="secondary">
                      {meta.description}
                    </Typography.Text>
                    {item.status === 'DISPATCH_UNKNOWN' && (
                      <Alert
                        type="warning"
                        showIcon
                        message="请由管理员在钉钉后台核对实例，系统不会自动重发。"
                      />
                    )}
                    <Descriptions size="small" column={1} bordered>
                      <Descriptions.Item label="申请人">
                        {item.requestedByName || item.requestedBy || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="处理路径">
                        {item.route || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="申请原因">
                        {item.reason || '-'}
                      </Descriptions.Item>
                      <Descriptions.Item label="审批候选人">
                        {item.approverCandidates
                          ?.map((candidate) => candidate.displayNameSnapshot)
                          .filter(Boolean)
                          .join('、') || '-'}
                      </Descriptions.Item>
                      {item.decidedAt && (
                        <Descriptions.Item label="审批时间">
                          {dayjs(item.decidedAt).format('YYYY-MM-DD HH:mm:ss')}
                        </Descriptions.Item>
                      )}
                      {item.decidedByName && (
                        <Descriptions.Item label="审批人">
                          {item.decidedByName}
                        </Descriptions.Item>
                      )}
                      {item.failureMessage && (
                        <Descriptions.Item label="失败原因">
                          <Typography.Text type="danger">
                            {item.failureMessage}
                          </Typography.Text>
                        </Descriptions.Item>
                      )}
                      {item.unlockedAt && (
                        <Descriptions.Item label="解锁时间">
                          {dayjs(item.unlockedAt).format('YYYY-MM-DD HH:mm:ss')}
                        </Descriptions.Item>
                      )}
                    </Descriptions>
                  </Space>
                </List.Item>
              );
            }}
          />
        )}
      </Spin>
    </Drawer>
  );
}
