import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { saveColumnSettingsPreference } from './preference';
import { useColumnSettings } from './useColumnSettings';

const scope = { userId: 'u1', organizationId: 'o1' };
const columns = [
  { key: 'index', title: '序号', valueType: 'index', fixed: 'left' as const },
  { key: 'name', title: '姓名', fixed: 'left' as const, width: 180 },
  { key: 'email', title: '邮箱', fixed: 'left' as const },
  { key: 'phone', title: '电话' },
  { key: 'action', title: '操作', fixed: 'right' as const },
];

function Inner({ tableKey = 'test:freeze' }: { tableKey?: string }) {
  const settings = useColumnSettings<(typeof columns)[number]>({
    tableKey,
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

async function open() {
  fireEvent.click(screen.getByRole('button', { name: '列设置' }));
  await screen.findByRole('switch', { name: '冻结首列' });
}

function readColumns() {
  return JSON.parse(
    screen.getByTestId('columns').textContent ?? '[]',
  ) as typeof columns;
}

describe('列设置首尾冻结交互', () => {
  afterEach(() => {
    cleanup();
    window.localStorage.clear();
    vi.restoreAllMocks();
  });

  it('两个开关独立生效，操作与序号不进入排序或显隐', async () => {
    render(
      <App>
        <Inner />
      </App>,
    );
    await open();
    expect(screen.getByRole('switch', { name: '冻结首列' })).toBeChecked();
    expect(screen.getByRole('switch', { name: '冻结尾列' })).toBeChecked();
    expect(
      screen.queryByRole('checkbox', { name: '操作' }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('checkbox', { name: '序号' }),
    ).not.toBeInTheDocument();
    expect(readColumns().map((column) => column.fixed)).toEqual([
      'left',
      'left',
      false,
      false,
      'right',
    ]);
    fireEvent.click(screen.getByRole('switch', { name: '冻结首列' }));
    expect(readColumns().map((column) => column.fixed)).toEqual([
      'left',
      false,
      false,
      false,
      'right',
    ]);
    fireEvent.click(screen.getByRole('switch', { name: '冻结尾列' }));
    expect(readColumns().map((column) => column.fixed)).toEqual([
      'left',
      false,
      false,
      false,
      false,
    ]);
    expect(screen.getByPlaceholderText('搜索列名')).toBeInTheDocument();
    expect(
      JSON.parse(
        window.localStorage.getItem(
          'roncin:column-settings:v1:test:freeze:u1:o1',
        ) ?? '{}',
      ).freeze,
    ).toEqual({ first: false, last: false });
  });

  it('隐藏和排序后首列跟随新顺序，设置搜索不改变目标', async () => {
    render(
      <App>
        <Inner />
      </App>,
    );
    await open();
    fireEvent.click(screen.getByRole('button', { name: '上移邮箱' }));
    expect(readColumns()[1]).toMatchObject({ key: 'email', fixed: 'left' });
    expect(screen.getByText('当前：邮箱')).toBeInTheDocument();
    fireEvent.click(screen.getByRole('checkbox', { name: '邮箱' }));
    expect(readColumns()[1]).toMatchObject({ key: 'name', fixed: 'left' });
    expect(screen.getByText('当前：姓名')).toBeInTheDocument();
    fireEvent.change(screen.getByPlaceholderText('搜索列名'), {
      target: { value: '电话' },
    });
    expect(screen.getByText('当前：姓名')).toBeInTheDocument();
    expect(screen.getByText('当前：操作')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: '上移电话' })).toBeDisabled();
  });

  it('重挂载保留关闭冻结与显隐，恢复默认同时重置冻结和顺序', async () => {
    saveColumnSettingsPreference('test:freeze', scope, {
      order: ['phone', 'email', 'name'],
      hidden: ['name'],
      freeze: { first: false, last: false },
    });
    const first = render(
      <App>
        <Inner />
      </App>,
    );
    expect(readColumns().map((column) => column.key)).toEqual([
      'index',
      'phone',
      'email',
      'action',
    ]);
    first.unmount();
    render(
      <App>
        <Inner />
      </App>,
    );
    await open();
    expect(screen.getByRole('switch', { name: '冻结首列' })).not.toBeChecked();
    expect(screen.getByRole('switch', { name: '冻结尾列' })).not.toBeChecked();
    fireEvent.click(screen.getByRole('button', { name: '恢复默认' }));
    expect(readColumns().map((column) => column.key)).toEqual([
      'index',
      'name',
      'email',
      'phone',
      'action',
    ]);
    expect(readColumns().map((column) => column.fixed)).toEqual([
      'left',
      'left',
      false,
      false,
      'right',
    ]);
    expect(screen.getByRole('switch', { name: '冻结首列' })).toBeChecked();
    expect(screen.getByRole('switch', { name: '冻结尾列' })).toBeChecked();
  });

  it('冻结保存失败仍生效且明确提示，浮层保持打开', async () => {
    vi.spyOn(window.localStorage, 'setItem').mockImplementation(() => {
      throw new Error('quota');
    });
    render(
      <App>
        <Inner />
      </App>,
    );
    await open();
    fireEvent.click(screen.getByRole('switch', { name: '冻结尾列' }));
    expect(readColumns().at(-1)?.fixed).toBe(false);
    expect(
      await screen.findByText(
        '列设置保存到本地浏览器失败，本次设置仅当前页面生效',
      ),
    ).toBeInTheDocument();
    expect(
      screen.getByRole('switch', { name: '冻结尾列' }),
    ).toBeInTheDocument();
  });
});
