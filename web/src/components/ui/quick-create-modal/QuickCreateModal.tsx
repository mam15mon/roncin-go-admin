import { Alert, Form, type FormInstance, Modal } from 'antd';
import React, { useState, type ReactNode } from 'react';

export interface QuickCreateModalProps<TFormValues = any, TResult = any> {
  title: string | ReactNode;
  open: boolean;
  onCancel: () => void;
  onSubmit: (values: TFormValues) => Promise<TResult | undefined>;
  onSuccess?: (result: TResult) => void;
  width?: number;
  okText?: string;
  cancelText?: string;
  alertText?: string | ReactNode;
  alertType?: 'info' | 'warning' | 'success' | 'error';
  initialValues?: Partial<TFormValues>;
  form?: FormInstance<TFormValues>;
  children: ReactNode | ((form: FormInstance<TFormValues>) => ReactNode);
}

export function QuickCreateModal<TFormValues = any, TResult = any>({
  title,
  open,
  onCancel,
  onSubmit,
  onSuccess,
  width = 560,
  okText = '保存并选用',
  cancelText = '取消',
  alertText,
  alertType = 'info',
  initialValues,
  form: externalForm,
  children,
}: QuickCreateModalProps<TFormValues, TResult>) {
  const [internalForm] = Form.useForm<TFormValues>();
  const form = externalForm || internalForm;
  const [saving, setSaving] = useState(false);

  const handleSave = async () => {
    try {
      const values = await form.validateFields();
      setSaving(true);
      const result = await onSubmit(values);
      if (result !== undefined && result !== null) {
        onSuccess?.(result);
      }
      form.resetFields();
    } catch {
      // Form validation error
    } finally {
      setSaving(false);
    }
  };

  const handleCancel = () => {
    form.resetFields();
    onCancel();
  };

  return (
    <Modal
      title={title}
      open={open}
      confirmLoading={saving}
      okText={okText}
      cancelText={cancelText}
      onOk={() => void handleSave()}
      onCancel={handleCancel}
      destroyOnHidden
      width={width}
    >
      <Form
        form={form}
        layout="vertical"
        preserve={false}
        initialValues={initialValues}
      >
        {alertText && (
          <Alert
            type={alertType}
            showIcon
            title={alertText}
            style={{ marginBottom: 16 }}
          />
        )}
        {typeof children === 'function' ? children(form) : children}
      </Form>
    </Modal>
  );
}

export default QuickCreateModal;
