import {
  HistoryOutlined,
  LockOutlined,
  UnlockOutlined,
} from '@ant-design/icons';
import {
  Alert,
  App,
  Button,
  Form,
  Input,
  Modal,
  Spin,
  Tag,
  Tooltip,
} from 'antd';
import React, { type ReactNode, useContext, useEffect, useState } from 'react';
import { appInitialStateContext } from '@/app/initialStateContext';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';
import { GuardedModalForm } from '@/components/ui';
import { OrderBusinessType } from '@/enums.generated';
import {
  FeeReviewSummary,
  useOrderFeeReview,
} from '@/features/orders/fee-review';
import {
  orderLockServiceLockOrder,
  orderLockServiceRequestOrderUnlock,
} from '@/services/roncin/orderLockService';
import { isWriteResponse } from '@/utils/api';
import { formatDate } from '@/utils/format';
import { generateUUID } from '@/utils/uuid';
import { getOrderBusinessTypeLabel } from '../../use-order-lock-state';
import OrderLockFeeDetails from './OrderLockFeeDetails';
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
  /** 详情页将低频记录并入菜单；其它入口保留完整动作。 */
  renderActions?: (actions: ReactNode, onOpenHistory: () => void) => ReactNode;
};

type OrderLockStatusTagProps = Pick<
  OrderLockControlProps,
  'state' | 'loading' | 'error'
>;

function generateIdempotencyKey(): string {
  return generateUUID();
}

export function getOrderLockConfirmationDescription(
  businessType?: number,
): string {
  if (businessType === OrderBusinessType.BUSINESS_TYPE_SE) {
    return '锁定后将固定 MBL/HBL 提单不可变版本，订单业务资料和费用进入审核锁定。锁后费用变更须申请补录并补审，已建账费用须先完成账务回退。';
  }
  return '审核后订单业务资料和费用进入锁定。锁后费用变更须申请补录并补审，已建账费用须先完成账务回退。';
}

