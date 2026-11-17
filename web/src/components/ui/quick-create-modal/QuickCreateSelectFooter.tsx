import { PlusOutlined } from '@ant-design/icons';
import { Button, theme } from 'antd';
import type { ReactNode } from 'react';

/** 保留候选菜单，在底部提供不会抢走输入焦点的新增入口。 */
export function QuickCreateSelectFooter({
  children,
  label,
  onCreate,
  disabled = false,
}: {
  children: ReactNode;
  label: string;
  onCreate: () => void;
  disabled?: boolean;
}) {
  const { token } = theme.useToken();
  return (
    <>
      {children}
      <div
        style={{
          borderTop: `1px solid ${token.colorBorderSecondary}`,
          padding: 4,
        }}
        onMouseDown={(event) => {
          event.preventDefault();
          event.stopPropagation();
        }}
      >
        <Button
          type="text"
          block
          icon={<PlusOutlined aria-hidden />}
          disabled={disabled}
          style={{
            color: disabled ? undefined : token.colorPrimary,
            textAlign: 'left',
          }}
          onClick={onCreate}
        >
          {label}
        </Button>
      </div>
    </>
  );
}
