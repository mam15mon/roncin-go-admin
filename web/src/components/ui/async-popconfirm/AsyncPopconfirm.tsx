import { Popconfirm, type PopconfirmProps } from 'antd';
import { useRef, useState } from 'react';

export type AsyncPopconfirmProps = Omit<PopconfirmProps, 'onConfirm'> & {
  onConfirm: (
    event?: React.MouseEvent<HTMLElement>,
  ) => unknown | Promise<unknown>;
};

/** 写确认保留官方 Promise loading；false 或拒绝时保留弹层，不重复提示请求错误。 */
export function AsyncPopconfirm({
  onConfirm,
  open: controlledOpen,
  defaultOpen = false,
  onOpenChange,
  ...props
}: AsyncPopconfirmProps) {
  const [internalOpen, setInternalOpen] = useState(defaultOpen);
  const preserveOpen = useRef(false);
  return (
    <Popconfirm
      {...props}
      open={controlledOpen ?? internalOpen}
      onOpenChange={(open) => {
        // ActionButton 在 Promise resolve 后必定请求关闭；本次写失败时拒绝该次关闭。
        if (!open && preserveOpen.current) {
          preserveOpen.current = false;
          return;
        }
        setInternalOpen(open);
        onOpenChange?.(open);
      }}
      onConfirm={async (event) => {
        preserveOpen.current = false;
        try {
          preserveOpen.current = (await onConfirm(event)) === false;
        } catch {
          // 请求与业务回调负责错误反馈；这里仅维持可重试的确认状态。
          preserveOpen.current = true;
        }
      }}
    />
  );
}
