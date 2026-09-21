import { Alert, App, Button, Form, type FormInstance, Modal } from 'antd';
import React, { type ReactNode, useRef, useState } from 'react';

/** 可选附加底部动作：点击不触发表单校验或提交，可读取当前表单实例。 */
export interface QuickCreateModalExtraAction<
  TFormValues = Record<string, unknown>,
> {
  text: ReactNode;
  onClick: (form: FormInstance<TFormValues>) => void;
}

export interface QuickCreateModalProps<
  TFormValues = Record<string, unknown>,
  TResult = unknown,
> {
  title: string | ReactNode;
  open: boolean;
  onCancel: () => void;
  /** 返回 undefined 表示没有可用的创建结果，保留输入且不触发成功回调。 */
  onSubmit: (values: TFormValues) => Promise<TResult | undefined>;
  onSuccess?: (result: TResult) => void;
  width?: number;
  okText?: string;
  cancelText?: string;
  alertText?: string | ReactNode;
  alertType?: 'info' | 'warning' | 'success' | 'error';
  initialValues?: Partial<TFormValues>;
  form?: FormInstance<TFormValues>;
  /** 附加底部动作；提供后底部按钮顺序为「附加动作 / 取消 / 保存」。 */
  extraAction?: QuickCreateModalExtraAction<TFormValues>;
  /** 垂直居中弹窗；默认跟随 Modal 顶部对齐。 */
  centered?: boolean;
  children: ReactNode | ((form: FormInstance<TFormValues>) => ReactNode);
}

export function QuickCreateModal<
  TFormValues = Record<string, unknown>,
  TResult = unknown,
>({
  title,
  open,
  onCancel,
  onSubmit,
  onSuccess,
  width = 560,
  okText = '保存并选用',
  cancelText = '取消',
  alertText,
  alertType,
  initialValues,
  form: externalForm,
  extraAction,
  centered,
  children,
}: QuickCreateModalProps<TFormValues, TResult>) {
  const [internalForm] = Form.useForm<TFormValues>();
  const { message } = App.useApp();
  const form = externalForm || internalForm;
  const [saving, setSaving] = useState(false);
  const savingRef = useRef(false);

  const handleSave = async () => {
    if (savingRef.current) return;
    savingRef.current = true;
    setSaving(true);

    let values: TFormValues;
    try {
      values = await form.validateFields();
    } catch {
      savingRef.current = false;
      setSaving(false);
      return;
    }

    try {
      const result = await onSubmit(values);
      if (result !== undefined && result !== null) {
        onSuccess?.(result);
        form.resetFields();
      }
    } catch (error) {
      message.error(error instanceof Error ? error.message : '保存失败');
    } finally {
      savingRef.current = false;
      setSaving(false);
    }
  };

  const handleCancel = () => {
    if (savingRef.current) return;
    form.resetFields();
    onCancel();
  };

  const handleExtraAction = () => {
    if (savingRef.current || !extraAction) return;
    extraAction.onClick(form);
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
      closable={!saving}
      mask={{ closable: !saving }}
      keyboard={!saving}
      cancelButtonProps={{ disabled: saving }}
      centered={centered}
      footer={
        extraAction ? (
          <>
            <Button
              type="primary"
              disabled={saving}
              onClick={handleExtraAction}
            >
              {extraAction.text}
            </Button>
            <Button disabled={saving} onClick={handleCancel}>
              {cancelText}
            </Button>
            <Button
              type="primary"
              loading={saving}
              onClick={() => void handleSave()}
            >
              {okText}
            </Button>
          </>
        ) : undefined
      }
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
