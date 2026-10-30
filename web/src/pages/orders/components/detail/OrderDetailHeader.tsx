import {
  AlertOutlined,
  DollarOutlined,
  DownOutlined,
  FileDoneOutlined,
  HistoryOutlined,
  SaveOutlined,
  UndoOutlined,
} from '@ant-design/icons';
import { Button, Dropdown, type MenuProps, Tooltip } from 'antd';
import React, { type ReactNode } from 'react';
import { useAccess } from '@/app/access';
import { OrderAllowedAction } from '@/enums.generated';
import { history } from '@/router/history';
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
  canReadAbnormal: boolean;
  /** 类型专属业务菜单（如拆票、改配），由订单类型详情扩展提供。 */
  businessMenuItems?: MenuProps['items'];
  moreMenuItems: MenuProps['items'];
  hasAction: (action: number) => boolean;
  onSave: () => void;
  onReset?: () => void;
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
  canReadAbnormal,
  businessMenuItems,
  moreMenuItems,
  hasAction,
  onSave,
  onReset,
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
  const canEdit = hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT);
  const renderActions = (
    lockActions?: ReactNode,
    onOpenHistory?: () => void,
  ) => {
    const businessItems: MenuProps['items'] = [
      ...(canCreatePod
        ? [
            {
              key: 'release-pod',
              icon: <FileDoneOutlined />,
              label: (
                <Tooltip title={businessWriteBlockedReason}>
                  <span>导出单证 (POD)</span>
                </Tooltip>
              ),
              disabled: businessWritesDisabled,
              onClick: onOpenReleasePod,
            },
          ]
        : []),
      ...(canReadAbnormal
        ? [
            {
              key: 'abnormal-case',
              icon: <AlertOutlined />,
              label: '异常情况',
              onClick: onOpenAbnormalCase,
            },
          ]
        : []),
      ...(businessMenuItems ?? []),
    ];
    const recordItems: MenuProps['items'] = [
      ...(onOpenHistory
        ? [
            {
              key: 'unlock-history',
              icon: <HistoryOutlined />,
              label: '撤销审核记录',
              onClick: onOpenHistory,
            },
          ]
        : []),
      ...(moreMenuItems ?? []),
    ];
    const resetItems: MenuProps['items'] =
      canEdit && onReset
        ? [
            {
              key: 'reset',
              icon: <UndoOutlined />,
              label: '重置修改',
              danger: true,
              disabled: businessWritesDisabled || saving,
              onClick: onReset,
            },
          ]
        : [];
    const groups = [businessItems, recordItems, resetItems].filter(
      (items) => items.length > 0,
    );
    const menuItems = groups.flatMap((items, index) => [
      ...(index ? [{ type: 'divider' as const, key: `divider-${index}` }] : []),
      ...items,
    ]);
    return (
      <>
        {canManageFee && (
          <Button
            icon={<DollarOutlined />}
            onClick={() => history.push(`/orders/${kind}/${orderId}/fees`)}
          >
            费用录入
          </Button>
        )}
        {lockActions}
        {canEdit && (
          <Tooltip title={businessWriteBlockedReason}>
            <span>
              <Button
                type="primary"
                icon={<SaveOutlined />}
                loading={saving}
                disabled={businessWritesDisabled}
                onClick={onSave}
              >
                保存
              </Button>
            </span>
          </Tooltip>
        )}
        {menuItems.length > 0 && (
          <Dropdown menu={{ items: menuItems }} trigger={['click']}>
            <Button>
              更多操作 <DownOutlined />
            </Button>
          </Dropdown>
        )}
      </>
    );
  };

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
        access.canOperateOrganization(order.organizationId) ? (
          <OrderLockControl
            orderId={orderId}
            orderNo={order.orderNo}
            state={lockState}
            loading={lockStateLoading}
            error={lockStateError}
            onRetry={onRetryLockState}
            onSynchronize={onSynchronizeLockChange}
            renderActions={renderActions}
          />
        ) : (
          renderActions()
        )
      }
    />
  );
}
