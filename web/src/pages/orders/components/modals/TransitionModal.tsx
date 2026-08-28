import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormSelect,
  ProFormText,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { App } from 'antd';
import { forwardRef, useImperativeHandle, useRef, useState } from 'react';
import { seFlowStatusLabels } from '../../common';
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

const transitionsByStatus: Record<number, { label: string; value: number }[]> = {
  1: [{ label: '已订舱', value: 2 }],
  2: [{ label: '已配舱', value: 3 }],
  3: [
    { label: '拖车已安排', value: 4 },
    { label: '已截单（无拖车）', value: 5 },
  ],
  4: [{ label: '已截单', value: 5 }],
  5: [{ label: '报关已安排', value: 6 }],
  6: [{ label: '已放单', value: 7 }],
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
            seFlowStatusLabels[order.flowStatus ?? 0] ?? '未知状态',
          targetStatus: undefined,
          reason: undefined,
        });
        setTargetStatusOptions(transitionsByStatus[order.flowStatus ?? 0] ?? []);
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
                  seFlowStatusLabels[record.flowStatus ?? 0] ?? '未知状态',
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
            seFlowStatusLabels[record?.flowStatus ?? 0] ?? '未知状态'
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
