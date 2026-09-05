import {
  HistoryOutlined,
  LockOutlined,
  UnlockOutlined,
} from '@ant-design/icons';
import { App, Button, Form, Input, Modal, Tag, Tooltip } from 'antd';
import dayjs from 'dayjs';
import React, { useEffect, useState } from 'react';
import { OrderBusinessType } from '@/enums.generated';
import {
  orderLockServiceLockOrder,
  orderLockServiceRequestOrderUnlock,
} from '@/services/roncin/orderLockService';
import { getOrderBusinessTypeLabel } from '../../use-order-lock-state';
import UnlockRequestHistoryDrawer from './UnlockRequestHistoryDrawer';

type UnlockRoute = 'ROLE_DIRECT' | 'ADMIN_EMERGENCY' | 'DINGTALK_APPROVAL';

type OrderLockControlProps = {
  orderId: string;
  orderNo?: string;
  state: API.OrderLockStateData | null;
  loading: boolean;
  error: Error | null;
  onRetry: () => Promise<API.OrderLockStateData | null>;
  onSynchronize: () => Promise<void>;
};

type OrderLockStatusTagProps = Pick<
  OrderLockControlProps,
  'state' | 'loading' | 'error'
>;

function generateIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (value) => {
    const random = (Math.random() * 16) | 0;
    const digit = value === 'x' ? random : (random & 0x3) | 0x8;
    return digit.toString(16);
  });
}

export function getOrderLockConfirmationDescription(
  businessType?: number,
): string {
  if (businessType === OrderBusinessType.BUSINESS_TYPE_SE) {
    return '锁定后将固定 MBL/HBL 提单不可变版本，并冻结订单业务资料和费用。如需修改必须先解锁。';
  }
  return '锁定后将冻结订单业务资料和费用。如需修改必须先解锁。';
}

function getUnlockTitle(route: UnlockRoute, businessType?: number): string {
  const businessTypeLabel = getOrderBusinessTypeLabel(businessType);
  if (route === 'ROLE_DIRECT') return `直接解锁${businessTypeLabel}订单`;
  if (route === 'ADMIN_EMERGENCY') return '系统管理员紧急解锁';
  return `申请解锁${businessTypeLabel}订单`;
}

export function OrderLockStatusTag({
  state,
  loading,
  error,
}: OrderLockStatusTagProps) {
  if (loading) return <Tag color="processing">锁状态同步中</Tag>;
  if (error) return <Tag color="error">锁状态加载失败</Tag>;
  if (!state) return <Tag color="warning">锁状态未知</Tag>;

  const businessTypeLabel = getOrderBusinessTypeLabel(state.businessType);
  if (!state.isLocked) {
    return (
      <Tag color="success" icon={<UnlockOutlined />}>
        {businessTypeLabel} · 未锁定
      </Tag>
    );
  }

  const timeText = state.lockedAt
    ? dayjs(state.lockedAt).format('YYYY-MM-DD HH:mm')
    : '';
  return (
    <Tooltip
      title={
        <div style={{ fontSize: 12 }}>
          <div>业务类型：{businessTypeLabel}</div>
          <div>锁定轮次：第 {state.lockGeneration} 代</div>
          <div>锁定人：{state.lockedByName || state.lockedBy || '系统'}</div>
          {timeText && <div>锁定时间：{timeText}</div>}
          {state.activeUnlockRequest && (
            <div style={{ marginTop: 4, color: '#faad14' }}>
              当前解锁申请：{state.activeUnlockRequest.status}（
              {state.activeUnlockRequest.route}）
            </div>
          )}
        </div>
      }
    >
      <Tag color="error" icon={<LockOutlined />}>
        {businessTypeLabel} · 已锁定 · {state.lockedByName || '已锁定'}
        {timeText ? ` · ${timeText}` : ''}
      </Tag>
    </Tooltip>
  );
}

