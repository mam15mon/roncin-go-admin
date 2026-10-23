import {
  act,
  cleanup,
  fireEvent,
  render,
  screen,
} from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { saveColumnSettingsPreference } from './preference';
import { useColumnSettings } from './useColumnSettings';

const scope = { userId: 'u1', organizationId: 'o1' };
const columns = [
  { key: 'index', title: '序号', valueType: 'index', fixed: 'left' as const },
  { key: 'name', title: '姓名', fixed: 'left' as const },
  { key: 'email', title: '邮箱', fixed: 'left' as const },
  { key: 'phone', title: '电话' },
  { key: 'action', title: '操作', fixed: 'right' as const },
];
function Inner() {
  const settings = useColumnSettings({
    tableKey: 'test:freeze',
    columns,
    scope,
  });
  return (
    <>
      {settings.entry}
      <output data-testid="columns">{JSON.stringify(settings.columns)}</output>
    </>
  );
}
function read() {
  return JSON.parse(
    screen.getByTestId('columns').textContent ?? '[]',
  ) as typeof columns;
}
async function open() {
  fireEvent.click(screen.getByRole('button', { name: '列设置' }));
  await screen.findByRole('combobox', { name: '姓名冻结位置' });
}
async function setFreeze(title: string, label: string) {
  await act(async () => {
    fireEvent.mouseDown(
      screen.getByRole('combobox', { name: `${title}冻结位置` }),
    );
  });
  fireEvent.click(
    await screen.findByText(label, {
      selector: '.ant-select-item-option-content',
    }),
  );
}
describe('逐字段冻结全链路', () => {
  afterEach(() => {
    cleanup();
    window.localStorage.clear();
    vi.restoreAllMocks();
  });
  it('业务默认保留两个左冻结，结构操作可独立解除', async () => {
    render(
      <App>
        <Inner />
      </App>,
    );
    await open();
    expect(read().map((column) => column.fixed)).toEqual([
      'left',
      'left',
      'left',
      false,
      'right',
    ]);
    expect(screen.queryByRole('checkbox', { name: '操作' })).toBeNull();
    expect(screen.queryByRole('checkbox', { name: '序号' })).toBeNull();
    await setFreeze('操作', '不冻结');
    await setFreeze('电话', '左冻结');
    expect(read().map((column) => column.fixed)).toEqual([
      'left',
      'left',
      'left',
      'left',
      false,
    ]);
    expect(
      JSON.parse(
        window.localStorage.getItem(
          'roncin:column-settings:v1:test:freeze:u1:o1',
        ) ?? '{}',
      ).freeze,
    ).toEqual({ left: ['name', 'email', 'phone'], right: [] });
  });
  it('隐藏和重挂载保留字段冻结，恢复默认恢复完整区域', async () => {
    saveColumnSettingsPreference('test:freeze', scope, {
      order: ['phone', 'email', 'name'],
      hidden: ['name'],
      freeze: { left: ['phone', 'email'], right: [] },
    });
    const first = render(
      <App>
        <Inner />
      </App>,
    );
    first.unmount();
    render(
      <App>
        <Inner />
      </App>,
    );
    await open();
    expect(read().map((column) => column.key)).toEqual([
      'index',
      'phone',
      'email',
      'action',
    ]);
    fireEvent.click(screen.getByRole('checkbox', { name: '姓名' }));
    expect(read().find((column) => column.key === 'name')?.fixed).toBe(false);
    fireEvent.click(screen.getByRole('button', { name: '恢复默认' }));
    expect(read().map((column) => column.key)).toEqual([
      'index',
      'name',
      'email',
      'phone',
      'action',
    ]);
    expect(read().map((column) => column.fixed)).toEqual([
      'left',
      'left',
      'left',
      false,
      'right',
    ]);
  });
  it('存储失败仍即时冻结且面板保持打开', async () => {
    vi.spyOn(window.localStorage, 'setItem').mockImplementation(() => {
      throw new Error('quota');
    });
    render(
      <App>
        <Inner />
      </App>,
    );
    await open();
    await setFreeze('电话', '右冻结');
    expect(read().find((column) => column.key === 'phone')?.fixed).toBe(
      'right',
    );
    expect(
      await screen.findByText(
        '列设置保存到本地浏览器失败，本次设置仅当前页面生效',
      ),
    ).toBeInTheDocument();
    expect(screen.getByRole('dialog', { name: '列设置' })).toBeInTheDocument();
  });
});
