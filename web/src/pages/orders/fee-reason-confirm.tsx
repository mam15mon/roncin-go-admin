import { type App, Input } from 'antd';

type AppInstance = ReturnType<typeof App.useApp>;

/** 弹出必填操作原因的确认框，确认后执行回调。 */
export function confirmWithReason(
  app: Pick<AppInstance, 'modal' | 'message'>,
  title: string,
  onSubmit: (reason: string) => Promise<void>,
) {
  let reason = '';
  app.modal.confirm({
    title,
    content: (
      <Input.TextArea
        autoFocus
        maxLength={500}
        showCount
        placeholder="请输入操作原因（必填）"
        onChange={(event) => {
          reason = event.target.value.trim();
        }}
      />
    ),
    okText: '确认',
    cancelText: '取消',
    onOk: async () => {
      if (!reason) {
        app.message.warning('请输入操作原因');
        throw new Error('操作原因不能为空');
      }
      await onSubmit(reason);
    },
  });
}
