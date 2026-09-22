import { type App, Input } from 'antd';

export type AppInstance = ReturnType<typeof App.useApp>;

export type ConfirmWithReasonOptions = {
  danger?: boolean;
  placeholder?: string;
  requiredMessage?: string;
  /** 选填模式：留空可直接确认，回调收到空字符串。 */
  optional?: boolean;
};

/** 弹出操作原因确认框；默认必填，optional 模式下原因选填。 */
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
        placeholder={
          options.placeholder ??
          (options.optional
            ? '请输入操作原因（选填）'
            : '请输入操作原因（必填）')
        }
        onChange={(event) => {
          reason = event.target.value.trim();
        }}
      />
    ),
    okText: '确认',
    cancelText: '取消',
    okButtonProps: options.danger ? { danger: true } : undefined,
    onOk: (_close) => {
      if (!reason && !options.optional) {
        app.message.warning(requiredMessage);
        return;
      }
      return onSubmit(reason);
    },
  });
}
