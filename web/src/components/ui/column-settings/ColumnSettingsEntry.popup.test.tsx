import {
  cleanup,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { ConfigProvider } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { ColumnSettingsEntry } from './ColumnSettingsEntry';

describe('列设置浮层挂载边界', () => {
  let fullscreenRoot: HTMLElement;
  afterEach(() => {
    cleanup();
    fullscreenRoot?.remove();
    Reflect.deleteProperty(document, 'fullscreenElement');
  });

  it.each([false, true])(
    '表格全屏=%s 时，弹窗、冻结下拉和高级设置均脱离表格局部容器',
    async (fullscreen) => {
      fullscreenRoot = document.createElement('section');
      const tableRoot = document.createElement('div');
      fullscreenRoot.appendChild(tableRoot);
      document.body.appendChild(fullscreenRoot);
      Object.defineProperty(document, 'fullscreenElement', {
        configurable: true,
        value: fullscreen ? fullscreenRoot : null,
      });
      const popupContainer = fullscreen ? fullscreenRoot : document.body;
      render(
        <ConfigProvider
          theme={{ token: { motion: false } }}
          getPopupContainer={() => tableRoot}
        >
          <ColumnSettingsEntry
            fields={[{ key: 'name', title: '名称' }]}
            value={{ order: ['name'], hidden: [] }}
            onChange={vi.fn()}
            onReset={vi.fn()}
            advanced={<div>行配色设置</div>}
          />
        </ConfigProvider>,
        { container: tableRoot },
      );
      fireEvent.click(screen.getByRole('button', { name: '列设置' }));
      const dialog = await screen.findByRole('dialog', { name: '列设置' });
      expect(dialog.closest('.ant-modal-root')?.parentElement).toBe(
        popupContainer,
      );
      expect(tableRoot).not.toContainElement(dialog);

      fireEvent.mouseDown(
        screen.getByRole('combobox', { name: '名称冻结位置' }),
      );
      const listbox = await screen.findByRole('listbox');
      expect(listbox.closest('.ant-select-dropdown')?.parentElement).toBe(
        popupContainer,
      );
      expect(tableRoot).not.toContainElement(listbox);

      fireEvent.click(screen.getByRole('button', { name: '更多设置' }));
      const advanced = await screen.findByRole('dialog', { name: '高级设置' });
      expect(advanced.closest('.ant-modal-root')?.parentElement).toBe(
        popupContainer,
      );
      expect(tableRoot).not.toContainElement(advanced);
      await waitFor(() =>
        expect(screen.queryByRole('dialog', { name: '列设置' })).toBeNull(),
      );
    },
  );
});
