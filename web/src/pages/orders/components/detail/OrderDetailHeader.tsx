import {
  AlertOutlined,
  DollarOutlined,
  DownOutlined,
  FileDoneOutlined,
  SaveOutlined,
  ScissorOutlined,
  SwapOutlined,
} from '@ant-design/icons';
import { history } from '@umijs/max';
import { Button, Dropdown, Tooltip, type MenuProps } from 'antd';
import React from 'react';
import { DocumentDetailLayout } from '@/components/ui/document-detail-layout';
import {
  OrderAllowedAction,
  OrderClosureStatus,
  OrderTerminationStatus,
} from '@/enums.generated';

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
};

export default function OrderDetailHeader({
  kind,
  orderId,
  configTitle,
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
}: OrderDetailHeaderProps) {
  return (
    <DocumentDetailLayout
      breadcrumbs={[
        { label: configTitle, path: `/orders/${kind}` },
        { label: `${configTitle}详情` },
      ]}
      code={order.orderNo}
      actions={
        <>
          {/* 实心蓝底主保存按钮 */}
          {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_EDIT) && (
            <Button
              type="primary"
              icon={<SaveOutlined />}
              loading={saving}
              onClick={onSave}
              style={{ fontWeight: 500 }}
            >
              保存
            </Button>
          )}

          {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_START_TERMINATION) && (
            <Button
              danger
              onClick={() =>
                onConfirmTermination(
                  OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
                )
              }
            >
              发起退关
            </Button>
          )}
          {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_COMPLETE_TERMINATION) && (
            <Button
              danger
              type="primary"
              onClick={() =>
                onConfirmTermination(
                  OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED,
                )
              }
            >
              完成退关
            </Button>
          )}
          {hasAction(OrderAllowedAction.ORDER_ALLOWED_ACTION_CANCEL_TERMINATION) && (
            <Button
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
              onClick={() =>
                onConfirmClosure(
                  OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
                )
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
                splitDisabled && splitBlockedReasons && splitBlockedReasons.length > 0
                  ? splitBlockedReasons.join('；')
                  : undefined
              }
            >
              <span>
                <Button
                  style={{ color: '#722ed1', borderColor: '#722ed1' }}
                  icon={<ScissorOutlined />}
                  disabled={splitDisabled}
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
                reassignDisabled && reassignBlockedReasons && reassignBlockedReasons.length > 0
                  ? reassignBlockedReasons.join('；')
                  : undefined
              }
            >
              <span>
                <Button
                  style={{ color: '#fa8c16', borderColor: '#fa8c16' }}
                  icon={<SwapOutlined />}
                  disabled={reassignDisabled}
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
  );
}
