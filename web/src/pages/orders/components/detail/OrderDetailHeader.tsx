import {
  AlertOutlined,
  DollarOutlined,
  DownOutlined,
  FileDoneOutlined,
  HistoryOutlined,
  LockOutlined,
  SaveOutlined,
  ScissorOutlined,
  SwapOutlined,
  UnlockOutlined,
} from '@ant-design/icons';
import { history } from '@umijs/max';
import {
  Button,
  Dropdown,
  Form,
  Input,
  type MenuProps,
  Modal,
  message,
  Tag,
  Tooltip,
} from 'antd';
import dayjs from 'dayjs';
import React, { useState } from 'react';
import { DocumentDetailLayout } from '@/components/ui/document-detail-layout';
import {
  OrderAllowedAction,
  OrderClosureStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import {
  orderLockServiceLockOrder,
  orderLockServiceRequestOrderUnlock,
} from '@/services/roncin/orderLockService';
import UnlockRequestHistoryDrawer from './UnlockRequestHistoryDrawer';

function generateIdempotencyKey(): string {
  if (typeof crypto !== 'undefined' && crypto.randomUUID) {
    return crypto.randomUUID();
  }
  return 'xxxxxxxx-xxxx-4xxx-yxxx-xxxxxxxxxxxx'.replace(/[xy]/g, (c) => {
    const r = (Math.random() * 16) | 0;
    const v = c === 'x' ? r : (r & 0x3) | 0x8;
    return v.toString(16);
  });
}

type OrderDetailHeaderProps = {
  kind: string;
  orderId: string;
  configTitle: string;
  businessType: string;
  order: API.Order;
  saving: boolean;
  canManageFee: boolean;
  canCreatePod: boolean;
  canCreateAbnormal: boolean;
  canSplit?: boolean;
  canReassign?: boolean;
  splitDisabled?: boolean;
  splitBlockedReasons?: string[];
  reassignDisabled?: boolean;
  reassignBlockedReasons?: string[];
  moreMenuItems: MenuProps['items'];
  hasAction: (action: number) => boolean;
  onSave: () => void;
  onConfirmTermination: (targetStatus: number) => void;
  onConfirmClosure: (targetStatus: number) => void;
  onOpenReleasePod: () => void;
  onOpenAbnormalCase: () => void;
  onOpenSplit?: () => void;
  onOpenReassign?: () => void;
  lockState?: API.OrderLockStateData | null;
  onRefreshLockState?: () => void;
};

export default function OrderDetailHeader({
  kind,
  orderId,
  configTitle,
  businessType,
  order,
  saving,
  canManageFee,
  canCreatePod,
  canCreateAbnormal,
  canSplit,
  canReassign,
  splitDisabled,
  splitBlockedReasons,
  reassignDisabled,
  reassignBlockedReasons,
  moreMenuItems,
  hasAction,
  onSave,
  onConfirmTermination,
  onConfirmClosure,
  onOpenReleasePod,
  onOpenAbnormalCase,
  onOpenSplit,
  onOpenReassign,
  lockState,
  onRefreshLockState,
}: OrderDetailHeaderProps) {
  const [locking, setLocking] = useState(false);
  const [unlocking, setUnlocking] = useState(false);
  const [unlockModalVisible, setUnlockModalVisible] = useState(false);
  const [unlockHistoryVisible, setUnlockHistoryVisible] = useState(false);
  const [unlockRoute, setUnlockRoute] = useState<
    'ROLE_DIRECT' | 'ADMIN_EMERGENCY' | 'DINGTALK_APPROVAL'
  >('ROLE_DIRECT');
  const [unlockForm] = Form.useForm<{ reason?: string }>();

  const isSeaExport = businessType === 'SE' || kind === 'sea-export';
  const isLocked = Boolean(lockState?.isLocked);
  const businessWritesDisabled = isSeaExport && (!lockState || isLocked);
  const businessWriteBlockedTip = !lockState
    ? '订单锁定状态加载失败，请刷新后重试'
    : isLocked
      ? '订单已锁定，如需修改请先申请解锁'
      : undefined;

  const handleLock = () => {
    Modal.confirm({
      title: '锁定海运出口订单',
      content: (
        <div>
          <p>
            确定要锁定订单 <b>{order.orderNo}</b> 吗？
          </p>
          <p style={{ color: '#64748b' }}>
            锁定后将固定提单不可变版本，订单及关联合同、箱货、单证将禁止常规业务编辑。如需修改需由业务角色或管理员解锁。
          </p>
        </div>
      ),
      okText: '确认锁定',
      cancelText: '取消',
      okButtonProps: { danger: true },
      onOk: async () => {
        setLocking(true);
        try {
          const resp = await orderLockServiceLockOrder(
            { orderId },
            {
              orderId,
              expectedOrderVersion: String(order.version || 1),
              idempotencyKey: generateIdempotencyKey(),
            },
          );
          if (resp?.success) {
            message.success('订单已成功锁定');
            onRefreshLockState?.();
          }
        } catch (err: unknown) {
          message.error(err instanceof Error ? err.message : '锁定订单失败');
        } finally {
          setLocking(false);
        }
      },
    });
  };

  const openUnlockModal = (
    route: 'ROLE_DIRECT' | 'ADMIN_EMERGENCY' | 'DINGTALK_APPROVAL',
  ) => {
    setUnlockRoute(route);
    unlockForm.resetFields();
    setUnlockModalVisible(true);
  };

  const handleUnlockSubmit = async (values: { reason?: string }) => {
    setUnlocking(true);
    try {
      const resp = await orderLockServiceRequestOrderUnlock(
        { orderId },
        {
          orderId,
          expectedOrderVersion: String(order.version || 1),
          idempotencyKey: generateIdempotencyKey(),
          reason: values.reason?.trim() || undefined,
        },
      );
      if (resp?.success) {
        const actualRoute = resp.data?.request?.route;
        const actualStatus = resp.data?.request?.status;
        if (
          actualStatus === 'CONFIGURATION_FAILED' ||
          actualStatus === 'DISPATCH_FAILED' ||
          actualStatus === 'DISPATCH_UNKNOWN'
        ) {
          message.error(
            resp.data?.request?.failureMessage ||
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
        onRefreshLockState?.();
        if (actualRoute === 'DINGTALK_APPROVAL') {
          setUnlockHistoryVisible(true);
        }
      }
    } catch (err: unknown) {
      message.error(err instanceof Error ? err.message : '处理解锁失败');
    } finally {
      setUnlocking(false);
    }
  };

  const renderLockTag = () => {
    if (!isSeaExport || !lockState) return null;
    if (lockState.isLocked) {
      const timeStr = lockState.lockedAt
        ? dayjs(lockState.lockedAt).format('YYYY-MM-DD HH:mm')
        : '';
      const tooltipContent = (
        <div style={{ fontSize: 12 }}>
          <div>锁定轮次：第 {lockState.lockGeneration} 代</div>
          <div>
            锁定人：{lockState.lockedByName || lockState.lockedBy || '系统'}
          </div>
          {lockState.lockedAt && <div>锁定时间：{timeStr}</div>}
          {lockState.activeUnlockRequest && (
            <div style={{ marginTop: 4, color: '#faad14' }}>
              当前解锁申请：{lockState.activeUnlockRequest.status} (
              {lockState.activeUnlockRequest.route})
            </div>
          )}
        </div>
      );
      return (
        <Tooltip title={tooltipContent}>
          <Tag color="error" icon={<LockOutlined />}>
            已锁定 · {lockState.lockedByName || '已锁定'}
            {timeStr ? ` · ${timeStr}` : ''}
          </Tag>
        </Tooltip>
      );
    }
    return (
      <Tag color="success" icon={<UnlockOutlined />}>
        未锁定
      </Tag>
    );
  };

  return (
    <>
      <DocumentDetailLayout
        breadcrumbs={[
          { label: configTitle, path: `/orders/${kind}` },
          { label: `${configTitle}详情` },
        ]}
        code={order.orderNo}
        extraBreadcrumb={renderLockTag()}
        actions={
          <>
            {isSeaExport && (
              <Button
                icon={<HistoryOutlined />}
                onClick={() => setUnlockHistoryVisible(true)}
              >
                解锁记录
              </Button>
            )}
            {/* 海运出口订单锁定操作 */}
            {isSeaExport && !isLocked && (
              <Tooltip
                title={
                  !lockState?.canLock &&
                  lockState?.lockBlockedReasons &&
                  lockState.lockBlockedReasons.length > 0
                    ? lockState.lockBlockedReasons.join('；')
                    : undefined
                }
              >
                <span>
                  <Button
                    style={{ color: '#d4380d', borderColor: '#ffbb96' }}
                    icon={<LockOutlined />}
                    disabled={!lockState?.canLock}
                    loading={locking}
                    onClick={handleLock}
                  >
                    锁定订单
                  </Button>
                </span>
              </Tooltip>
            )}

            {/* 海运出口订单解锁操作 */}
            {isSeaExport && isLocked && (
              <>
                {lockState?.canRoleDirectUnlock && (
                  <Button
                    type="primary"
                    style={{
                      backgroundColor: '#fa8c16',
                      borderColor: '#fa8c16',
                    }}
                    icon={<UnlockOutlined />}
                    onClick={() => openUnlockModal('ROLE_DIRECT')}
                  >
                    直接解锁
                  </Button>
                )}
                {lockState?.canAdminEmergencyUnlock && (
                  <Button
                    danger
                    type="primary"
                    icon={<UnlockOutlined />}
                    onClick={() => openUnlockModal('ADMIN_EMERGENCY')}
                  >
                    紧急解锁
                  </Button>
                )}
                {lockState?.canRequestUnlock && (
                  <Tooltip
                    title={
                      lockState?.unlockBlockedReasons &&
                      lockState.unlockBlockedReasons.length > 0
                        ? lockState.unlockBlockedReasons.join('；')
                        : undefined
                    }
                  >
                    <span>
                      <Button
                        style={{ color: '#1677ff', borderColor: '#1677ff' }}
                        icon={<UnlockOutlined />}
                        disabled={Boolean(
                          lockState?.unlockBlockedReasons &&
                            lockState.unlockBlockedReasons.length > 0,
                        )}
                        onClick={() => openUnlockModal('DINGTALK_APPROVAL')}
                      >
                        申请解锁
                      </Button>
                    </span>
                  </Tooltip>
                )}
              </>
            )}

            {/* 实心蓝底主保存按钮 */}
            {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT) && (
              <Tooltip title={businessWriteBlockedTip}>
                <span>
                  <Button
                    type="primary"
                    icon={<SaveOutlined />}
                    loading={saving}
                    disabled={businessWritesDisabled}
                    onClick={onSave}
                    style={{ fontWeight: 500 }}
                  >
                    保存
                  </Button>
                </span>
              </Tooltip>
            )}

            {hasAction(
              OrderAllowedAction.ORDER_ALLOWED_ACTION_START_TERMINATION,
            ) && (
              <Button
                danger
                disabled={businessWritesDisabled}
                onClick={() =>
                  onConfirmTermination(
                    OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
                  )
                }
              >
                发起退关
              </Button>
            )}
            {hasAction(
              OrderAllowedAction.ORDER_ALLOWED_ACTION_COMPLETE_TERMINATION,
            ) && (
              <Button
                danger
                type="primary"
                disabled={businessWritesDisabled}
                onClick={() =>
                  onConfirmTermination(
                    OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
                  )
                }
              >
                完成退关
              </Button>
            )}
            {hasAction(
              OrderAllowedAction.ORDER_ALLOWED_ACTION_CANCEL_TERMINATION,
            ) && (
              <Button
                disabled={businessWritesDisabled}
                onClick={() =>
                  onConfirmTermination(
                    OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE,
                  )
                }
              >
                取消退关
              </Button>
            )}
            {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_CLOSE) && (
              <Button
                type="primary"
                disabled={businessWritesDisabled}
                onClick={() =>
                  onConfirmClosure(
                    OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED,
                  )
                }
              >
                完结订单
              </Button>
            )}
            {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_REOPEN) && (
              <Button
                disabled={businessWritesDisabled}
                onClick={() =>
                  onConfirmClosure(OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN)
                }
              >
                反结案
              </Button>
            )}

            {/* 费用录入（直达独立全屏费用工作台页面） */}
            {canManageFee && (
              <Button
                type="primary"
                icon={<DollarOutlined />}
                onClick={() => history.push(`/orders/${kind}/${orderId}/fees`)}
                style={{ fontWeight: 500 }}
              >
                费用录入
              </Button>
            )}

            {/* 导出单证 / 放货凭证 POD */}
            {canCreatePod && (
              <Button
                style={{ color: '#1677ff', borderColor: '#1677ff' }}
                icon={<FileDoneOutlined />}
                onClick={onOpenReleasePod}
              >
                导出单证 (POD)
              </Button>
            )}

            {/* 异常情况 */}
            {canCreateAbnormal && (
              <Button
                style={{ color: '#ff4d4f', borderColor: '#ff4d4f' }}
                icon={<AlertOutlined />}
                onClick={onOpenAbnormalCase}
              >
                异常情况
              </Button>
            )}

            {/* 拆票 */}
            {canSplit && (
              <Tooltip
                title={
                  businessWritesDisabled
                    ? businessWriteBlockedTip
                    : splitDisabled &&
                        splitBlockedReasons &&
                        splitBlockedReasons.length > 0
                      ? splitBlockedReasons.join('；')
                      : undefined
                }
              >
                <span>
                  <Button
                    style={{ color: '#722ed1', borderColor: '#722ed1' }}
                    icon={<ScissorOutlined />}
                    disabled={splitDisabled || businessWritesDisabled}
                    onClick={onOpenSplit}
                  >
                    拆票
                  </Button>
                </span>
              </Tooltip>
            )}

            {/* 改配 */}
            {canReassign && (
              <Tooltip
                title={
                  businessWritesDisabled
                    ? businessWriteBlockedTip
                    : reassignDisabled &&
                        reassignBlockedReasons &&
                        reassignBlockedReasons.length > 0
                      ? reassignBlockedReasons.join('；')
                      : undefined
                }
              >
                <span>
                  <Button
                    style={{ color: '#fa8c16', borderColor: '#fa8c16' }}
                    icon={<SwapOutlined />}
                    disabled={reassignDisabled || businessWritesDisabled}
                    onClick={onOpenReassign}
                  >
                    改配
                  </Button>
                </span>
              </Tooltip>
            )}

            {/* 更多操作 */}
            <Dropdown menu={{ items: moreMenuItems }} trigger={['click']}>
              <Button style={{ color: '#64748b', borderColor: '#d9d9d9' }}>
                更多操作 <DownOutlined style={{ fontSize: 10 }} />
              </Button>
            </Dropdown>
          </>
        }
      >
        {null}
      </DocumentDetailLayout>

      {/* 解锁弹窗 */}
      <Modal
        title={
          unlockRoute === 'ROLE_DIRECT'
            ? '直接解锁海运出口订单'
            : unlockRoute === 'ADMIN_EMERGENCY'
              ? '系统管理员紧急解锁'
              : '申请解锁海运出口订单'
        }
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
            '您具备海运出口锁定业务角色权限，可以直接解除订单业务锁定。解锁后订单将恢复可编辑。'}
          {unlockRoute === 'ADMIN_EMERGENCY' &&
            '您正在以系统管理员身份执行紧急解锁。该操作将直接解除业务锁定并记录安全审计追踪。'}
          {unlockRoute === 'DINGTALK_APPROVAL' &&
            '提交解锁申请后，将生成审批任务并指派给具备锁定权限的业务角色成员审批。'}
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
