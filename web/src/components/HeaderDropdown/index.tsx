import { Dropdown } from 'antd';
import type { DropDownProps } from 'antd/es/dropdown';
import { createStyles } from 'antd-style';
import { clsx } from 'clsx';
import React, { useEffect, useRef, useState } from 'react';

const useStyles = createStyles(({ token }) => {
  return {
    dropdown: {
      [`@media screen and (max-width: ${token.screenXS}px)`]: {
        width: '100%',
      },
      '.ant-dropdown-menu-item .anticon, .ant-dropdown-menu-submenu-title .anticon':
        {
          display: 'inline-flex',
          alignItems: 'center',
        },
      '.ant-dropdown-menu-submenu-title .anticon': {
        color: token.colorTextSecondary,
      },
    },
  };
});

export type HeaderDropdownProps = {
  /** 目标顶栏按钮显式启用键盘打开与焦点返回，其他消费者保持原行为。 */
  keyboardAccessible?: boolean;
  overlayClassName?: string;
  placement?:
    | 'bottomLeft'
    | 'bottomRight'
    | 'topLeft'
    | 'topCenter'
    | 'topRight'
    | 'bottomCenter';
} & Omit<DropDownProps, 'overlay'>;

function KeyboardDropdown({
  children,
  onOpenChange,
  ...props
}: Omit<HeaderDropdownProps, 'keyboardAccessible'>) {
  const [open, setOpen] = useState(false);
  const [keyboardOpening, setKeyboardOpening] = useState(false);
  const triggerRef = useRef<HTMLElement>(null);
  useEffect(() => {
    if (!open) return;
    const handleEscape = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return;
      event.preventDefault();
      setOpen(false);
      setKeyboardOpening(false);
      triggerRef.current?.focus();
    };
    document.addEventListener('keydown', handleEscape, true);
    return () => document.removeEventListener('keydown', handleEscape, true);
  }, [open]);
  const child = React.Children.only(children) as React.ReactElement<
    React.HTMLAttributes<HTMLElement> & { disabled?: boolean }
  >;
  return (
    <Dropdown
      {...props}
      ref={triggerRef}
      trigger={['hover', 'click']}
      open={open}
      autoFocus={keyboardOpening}
      onOpenChange={(nextOpen, info) => {
        setOpen(nextOpen);
        if (!nextOpen) setKeyboardOpening(false);
        onOpenChange?.(nextOpen, info);
      }}
    >
      {React.cloneElement(child, {
        'aria-haspopup': 'menu',
        'aria-expanded': open,
        onKeyDown: (event: React.KeyboardEvent<HTMLElement>) => {
          child.props.onKeyDown?.(event);
          if (event.defaultPrevented || child.props.disabled) return;
          if (event.key === 'ArrowDown') {
            event.preventDefault();
            setKeyboardOpening(true);
            setOpen(true);
          } else if (event.key === 'Enter' || event.key === ' ') {
            setKeyboardOpening(true);
          }
        },
      })}
    </Dropdown>
  );
}

const HeaderDropdown: React.FC<HeaderDropdownProps> = ({
  keyboardAccessible = false,
  overlayClassName: cls,
  dropdownRender,
  popupRender,
  ...restProps
}) => {
  const { styles } = useStyles();
  const DropdownComponent = keyboardAccessible ? KeyboardDropdown : Dropdown;
  return (
    <DropdownComponent
      classNames={{
        root: clsx(styles.dropdown, cls),
      }}
      popupRender={popupRender ?? dropdownRender}
      {...restProps}
    />
  );
};

export default HeaderDropdown;
