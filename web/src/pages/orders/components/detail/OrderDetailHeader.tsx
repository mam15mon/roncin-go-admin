import {
  AlertOutlined,
  DollarOutlined,
  DownOutlined,
  FileDoneOutlined,
  SaveOutlined,
  UndoOutlined,
} from '@ant-design/icons';
import { history } from '@/router/history';
import { useAccess } from '@/app/access';
import { Button, Dropdown, type MenuProps, Tooltip } from 'antd';
import React, { type ReactNode } from 'react';
import {
  OrderAllowedAction,
  OrderClosureStatus,
  OrderTerminationStatus,
} from '@/enums.generated';
import OrderPageHeader from '../OrderPageHeader';
import OrderLockControl, { OrderLockStatusTag } from './OrderLockControl';

type OrderDetailHeaderProps = {
  kind: string;
  navigationTitle: string;
  orderId: string;
  order: API.Order;
  saving: boolean;
  canManageFee: boolean;
  canCreatePod: boolean;
  canCreateAbnormal: boolean;
  /** 类型专属头部动作插槽（如拆票、改配），由订单类型详情扩展提供。 */
  businessActions?: ReactNode;
  moreMenuItems: MenuProps['items'];
  hasAction: (action: number) => boolean;
  onSave: () => void;
  onReset?: () => void;
  onConfirmTermination: (targetStatus: number) => void;
  onConfirmClosure: (targetStatus: number) => void;
  onOpenReleasePod: () => void;
  onOpenAbnormalCase: () => void;
  lockState: API.OrderLockStateData | null;
  lockStateLoading: boolean;
  lockStateError: Error | null;
  businessWritesDisabled: boolean;
  businessWriteBlockedReason?: string;
  onRetryLockState: () => Promise<API.OrderLockStateData | null>;
  onSynchronizeLockChange: () => Promise<void>;
};

export default function OrderDetailHeader({
  kind,
  navigationTitle,
  orderId,
  order,
  saving,
  canManageFee,
  canCreatePod,
  canCreateAbnormal,
  businessActions,
  moreMenuItems,
  hasAction,
  onSave,
  onReset,
  onConfirmTermination,
  onConfirmClosure,
  onOpenReleasePod,
  onOpenAbnormalCase,
  lockState,
  lockStateLoading,
  lockStateError,
  businessWritesDisabled,
  businessWriteBlockedReason,
  onRetryLockState,
  onSynchronizeLockChange,
}: OrderDetailHeaderProps) {
  const access = useAccess();
  return (
    <OrderPageHeader
      page="detail"
      orderKind={kind}
      navigationTitle={navigationTitle}
      orderId={orderId}
      orderNo={order?.orderNo}
      tags={
        <OrderLockStatusTag
          state={lockState}
          loading={lockStateLoading}
          error={lockStateError}
        />
      }
      actions={
        <>
          {access.canOperateOrganization(order.organizationId) && (
            <OrderLockControl
              orderId={orderId}
              orderNo={order.orderNo}
              state={lockState}
              loading={lockStateLoading}
              error={lockStateError}
              onRetry={onRetryLockState}
              onSynchronize={onSynchronizeLockChange}
            />
          )}

          {/* 实心蓝底主保存按钮 */}
          {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT) && (
            <Tooltip title={businessWriteBlockedReason}>
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

          {/* 重置修改按钮 */}
          {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT) &&
            onReset && (
              <Button
                icon={<UndoOutlined />}
                disabled={businessWritesDisabled || saving}
                onClick={onReset}
              >
                重置修改
              </Button>
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
                onConfirmClosure(OrderClosureStatus.ORDER_CLOSURE_STATUS_CLOSED)
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
            <Tooltip title={businessWriteBlockedReason}>
              <span>
                <Button
                  style={{ color: '#1677ff', borderColor: '#1677ff' }}
                  icon={<FileDoneOutlined />}
                  disabled={businessWritesDisabled}
                  onClick={onOpenReleasePod}
                >
                  导出单证 (POD)
                </Button>
              </span>
            </Tooltip>
          )}

          {/* 异常情况 */}
          {canCreateAbnormal && (
            <Tooltip title={businessWriteBlockedReason}>
              <span>
                <Button
                  style={{ color: '#ff4d4f', borderColor: '#ff4d4f' }}
                  icon={<AlertOutlined />}
                  disabled={businessWritesDisabled}
                  onClick={onOpenAbnormalCase}
                >
                  异常情况
                </Button>
              </span>
            </Tooltip>
          )}

          {/* 类型专属动作（由订单类型详情扩展贡献，如拆票、改配） */}
          {businessActions}

          {/* 更多操作 */}
          <Dropdown menu={{ items: moreMenuItems }} trigger={['click']}>
            <Button style={{ color: '#64748b', borderColor: '#d9d9d9' }}>
              更多操作 <DownOutlined style={{ fontSize: 10 }} />
            </Button>
          </Dropdown>
        </>
      }
    />
  );
}
