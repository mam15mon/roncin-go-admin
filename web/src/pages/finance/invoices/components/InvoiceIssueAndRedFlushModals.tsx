import { DatePicker, Form, type FormInstance, Input, Modal } from 'antd';
import React from 'react';

interface InvoiceIssueModalProps {
  open: boolean;
  submitting: boolean;
  issueForm: FormInstance;
  onCancel: () => void;
  onOk: () => Promise<void>;
}

export function InvoiceIssueModal({
  open,
  submitting,
  issueForm,
  onCancel,
  onOk,
}: InvoiceIssueModalProps) {
  return (
    <Modal
      title="确认开具发票"
      open={open}
      confirmLoading={submitting}
      onCancel={onCancel}
      onOk={() => void onOk()}
    >
      <Form form={issueForm} layout="vertical">
        <Form.Item
          name="taxInvoiceNo"
          label="税务发票号码"
          rules={[{ required: true, message: '请输入税务发票号码' }]}
        >
          <Input maxLength={100} />
        </Form.Item>
        <Form.Item
          name="invoiceDate"
          label="开票日期"
          rules={[{ required: true, message: '请选择开票日期' }]}
        >
          <DatePicker />
        </Form.Item>
      </Form>
    </Modal>
  );
}

interface InvoiceRedFlushModalProps {
  open: boolean;
  submitting: boolean;
  redFlushTarget?: API.FinanceInvoice;
  redFlushForm: FormInstance;
  onCancel: () => void;
  onOk: () => Promise<void>;
}

export function InvoiceRedFlushModal({
  open,
  submitting,
  redFlushTarget,
  redFlushForm,
  onCancel,
  onOk,
}: InvoiceRedFlushModalProps) {
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
      <Form form={redFlushForm} layout="vertical">
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
