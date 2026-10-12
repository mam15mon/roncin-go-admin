import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { Button } from 'antd';
import React from 'react';
import { afterEach, expect, it, vi } from 'vitest';
import HeaderDropdown from './index';

afterEach(cleanup);
it('opt-in原生按钮点击及ArrowDown打开，Escape返回触发焦点并保留menu原生选择', async () => {
  const onClick = vi.fn();
  render(
    <HeaderDropdown
      keyboardAccessible
      menu={{
        items: [
          { key: 'disabled', label: '无权限项', disabled: true },
          { key: 'allowed', label: '可用入口' },
        ],
        onClick,
      }}
    >
      <Button>键盘菜单</Button>
    </HeaderDropdown>,
  );
  const trigger = screen.getByRole('button', { name: '键盘菜单' });
  expect(trigger).toHaveAttribute('aria-haspopup', 'menu');
  expect(trigger).toHaveAttribute('aria-expanded', 'false');
  act(() => trigger.focus());
  fireEvent.keyDown(trigger, {
    key: 'ArrowDown',
    code: 'ArrowDown',
    keyCode: 40,
  });
  await screen.findByRole('menu');
  expect(trigger).toHaveAttribute('aria-expanded', 'true');
  await waitFor(() =>
    expect(screen.getByRole('menu')).toContainElement(
      document.activeElement as HTMLElement,
    ),
  );
  fireEvent.keyDown(screen.getByRole('menu'), {
    key: 'Escape',
    code: 'Escape',
    keyCode: 27,
  });
  await waitFor(() =>
    expect(trigger).toHaveAttribute('aria-expanded', 'false'),
  );
  expect(trigger).toHaveFocus();
  // happy-dom不生成原生键盘click，显式发送浏览器按钮激活产生的click。
  fireEvent.keyDown(trigger, { key: 'Enter', code: 'Enter', keyCode: 13 });
  fireEvent.click(trigger);
  await waitFor(() => expect(trigger).toHaveAttribute('aria-expanded', 'true'));
  const enabled = await screen.findByRole('menuitem', { name: '可用入口' });
  expect(screen.getByRole('menuitem', { name: '无权限项' })).toHaveAttribute(
    'aria-disabled',
    'true',
  );
  fireEvent.click(enabled);
  expect(onClick).toHaveBeenCalledOnce();
  expect(onClick.mock.calls[0][0].key).toBe('allowed');
  await waitFor(() =>
    expect(trigger).toHaveAttribute('aria-expanded', 'false'),
  );
  fireEvent.keyDown(trigger, { key: ' ', code: 'Space', keyCode: 32 });
  fireEvent.keyUp(trigger, { key: ' ', code: 'Space', keyCode: 32 });
  fireEvent.click(trigger);
  await waitFor(() => expect(trigger).toHaveAttribute('aria-expanded', 'true'));
  fireEvent.keyDown(trigger, { key: 'Escape', keyCode: 27 });
  expect(trigger).toHaveFocus();
});
it('未启用时维持hover默认行为，启用仍支持hover', async () => {
  const view = render(
    <HeaderDropdown menu={{ items: [{ key: 'item', label: '默认入口' }] }}>
      <Button>默认菜单</Button>
    </HeaderDropdown>,
  );
  const trigger = screen.getByRole('button', { name: '默认菜单' });
  expect(trigger).not.toHaveAttribute('aria-expanded');
  fireEvent.click(trigger);
  expect(screen.queryByRole('menu')).not.toBeInTheDocument();
  fireEvent.mouseEnter(trigger);
  await screen.findByRole('menuitem', { name: '默认入口' });
  view.unmount();
  render(
    <HeaderDropdown
      keyboardAccessible
      menu={{ items: [{ key: 'item', label: '悬停入口' }] }}
    >
      <Button>增强菜单</Button>
    </HeaderDropdown>,
  );
  fireEvent.mouseEnter(screen.getByRole('button', { name: '增强菜单' }));
  await screen.findByRole('menuitem', { name: '悬停入口' });
});