export default function OrderLockControl({
  orderId,
  orderNo,
  state,
  loading,
  error,
  onRetry,
  onSynchronize,
}: OrderLockControlProps) {
  const { message, modal } = App.useApp();
  const [locking, setLocking] = useState(false);
  const [unlocking, setUnlocking] = useState(false);
  const [unlockModalVisible, setUnlockModalVisible] = useState(false);
  const [unlockHistoryVisible, setUnlockHistoryVisible] = useState(false);
  const [unlockRoute, setUnlockRoute] = useState<UnlockRoute>('ROLE_DIRECT');
  const [unlockForm] = Form.useForm<{ reason?: string }>();

  useEffect(() => {
    if (unlockModalVisible && (loading || error || !state?.isLocked)) {
      setUnlockModalVisible(false);
      unlockForm.resetFields();
    }
  }, [error, loading, state, unlockForm, unlockModalVisible]);

  const expectedOrderVersion = state?.orderVersion;
  const versionMissing = !expectedOrderVersion;
  const businessTypeLabel = getOrderBusinessTypeLabel(state?.businessType);

  const handleLock = () => {
    if (!state?.canLock || !expectedOrderVersion) {
      message.warning('订单锁状态或版本已变化，请刷新后重试');
      return;
    }
    modal.confirm({
      title: `锁定${businessTypeLabel}订单`,
      content: (
        <div>
          <p>
            确定要锁定订单 <b>{orderNo || state.orderNo || orderId}</b> 吗？
          </p>
          <p style={{ color: '#64748b' }}>
            {getOrderLockConfirmationDescription(state.businessType)}
          </p>
        </div>
      ),
      okText: '确认锁定',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        setLocking(true);
        try {
          const response = await orderLockServiceLockOrder(
            { orderId },
            {
              orderId,
              expectedOrderVersion,
              idempotencyKey: generateIdempotencyKey(),
            },
          );
          if (response?.success) {
            message.success('订单已成功锁定');
            await onSynchronize();
          }
        } catch (requestError: unknown) {
          message.error(
            requestError instanceof Error
              ? requestError.message
              : '锁定订单失败',
          );
          throw requestError;
        } finally {
          setLocking(false);
        }
      },
    });
  };

  const openUnlockModal = (route: UnlockRoute) => {
    if (!state?.isLocked || !expectedOrderVersion) {
      message.warning('订单锁状态或版本已变化，请刷新后重试');
      return;
    }
    setUnlockRoute(route);
    unlockForm.resetFields();
    setUnlockModalVisible(true);
  };

  const handleUnlockSubmit = async (values: { reason?: string }) => {
    if (!state?.isLocked || !state.orderVersion) {
      message.warning('订单锁状态或版本已变化，请刷新后重试');
      setUnlockModalVisible(false);
      return;
    }
    setUnlocking(true);
    try {
      const response = await orderLockServiceRequestOrderUnlock(
        { orderId },
        {
          orderId,
          expectedOrderVersion: state.orderVersion,
          idempotencyKey: generateIdempotencyKey(),
          reason: values.reason?.trim() || undefined,
        },
      );
      if (response?.success) {
        const actualRoute = response.data?.request?.route;
        const actualStatus = response.data?.request?.status;
        if (
          actualStatus === 'CONFIGURATION_FAILED' ||
          actualStatus === 'DISPATCH_FAILED' ||
          actualStatus === 'DISPATCH_UNKNOWN'
        ) {
          message.error(
            response.data?.request?.failureMessage ||
              '解锁审批暂时无法发起，请联系管理员',
          );
        } else if (
          actualRoute === 'DINGTALK_APPROVAL' &&
          (actualStatus === 'PENDING_DISPATCH' ||
            actualStatus === 'PENDING_APPROVAL')
        ) {
          message.success('解锁申请已提交，等待业务角色成员审批');
        } else if (actualStatus === 'APPROVED') {
          message.success('订单已成功解锁');
        } else {
          message.success('解锁请求已处理');
        }
        setUnlockModalVisible(false);
        unlockForm.resetFields();
        await onSynchronize();
        if (actualRoute === 'DINGTALK_APPROVAL') {
          setUnlockHistoryVisible(true);
        }
      }
    } catch (requestError: unknown) {
      message.error(
        requestError instanceof Error ? requestError.message : '处理解锁失败',
      );
    } finally {
      setUnlocking(false);
    }
  };

  const lockBlockedTip = versionMissing
    ? '订单版本信息缺失，请刷新后重试'
    : state?.lockBlockedReasons?.join('；');
  const unlockBlockedTip = versionMissing
    ? '订单版本信息缺失，请刷新后重试'
    : state?.unlockBlockedReasons?.join('；');

  return (
    <>
      {!loading && (error || !state) && (
        <Button onClick={() => void onRetry()}>重试锁状态</Button>
      )}
      <Button
        icon={<HistoryOutlined />}
        onClick={() => setUnlockHistoryVisible(true)}
      >
        解锁记录
      </Button>

      {state && !state.isLocked && (
        <Tooltip
          title={!state.canLock || versionMissing ? lockBlockedTip : undefined}
        >
          <span>
            <Button
              style={{ color: '#d4380d', borderColor: '#ffbb96' }}
              icon={<LockOutlined />}
              disabled={!state.canLock || versionMissing || loading}
              loading={locking}
              onClick={handleLock}
            >
              锁定订单
            </Button>
          </span>
        </Tooltip>
      )}

      {state?.isLocked && state.canRoleDirectUnlock && (
        <Button
          type="primary"
          style={{ backgroundColor: '#fa8c16', borderColor: '#fa8c16' }}
          icon={<UnlockOutlined />}
          disabled={versionMissing || loading}
          onClick={() => openUnlockModal('ROLE_DIRECT')}
        >
          直接解锁
        </Button>
      )}
      {state?.isLocked && state.canAdminEmergencyUnlock && (
        <Button
          danger
          type="primary"
          icon={<UnlockOutlined />}
          disabled={versionMissing || loading}
          onClick={() => openUnlockModal('ADMIN_EMERGENCY')}
        >
          紧急解锁
        </Button>
      )}
      {state?.isLocked && state.canRequestUnlock && (
        <Tooltip title={unlockBlockedTip}>
          <span>
            <Button
              style={{ color: '#1677ff', borderColor: '#1677ff' }}
              icon={<UnlockOutlined />}
              disabled={versionMissing || loading}
              onClick={() => openUnlockModal('DINGTALK_APPROVAL')}
            >
              申请解锁
            </Button>
          </span>
        </Tooltip>
      )}

      <Modal
        title={getUnlockTitle(unlockRoute, state?.businessType)}
        open={unlockModalVisible}
        onCancel={() => {
          setUnlockModalVisible(false);
          unlockForm.resetFields();
        }}
        onOk={() => unlockForm.submit()}
        confirmLoading={unlocking}
        okText={unlockRoute === 'DINGTALK_APPROVAL' ? '提交申请' : '确认解锁'}
        okButtonProps={{ danger: unlockRoute === 'ADMIN_EMERGENCY' }}
      >
        <div style={{ marginBottom: 16, color: '#64748b' }}>
          {unlockRoute === 'ROLE_DIRECT' &&
            `您具备${businessTypeLabel}订单锁定业务角色权限，可以直接解除业务锁。解锁后订单将恢复可编辑。`}
          {unlockRoute === 'ADMIN_EMERGENCY' &&
            '您正在以系统管理员身份执行紧急解锁。该操作将直接解除业务锁并记录安全审计追踪。'}
          {unlockRoute === 'DINGTALK_APPROVAL' &&
            '提交后将通过统一订单解锁审批流程，指派给具备对应业务类型锁定权限的角色成员。'}
        </div>
        <Form form={unlockForm} layout="vertical" onFinish={handleUnlockSubmit}>
          <Form.Item
            name="reason"
            label="解锁原因"
            rules={[{ max: 500, message: '解锁原因不能超过 500 个字符' }]}
          >
            <Input.TextArea rows={3} placeholder="请输入解锁原因（选填）..." />
          </Form.Item>
        </Form>
      </Modal>

      <UnlockRequestHistoryDrawer
        open={unlockHistoryVisible}
        orderId={orderId}
        onClose={() => setUnlockHistoryVisible(false)}
      />
    </>
  );
}
