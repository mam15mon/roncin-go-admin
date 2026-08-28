import type { ProFormInstance } from '@ant-design/pro-components';
import {
  ModalForm,
  ProFormDigit,
  ProFormTextArea,
} from '@ant-design/pro-components';
import { ProFormSearchableSelect } from '@/components/ui';
import { Alert, App } from 'antd';
import React, { useRef, useState } from 'react';
import { settlementServiceCreateCommissionAdjustment } from '@/services/roncin/settlementService';
import { decimalText, type AdjustmentValues } from '../types';

type CommissionAdjustmentModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  detail?: API.FinanceCommission;
  onSuccess: () => void;
};

export default function CommissionAdjustmentModal({
  open,
  onOpenChange,
  detail,
  onSuccess,
}: CommissionAdjustmentModalProps) {
  const { message } = App.useApp();
  const formRef = useRef<ProFormInstance | undefined>(undefined);
  const [adjustmentIdempotencyKey] = useState(() =>
    globalThis.crypto.randomUUID(),
  );

  return (
    <ModalForm<AdjustmentValues>
      formRef={formRef}
      title={`新增提成调整${detail?.commissionNo ? ` · ${detail.commissionNo}` : ''}`}
      open={open}
      width={620}
      initialValues={{ direction: 'INCREASE' }}
      modalProps={{
        destroyOnHidden: true,
        onCancel: () => onOpenChange(false),
      }}
      onFinish={async (values) => {
        if (!detail?.id) return false;
        try {
          await settlementServiceCreateCommissionAdjustment(
            { commissionId: detail.id },
            {
              commissionId: detail.id,
              orderId: values.orderId,
              direction: values.direction,
              amount: String(values.amount),
              reason: values.reason,
              note: values.note,
              idempotencyKey: adjustmentIdempotencyKey,
            },
          );
          message.success('提成调整草稿已创建');
          onOpenChange(false);
          onSuccess();
          return true;
        } catch (error: any) {
          message.error(error.message || '提成调整创建失败');
          return false;
        }
      }}
    >
      <Alert
        type="warning"
        showIcon
        title="原始提成不会被修改"
        description="请选择产生差异的具体订单。增提或冲减会形成独立编号，并保留确认、发放/扣回和取消轨迹。冲减金额不能使有效提成小于零。"
        style={{ marginBottom: 16 }}
      />
      <ProFormSearchableSelect
        name="orderId"
        label="归属订单"
        rules={[{ required: true, message: '请选择调整归属订单' }]}
        options={(detail?.lines || []).map((line) => ({
          label: `${line.orderNo}｜原始提成 ${decimalText(line.commissionAmount)} ${line.baseCurrency}`,
          value: line.orderId,
        }))}
      />
      <ProFormSearchableSelect
        name="direction"
        label="调整方向"
        rules={[{ required: true }]}
        options={[
          { label: '增提（增加应发提成）', value: 'INCREASE' },
          { label: '冲减（减少应发提成）', value: 'DECREASE' },
        ]}
      />
      <ProFormDigit
        name="amount"
        label={`调整金额（${detail?.baseCurrency || ''}）`}
        min={0.00000001}
        fieldProps={{
          precision: 8,
          stringMode: true,
        }}
        rules={[{ required: true, message: '请输入大于 0 的调整金额' }]}
      />
      <ProFormTextArea
        name="reason"
        label="调整原因"
        fieldProps={{ maxLength: 500, showCount: true }}
        rules={[{ required: true, message: '请输入调整原因' }]}
      />
      <ProFormTextArea
        name="note"
        label="补充备注"
        fieldProps={{ maxLength: 500, showCount: true }}
      />
    </ModalForm>
  );
}
