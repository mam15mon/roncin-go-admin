import { DatePicker, Form, type FormInstance, Input, Modal } from 'antd';
import React from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';

interface InvoiceRedFlushModalProps {
  open: boolean;
  submitting: boolean;
  redFlushTarget?: API.FinanceInvoice;
  redFlushForm: FormInstance;
  onCancel: () => void;
  onOk: () => Promise<unknown>;
}

export function InvoiceRedFlushModal({
  open,
  submitting,
  redFlushTarget,
  redFlushForm,
  onCancel,
  onOk,
}: InvoiceRedFlushModalProps) {
  const { markDirty } = useUnsavedChangesGuard({
    enabled: open,
    identity: redFlushTarget?.id,
  });
  return (
    <Modal
      title={`红冲发票 ${redFlushTarget?.taxInvoiceNo || ''}`}
      open={open}
      confirmLoading={submitting}
      okButtonProps={{ danger: true }}
      okText="确认红冲"
      onCancel={onCancel}
      onOk={() => void onOk()}
    >
      <Form form={redFlushForm} layout="vertical" onValuesChange={markDirty}>
        <Form.Item
          name="redInvoiceNo"
          label="红字发票号码"
          rules={[{ required: true, message: '请输入红字发票号码' }]}
        >
          <Input maxLength={100} />
        </Form.Item>
        <Form.Item
          name="redInvoiceDate"
          label="红冲日期"
          rules={[{ required: true, message: '请选择红冲日期' }]}
        >
          <DatePicker />
        </Form.Item>
        <Form.Item
          name="reason"
          label="红冲原因"
          rules={[{ required: true, message: '请输入红冲原因' }]}
        >
          <Input.TextArea maxLength={500} rows={3} />
        </Form.Item>
      </Form>
    </Modal>
  );
}
