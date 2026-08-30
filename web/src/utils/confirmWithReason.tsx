import { type App, Input } from 'antd';

export type AppInstance = ReturnType<typeof App.useApp>;

export type ConfirmWithReasonOptions = {
  danger?: boolean;
  placeholder?: string;
  requiredMessage?: string;
};

/** 弹出必填操作原因的确认框，确认后执行回调。 */
export function confirmWithReason(
  app: Pick<AppInstance, 'modal' | 'message'>,
  title: string,
  onSubmit: (reason: string) => Promise<void>,
  options: ConfirmWithReasonOptions = {},
) {
  let reason = '';
  const requiredMessage = options.requiredMessage ?? '请输入操作原因';
  app.modal.confirm({
    title,
    content: (
      <Input.TextArea
        autoFocus
        maxLength={500}
        showCount
        placeholder={options.placeholder ?? '请输入操作原因（必填）'}
        onChange={(event) => {
          reason = event.target.value.trim();
        }}
      />
    ),
    okText: '确认',
    cancelText: '取消',
    okButtonProps: options.danger ? { danger: true } : undefined,
    onOk: async () => {
      if (!reason) {
        app.message.warning(requiredMessage);
        throw new Error(`${requiredMessage.replace(/^请输入/, '')}不能为空`);
      }
      await onSubmit(reason);
    },
  });
}
