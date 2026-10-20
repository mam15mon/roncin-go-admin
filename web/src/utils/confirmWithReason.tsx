import { type App, Input } from 'antd';
import { resolveTabKey } from '@/components/layout/routeUtils';
import { registerTabCloseGuard } from '@/components/layout/tabCloseGuard';
import { history } from '@/router/history';

export type AppInstance = ReturnType<typeof App.useApp>;

export type ConfirmWithReasonOptions = {
  danger?: boolean;
  /** 所属业务页面卸载时销毁确认框，避免在新组织继续提交旧操作。 */
  signal?: AbortSignal;
  placeholder?: string;
  requiredMessage?: string;
  /** 选填模式：留空可直接确认，回调收到空字符串。 */
  optional?: boolean;
};

/** 弹出操作原因确认框；默认必填，optional 模式下原因选填。 */
export async function confirmWithReason(
  app: Pick<AppInstance, 'modal' | 'message'>,
  title: string,
  onSubmit: (reason: string) => Promise<unknown>,
  options: ConfirmWithReasonOptions = {},
) {
  if (options.signal?.aborted) return false;
  let reason = '';
  let dirty = false;
  const unregister = registerTabCloseGuard(
    resolveTabKey(history.location.pathname),
    {
      isDirty: () => dirty,
    },
  );
  const requiredMessage = options.requiredMessage ?? '请输入操作原因';
  const confirmation = app.modal.confirm({
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
          dirty = true;
          reason = event.target.value.trim();
        }}
      />
    ),
    okText: '确认',
    cancelText: '取消',
    okButtonProps: options.danger ? { danger: true } : undefined,
    onOk: (_close) => {
      if (options.signal?.aborted) return;
      if (!reason && !options.optional) {
        app.message.warning(requiredMessage);
        return;
      }
      return onSubmit(reason).then((result) => {
        if (result === false) throw new Error('操作未成功');
      });
    },
  });
  let abort: (() => void) | undefined;
  const ownerAborted = options.signal
    ? new Promise<false>((resolve) => {
        abort = () => {
          confirmation.destroy();
          resolve(false);
        };
        options.signal?.addEventListener('abort', abort, { once: true });
      })
    : undefined;
  try {
    // App modal 的 destroy 不保证结算 thenable，归属销毁必须显式结束等待并注销守卫。
    return ownerAborted
      ? await Promise.race([confirmation, ownerAborted])
      : await confirmation;
  } finally {
    if (abort) options.signal?.removeEventListener('abort', abort);
    unregister();
  }
}
