import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import { orderFlowStatusMeta, statusText } from '@/constants/statusMeta';
import { orderServiceTransitionOrderStatus } from '@/services/roncin/orderService';

export type TransitionModalRef = {
  open: (order: API.Order) => void;
};

type TransitionModalProps = {
  onSuccess: () => void;
};

type TransitionFormValues = {
  targetStatus: number;
  reason?: string;
};

const TransitionModal = forwardRef<TransitionModalRef, TransitionModalProps>(
  function TransitionModal({ onSuccess }, ref) {
    const { message } = App.useApp();
    const formRef = useRef<ProFormInstance | undefined>(undefined);
    const [open, setOpen] = useState(false);
    const [record, setRecord] = useState<API.Order>();
    const [targetStatusOptions, setTargetStatusOptions] = useState<
      { label: string; value: number }[]
    >([]);

    useImperativeHandle(ref, () => ({
      open: (order) => {
        setRecord(order);
        formRef.current?.setFieldsValue({
          currentStatus:
            statusText(orderFlowStatusMeta, order.flowStatus ?? 0, '未知状态'),
          targetStatus: undefined,
          reason: undefined,
        });
        setTargetStatusOptions(
          (order.allowedTargetFlowStatuses ?? []).map((status) => ({
            label: statusText(orderFlowStatusMeta, status, '未知状态'),
            value: status,
          })),
        );
        setOpen(true);
      },
    }));

    return (
      <ModalForm<TransitionFormValues>
        title="订单状态流转"
        open={open}
        formRef={formRef}
        initialValues={
          record
            ? {
                currentStatus:
                  statusText(
                    orderFlowStatusMeta,
                    record.flowStatus ?? 0,
                    '未知状态',
                  ),
              }
            : undefined
        }
        modalProps={{
          destroyOnHidden: true,
          width: 520,
          onCancel: () => setOpen(false),
        }}
        onOpenChange={setOpen}
        onFinish={async (values) => {
          if (!record?.id || !record?.version) return false;
          await orderServiceTransitionOrderStatus(
            { id: record.id },
            {
              id: record.id,
              expectedVersion: record.version,
              targetFlowStatus: values.targetStatus,
              reason: values.reason,
            },
          );
          message.success('状态流转成功');
          setOpen(false);
          onSuccess();
          return true;
        }}
      >
        <ProFormText
          name="currentStatus"
          label="当前状态"
          readonly
          initialValue={
            statusText(
              orderFlowStatusMeta,
              record?.flowStatus ?? 0,
              '未知状态',
            )
          }
        />
        <ProFormSelect
          name="targetStatus"
          label="目标流转状态"
          rules={[{ required: true, message: '请选择目标状态' }]}
          options={targetStatusOptions}
          placeholder="请选择目标状态"
        />
        <ProFormTextArea
          name="reason"
          label="流转原因说明"
          placeholder="请输入状态变更原因说明（可选）"
          fieldProps={{ maxLength: 500, showCount: true }}
        />
      </ModalForm>
    );
  },
);

export default TransitionModal;
