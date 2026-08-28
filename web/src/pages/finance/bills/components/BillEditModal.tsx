import {
  DatePicker,
  Form,
  type FormInstance,
  Input,
  InputNumber,
  Modal,
  Space,
} from 'antd';
import React from 'react';
import type { BillFormValues } from './billConstants';

interface BillEditModalProps {
  open: boolean;
  editing?: API.FinanceBill;
  form: FormInstance<BillFormValues>;
  submitting: boolean;
  onCancel: () => void;
  onOk: () => Promise<void>;
}

export default function BillEditModal({
  open,
  editing,
  form,
  submitting,
  onCancel,
  onOk,
}: BillEditModalProps) {
  return (
    <Modal
      title={`编辑账单 ${editing?.billNo || ''}`}
      open={open}
      width={680}
      destroyOnHidden
      confirmLoading={submitting}
      okText="保存"
      onCancel={onCancel}
      onOk={() => void onOk()}
    >
      <Form form={form} layout="vertical">
        <Space size={16} align="start" wrap style={{ width: '100%' }}>
          <Form.Item
            name="statementTitle"
            label="对账抬头"
            rules={[
              { required: true, whitespace: true, message: '请输入对账抬头' },
              { max: 200, message: '对账抬头不能超过 200 字' },
            ]}
            style={{ minWidth: 260 }}
          >
            <Input maxLength={200} />
          </Form.Item>
          <Form.Item
            name="billDate"
            label="账单日期"
            extra="修改账单日期将自动按新账单日重置 BILL 汇率快照"
            rules={[{ required: true, message: '请选择账单日期' }]}
          >
            <DatePicker allowClear={false} />
          </Form.Item>
          <Form.Item name="paymentTermsDays" label="账期（天，可选）">
            <InputNumber min={0} max={3650} precision={0} />
          </Form.Item>
          <Form.Item name="note" label="备注" style={{ minWidth: 620 }}>
            <Input maxLength={500} />
          </Form.Item>
        </Space>
      </Form>
    </Modal>
  );
}
