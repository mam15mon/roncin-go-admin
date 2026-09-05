import { type App, Input, Select, Space } from 'antd';
import {
  orderServiceTransitionOrderClosure,
  orderServiceTransitionOrderTermination,
} from '@/services/roncin/orderService';

type AppInstance = ReturnType<typeof App.useApp>;

type TransitionApp = Pick<AppInstance, 'modal' | 'message'>;

/** 退关/终止状态流转确认弹窗：可选终止类型 + 必填原因。 */
export function confirmOrderTermination(
  app: TransitionApp,
  order: API.Order,
  targetStatus: number,
  onCompleted: () => Promise<void> | void,
  canSubmit: () => boolean = () => true,
) {
  const orderID = order.id;
  const expectedVersion = order.version;
  if (!orderID || expectedVersion === undefined) {
    app.message.error('订单数据不完整，请刷新后重试');
    return;
  }
  let reason = '';
  let terminationType = 3;
  app.modal.confirm({
    title:
      targetStatus === 1
        ? '取消退关/终止'
        : targetStatus === 2
          ? '发起退关/终止'
          : '完成退关/终止',
    content: (
      <Space vertical style={{ width: '100%', marginTop: 12 }}>
        {targetStatus !== 1 && (
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
      if (!canSubmit()) return Promise.reject();
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
          terminationType: targetStatus === 1 ? undefined : terminationType,
          reason,
        },
      );
      if (response.data) {
        app.message.success('更新退关状态成功');
        await onCompleted();
      }
    },
  });
}

/** 结案/反结案状态流转确认弹窗：反结案必填原因，结案原因选填。 */
export function confirmOrderClosure(
  app: TransitionApp,
  order: API.Order,
  targetStatus: number,
  onCompleted: () => Promise<void> | void,
  canSubmit: () => boolean = () => true,
) {
  const orderID = order.id;
  const expectedVersion = order.version;
  if (!orderID || expectedVersion === undefined) {
    app.message.error('订单数据不完整，请刷新后重试');
    return;
  }
  let reason = '';
  app.modal.confirm({
    title: targetStatus === 1 ? '反结案/重新激活订单' : '完结订单',
    content: (
      <Space vertical style={{ width: '100%', marginTop: 12 }}>
        <Input.TextArea
          placeholder={
            targetStatus === 1
              ? '请输入反结案原因（必填）'
              : '请输入完结原因（选填）'
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
      if (!canSubmit()) return Promise.reject();
      if (targetStatus === 1 && !reason.trim()) {
        app.message.error('请输入反结案原因');
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
      if (response.data) {
        app.message.success(targetStatus === 1 ? '反结案成功' : '完结订单成功');
        await onCompleted();
      }
    },
  });
}
