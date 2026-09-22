import { type App, Input, Select, Space } from 'antd';
import { orderFlowStatusMeta, statusText } from '@/constants/statusMeta';
import { OrderClosureStatus, OrderTerminationStatus } from '@/enums.generated';
import {
  orderServiceTransitionOrderClosure,
  orderServiceTransitionOrderStatus,
  orderServiceTransitionOrderTermination,
} from '@/services/roncin/orderService';

type AppInstance = ReturnType<typeof App.useApp>;

type TransitionApp = Pick<AppInstance, 'modal' | 'message'>;

type ConfirmTransitionOptions = {
  canSubmit?: () => boolean;
  afterClose?: () => void;
};

/** 主流程状态流转确认弹窗：目标状态由用户点击的流程节点固定。 */
export function confirmOrderFlow(
  app: TransitionApp,
  order: API.Order,
  targetStatus: number,
  onCompleted: () => Promise<void> | void,
  options: ConfirmTransitionOptions = {},
) {
  const orderID = order.id;
  const expectedVersion = order.version;
  const currentStatus = order.flowStatus;
  if (
    !orderID ||
    expectedVersion === undefined ||
    currentStatus === undefined ||
    !targetStatus
  ) {
    app.message.error('订单数据不完整，请刷新后重试');
    return false;
  }

  const currentStatusLabel = statusText(
    orderFlowStatusMeta,
    currentStatus,
    '未知状态',
  );
  const targetStatusLabel = statusText(
    orderFlowStatusMeta,
    targetStatus,
    '未知状态',
  );
  let reason = '';

  app.modal.confirm({
    title: '流转订单状态',
    afterClose: options.afterClose,
    content: (
      <Space vertical style={{ width: '100%', marginTop: 12 }}>
        <div>
          当前状态：{currentStatusLabel} → 目标状态：{targetStatusLabel}
        </div>
        <Input.TextArea
          aria-label="流转原因"
          placeholder="请输入流转原因（选填）"
          maxLength={500}
          showCount
          onChange={(event) => {
            reason = event.target.value;
          }}
        />
      </Space>
    ),
    async onOk() {
      if (options.canSubmit?.() === false) return Promise.reject();
      const response = await orderServiceTransitionOrderStatus(
        { id: orderID },
        {
          id: orderID,
          expectedVersion,
          targetFlowStatus: targetStatus,
          reason: reason.trim() || undefined,
        },
      );
      if (!response?.data) return Promise.reject();
      app.message.success(`状态已流转至${targetStatusLabel}`);
      await onCompleted();
    },
  });
  return true;
}

/** 退关/终止状态流转确认弹窗：可选终止类型 + 必填原因。 */
export function confirmOrderTermination(
  app: TransitionApp,
  order: API.Order,
  targetStatus: number,
  onCompleted: () => Promise<void> | void,
  options: ConfirmTransitionOptions = {},
) {
  const orderID = order.id;
  const expectedVersion = order.version;
  if (!orderID || expectedVersion === undefined) {
    app.message.error('订单数据不完整，请刷新后重试');
    return false;
  }
  let reason = '';
  let terminationType = 3;
  const isRestore =
    targetStatus === OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE &&
    order.terminationStatus ===
      OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATED;
  app.modal.confirm({
    title:
      targetStatus === OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE
        ? isRestore
          ? '恢复订单'
          : '取消退关'
        : targetStatus ===
            OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING
          ? '发起退关/终止'
          : '完成退关/终止',
    afterClose: options.afterClose,
    content: (
      <Space vertical style={{ width: '100%', marginTop: 12 }}>
        {targetStatus !==
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE && (
          <Select
            defaultValue={3}
            style={{ width: '100%' }}
            options={[
              { label: '客户撤单', value: 1 },
              { label: '承运人取消', value: 2 },
              { label: '海关退关', value: 3 },
              { label: '操作取消', value: 4 },
              { label: '其他', value: 5 },
            ]}
            onChange={(value) => {
              terminationType = value;
            }}
          />
        )}
        <Input.TextArea
          placeholder="请输入原因（必填）"
          maxLength={500}
          showCount
          onChange={(event) => {
            reason = event.target.value;
          }}
        />
      </Space>
    ),
    async onOk() {
      if (options.canSubmit?.() === false) return Promise.reject();
      if (!reason.trim()) {
        app.message.error('请输入原因');
        return Promise.reject();
      }
      const response = await orderServiceTransitionOrderTermination(
        { id: orderID },
        {
          id: orderID,
          expectedVersion,
          targetStatus,
          terminationType:
            targetStatus ===
            OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE
              ? undefined
              : terminationType,
          reason: reason.trim(),
        },
      );
      if (!response?.data) return Promise.reject();
      app.message.success(
        isRestore
          ? '恢复订单成功'
          : targetStatus ===
              OrderTerminationStatus.ORDER_TERMINATION_STATUS_ACTIVE
            ? '取消退关成功'
            : '更新退关状态成功',
      );
      await onCompleted();
    },
  });
  return true;
}

/** 结案/反结案状态流转确认弹窗：两种操作均必填原因。 */
export function confirmOrderClosure(
  app: TransitionApp,
  order: API.Order,
  targetStatus: number,
  onCompleted: () => Promise<void> | void,
  options: ConfirmTransitionOptions = {},
) {
  const orderID = order.id;
  const expectedVersion = order.version;
  if (!orderID || expectedVersion === undefined) {
    app.message.error('订单数据不完整，请刷新后重试');
    return false;
  }
  let reason = '';
  const isReopen =
    targetStatus === OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN;
  app.modal.confirm({
    title: isReopen ? '反结案/重新激活订单' : '完结订单',
    afterClose: options.afterClose,
    content: (
      <Space vertical style={{ width: '100%', marginTop: 12 }}>
        <Input.TextArea
          placeholder={
            isReopen ? '请输入反结案原因（必填）' : '请输入完结原因（必填）'
          }
          maxLength={500}
          showCount
          onChange={(event) => {
            reason = event.target.value;
          }}
        />
      </Space>
    ),
    async onOk() {
      if (options.canSubmit?.() === false) return Promise.reject();
      if (!reason.trim()) {
        app.message.error(isReopen ? '请输入反结案原因' : '请输入完结原因');
        return Promise.reject();
      }
      const response = await orderServiceTransitionOrderClosure(
        { id: orderID },
        {
          id: orderID,
          expectedVersion,
          targetStatus,
          reason: reason.trim(),
        },
      );
      if (!response?.data) return Promise.reject();
      app.message.success(isReopen ? '反结案成功' : '完结订单成功');
      await onCompleted();
    },
  });
  return true;
}
