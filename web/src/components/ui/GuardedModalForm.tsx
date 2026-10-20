import { ModalForm, type ModalFormProps } from '@ant-design/pro-components';
import { useState } from 'react';
import { useUnsavedChangesGuard } from '@/components/layout/useUnsavedChangesGuard';

/** 保留官方表单与提交生命周期，只登记真实用户修改的离开保护。 */
export function GuardedModalForm<
  T = Record<string, unknown>,
  U = Record<string, unknown>,
>({
  guardIdentity,
  ...props
}: ModalFormProps<T, U> & { guardIdentity?: unknown }) {
  const [uncontrolledOpen, setUncontrolledOpen] = useState(false);
  const open = props.open ?? uncontrolledOpen;
  const { markDirty, resetDirty } = useUnsavedChangesGuard({
    enabled: open,
    identity: guardIdentity,
  });

  return (
    <ModalForm<T, U>
      {...props}
      onOpenChange={(nextOpen) => {
        setUncontrolledOpen(nextOpen);
        if (!nextOpen) resetDirty();
        props.onOpenChange?.(nextOpen);
      }}
      onValuesChange={(changedValues, allValues) => {
        markDirty();
        props.onValuesChange?.(changedValues, allValues);
      }}
      onFinish={
        props.onFinish
          ? async (values) => {
              const result = await props.onFinish?.(values);
              if (result === true) resetDirty();
              return result;
            }
          : undefined
      }
    />
  );
}