/** 审核锁仅展示服务端记录的人工审核人。 */
export function getOrderLockSourceDisplay(
  state: Pick<API.OrderLockStateData, 'lockedBy' | 'lockedByName'>,
) {
  return { actorText: state.lockedByName || state.lockedBy || '审核人待确认' };
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
    return <Tag icon={<UnlockOutlined />}>{businessTypeLabel} · 未审核</Tag>;
  }

  const timeText = state.lockedAt ? formatDate(state.lockedAt, 'minute') : '';
  const sourceDisplay = getOrderLockSourceDisplay(state);
  return (
    <Tooltip
      title={
        <div style={{ fontSize: 12 }}>
          <div>业务类型：{businessTypeLabel}</div>
          <div>锁定轮次：第 {state.lockGeneration} 代</div>
          <div>审核人：{sourceDisplay.actorText}</div>
          {timeText && <div>审核时间：{timeText}</div>}
          {state.currentLockRecord?.reviewComment && (
            <div>审核意见：{state.currentLockRecord.reviewComment}</div>
          )}
          {state.currentLockRecord?.feeContentVersion && (
            <div>审核费用版本：{state.currentLockRecord.feeContentVersion}</div>
          )}
          {state.activeUnlockRequest && (
            <div style={{ marginTop: 4, color: '#faad14' }}>
              当前解锁申请：{state.activeUnlockRequest.status}（
              {state.activeUnlockRequest.route}）
            </div>
          )}
        </div>
      }
    >
      <Tag color="processing" icon={<LockOutlined />}>
        {businessTypeLabel} · 已审核锁定 · {sourceDisplay.actorText}
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
  renderActions,
}: OrderLockControlProps) {
  const { message } = App.useApp();
  const workspaceId = useContext(appInitialStateContext)?.initialState
    ?.currentUser?.currentOrganization?.id;
  const [lockSnapshot, setLockSnapshot] = useState<API.OrderLockStateData>();
  const [lockKey, setLockKey] = useState(generateIdempotencyKey);
  const feeReview = useOrderFeeReview(orderId, Boolean(lockSnapshot));
  const feeReviewMatchesSnapshot =
    feeReview.data?.orderId === lockSnapshot?.orderId &&
    feeReview.data?.contentVersion === lockSnapshot?.feeContentVersion &&
    state?.orderVersion === lockSnapshot?.orderVersion &&
    state?.feeContentFingerprint === lockSnapshot?.feeContentFingerprint;
  const feeReviewReady =
    feeReview.isSuccess && !feeReview.isFetching && feeReviewMatchesSnapshot;
  const [unlocking, setUnlocking] = useState(false);
  const [unlockModalVisible, setUnlockModalVisible] = useState(false);
  const { markDirty: markUnlockDirty } = useUnsavedChangesGuard({
    enabled: unlockModalVisible,
    identity: orderId,
  });
  const [unlockHistoryVisible, setUnlockHistoryVisible] = useState(false);
  const [unlockRoute, setUnlockRoute] = useState<UnlockRoute>('ROLE_DIRECT');
  const [unlockForm] = Form.useForm<{ reason?: string }>();

  useEffect(() => {
    if (unlockModalVisible && (loading || error || !state?.isLocked)) {
      setUnlockModalVisible(false);
      unlockForm.resetFields();
    }
  }, [error, loading, state, unlockForm, unlockModalVisible]);

  useEffect(() => {
    setLockSnapshot(undefined);
    setUnlockModalVisible(false);
    setUnlockHistoryVisible(false);
  }, [orderId, workspaceId]);

  const expectedOrderVersion = state?.orderVersion;
  const versionMissing = !expectedOrderVersion;
  const reviewSnapshotMissing =
    versionMissing ||
    !state?.feeContentVersion ||
    !state?.feeContentFingerprint;
  const businessTypeLabel = getOrderBusinessTypeLabel(state?.businessType);

  const handleLock = () => {
    if (
      !state?.canLock ||
      !state.orderVersion ||
      !state.feeContentVersion ||
      !state.feeContentFingerprint
    ) {
      message.warning('订单审核快照或版本已变化，请刷新后重试');
      return;
    }
    setLockKey(generateIdempotencyKey());
    setLockSnapshot({ ...state, orderId });
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

  const handleUnlockSubmit = async (values: {
    reason?: string;
  }): Promise<false | undefined> => {
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
      if (!isWriteResponse(response) || !response.data?.request?.id)
        return false;
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

    return undefined;
  };

  const lockBlockedTip = reviewSnapshotMissing
    ? '订单或费用审核快照缺失，请刷新后重试'
    : state?.lockBlockedReasons?.join('；');
  const unlockBlockedTip = versionMissing
    ? '订单版本信息缺失，请刷新后重试'
    : state?.unlockBlockedReasons?.join('；');

  const primaryActions = (
    <>
      {!loading && (error || !state) && (
        <Button onClick={() => void onRetry()}>重试锁状态</Button>
      )}
      {state && !state.isLocked && (
        <Tooltip
          title={
            !state.canLock || reviewSnapshotMissing ? lockBlockedTip : undefined
          }
        >
          <span>
            <Button
              icon={<LockOutlined />}
              disabled={!state.canLock || reviewSnapshotMissing || loading}
              onClick={handleLock}
            >
              审核并锁定
            </Button>
          </span>
        </Tooltip>
      )}

      {state?.isLocked && state.canRoleDirectUnlock && (
        <Button
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
              icon={<UnlockOutlined />}
              disabled={versionMissing || loading}
              onClick={() => openUnlockModal('DINGTALK_APPROVAL')}
            >
              申请解锁
            </Button>
          </span>
        </Tooltip>
      )}
    </>
  );

  const openHistory = () => setUnlockHistoryVisible(true);
  return (
    <>
      {renderActions ? (
        renderActions(primaryActions, openHistory)
      ) : (
        <>
          <Button icon={<HistoryOutlined />} onClick={openHistory}>
            解锁记录
          </Button>
          {primaryActions}
        </>
      )}

      {lockSnapshot && (
        <GuardedModalForm
          key={lockKey}
          title={`审核并锁定${getOrderBusinessTypeLabel(lockSnapshot.businessType)}订单`}
          open
          width={1280}
          modalProps={{
            destroyOnHidden: true,
            styles: { body: { maxHeight: '70vh', overflowY: 'auto' } },
          }}
          submitter={{
            searchConfig: { submitText: '确认审核' },
            submitButtonProps: { disabled: !feeReviewReady },
          }}
          onOpenChange={(open) => {
            if (!open) setLockSnapshot(undefined);
          }}
          onFinish={async () => {
            if (
              !feeReviewReady ||
              lockSnapshot.orderId !== orderId ||
              !lockSnapshot.orderVersion ||
              !lockSnapshot.feeContentVersion ||
              !lockSnapshot.feeContentFingerprint
            )
              return false;
            try {
              const response = await orderLockServiceLockOrder(
                { orderId },
                {
                  orderId,
                  expectedOrderVersion: lockSnapshot.orderVersion,
                  expectedFeeContentVersion: lockSnapshot.feeContentVersion,
                  feeContentFingerprint: lockSnapshot.feeContentFingerprint,
                  idempotencyKey: lockKey,
                },
              );
              if (
                !isWriteResponse(response) ||
                response.data?.state?.orderId !== orderId
              )
                return false;
              message.success('订单已审核并锁定');
              await onSynchronize();
              setLockSnapshot(undefined);
              return true;
            } catch {
              return false;
            }
          }}
        >
          <p>订单：{orderNo || lockSnapshot.orderNo || orderId}</p>
          <p>
            {getOrderLockConfirmationDescription(lockSnapshot.businessType)}
          </p>
          {Number(lockSnapshot.feeSummary?.profit) < 0 && (
            <Alert
              type="warning"
              showIcon
              title="该订单亏损，请核对每笔应收、应付费用后确认审核。"
              style={{ marginBlock: 12 }}
            />
          )}
          {feeReview.isFetching && (
            <div role="status" style={{ padding: 24, textAlign: 'center' }}>
              <Spin /> 正在加载全部费用明细
            </div>
          )}
          {feeReview.isError && (
            <Alert
              type="error"
              showIcon
              title="费用明细加载失败，请重试后审核"
              action={
                <Button onClick={() => void feeReview.refetch()}>
                  重试费用明细
                </Button>
              }
            />
          )}
          {feeReview.isSuccess &&
            !feeReview.isFetching &&
            !feeReviewMatchesSnapshot && (
              <Alert
                type="warning"
                showIcon
                title="订单或费用已变化，请刷新后重新审核"
                action={
                  <Button
                    onClick={async () => {
                      setLockSnapshot(undefined);
                      await onRetry();
                    }}
                  >
                    刷新审核信息
                  </Button>
                }
              />
            )}
          {feeReviewReady && (
            <>
              <FeeReviewSummary summary={lockSnapshot.feeSummary} />
              <OrderLockFeeDetails fees={feeReview.data.fees || []} />
            </>
          )}
        </GuardedModalForm>
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
        <Form
          form={unlockForm}
          layout="vertical"
          onFinish={handleUnlockSubmit}
          onValuesChange={markUnlockDirty}
        >
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
