import { Alert, Form, type FormInstance, Input, Modal } from 'antd';
import React from 'react';
import { FinanceInvoiceStatus } from '@/enums.generated';
import {
  invoiceDraftCancelDescription,
  invoiceDraftCancelTitle,
  invoiceVoidTaxConditionNotice,
  invoiceVoidTitle,
} from './invoiceConstants';

type CancelValues = { reason: string };

interface InvoiceCancelModalProps {
  open: boolean;
  submitting: boolean;
  cancelForm: FormInstance<CancelValues>;
  cancelTarget?: API.FinanceInvoice;
  onCancel: () => void;
  onOk: () => Promise<void>;
}

/** 草稿显示「取消」，已开票显示带线下税务条件危险提示的「作废」确认。 */
export default function InvoiceCancelModal({
  open,
  submitting,
  cancelForm,
  cancelTarget,
  onCancel,
  onOk,
}: InvoiceCancelModalProps) {
  const issued =
    cancelTarget?.status === FinanceInvoiceStatus.FINANCE_INVOICE_STATUS_ISSUED;
  const actionVerb = issued ? '作废' : '取消';
  return (
    <Modal
      title={
        issued
          ? invoiceVoidTitle(
              cancelTarget?.direction,
              cancelTarget?.organizationName,
            )
          : invoiceDraftCancelTitle(
              cancelTarget?.direction,
              cancelTarget?.organizationName,
            )
      }
      open={open}
      confirmLoading={submitting}
      okButtonProps={{ danger: true }}
      okText={`确认${actionVerb}`}
      cancelText="再想想"
      onCancel={onCancel}
      onOk={() => void onOk()}
    >
      {issued ? (
        <Alert
          type="warning"
          showIcon
          title="线下税务条件确认"
          description={invoiceVoidTaxConditionNotice}
          style={{ marginBottom: 16 }}
        />
      ) : (
        <p style={{ color: '#595959', marginBottom: 16 }}>
          {invoiceDraftCancelDescription}
        </p>
      )}
      <Form form={cancelForm} layout="vertical">
        <Form.Item
          name="reason"
          label={`${actionVerb}原因`}
          rules={[{ required: true, message: `请输入${actionVerb}原因` }]}
        >
          <Input.TextArea
            autoFocus
            maxLength={500}
            showCount
            rows={3}
            placeholder={`请输入${actionVerb}原因（必填）`}
          />
        </Form.Item>
      </Form>
    </Modal>
  );
}
