import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { App, Table } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import {
  clearColumnSettingsPreference,
  saveColumnSettingsPreference,
} from './preference';
import { useColumnSettings } from './useColumnSettings';

const BASE_COLUMNS = [
  { title: '业务单号', dataIndex: 'code', key: 'code' },
  { title: '客户名称', dataIndex: 'customer', key: 'customer' },
  { title: '备注', dataIndex: 'note', key: 'note' },
  { title: '操作', key: 'option', fixed: 'right' as const },
];

const SCOPE = { userId: 'u1', organizationId: 'o1' };

function Inner(props: {
  tableKey: string;
  structuralKeys?: string[];
  lockVisibleKeys?: string[];
  defaultHiddenKeys?: string[];
  scope?: typeof SCOPE | undefined;
  advanced?: React.ReactNode;
}) {
  const settings = useColumnSettings({
    tableKey: props.tableKey,
    columns: BASE_COLUMNS,
    structuralKeys: props.structuralKeys,
    lockVisibleKeys: props.lockVisibleKeys,
    defaultHiddenKeys: props.defaultHiddenKeys,
    scope: props.scope,
    advanced: props.advanced,
  });
  return (
    <>
      {settings.entry}
      <Table
        size="small"
        pagination={false}
        columns={settings.columns}
        dataSource={[{ code: 'A1', customer: '客户甲', note: '备注' }]}
        rowKey="code"
      />
    </>
  );
}

function Harness(props: Parameters<typeof Inner>[0]) {
  return (
    <App>
      <Inner {...props} />
    </App>
  );
}

function renderHarness(props: Parameters<typeof Harness>[0]) {
  return render(<Harness {...props} />);
}

async function openSettings() {
  fireEvent.click(screen.getByRole('button', { name: '列设置' }));
  await screen.findByPlaceholderText('搜索列名');
}

function readHeaders() {
  return screen.getAllByRole('columnheader').map((node) => node.textContent);
}

describe('useColumnSettings', () => {
  afterEach(() => {
    cleanup();
    window.localStorage.clear();
  });

  it('勾选隐藏即时生效并自动持久化', async () => {
    const storageSpy = vi.spyOn(window.localStorage, 'setItem');
    renderHarness({ tableKey: 'test:demo', scope: SCOPE });
    await openSettings();
    fireEvent.click(screen.getByRole('checkbox', { name: '客户名称' }));
    // 无需保存确认：表格立即少一列
    expect(readHeaders()).toEqual(['业务单号', '备注', '操作']);
    expect(storageSpy).toHaveBeenCalled();
    const stored = window.localStorage.getItem(
      'roncin:column-settings:v1:test:demo:u1:o1',
    );
    expect(JSON.parse(stored as string)).toEqual({
      order: ['code', 'customer', 'note', 'option'],
      hidden: ['customer'],
    });
    storageSpy.mockRestore();
  });

  it('上下移排序即时生效并持久化完整顺序', async () => {
    renderHarness({ tableKey: 'test:move', scope: SCOPE });
    await openSettings();
    fireEvent.click(screen.getByRole('button', { name: '下移业务单号' }));
    expect(readHeaders()).toEqual(['客户名称', '业务单号', '备注', '操作']);
    expect(
      JSON.parse(
        window.localStorage.getItem(
          'roncin:column-settings:v1:test:move:u1:o1',
        ) as string,
      ),
    ).toEqual({
      order: ['customer', 'code', 'note', 'option'],
      hidden: [],
    });
  });

  it('重挂载后恢复已保存的偏好', () => {
    saveColumnSettingsPreference('test:restore', SCOPE, {
      order: ['note', 'code', 'customer', 'option'],
      hidden: ['note'],
    });
    const first = renderHarness({ tableKey: 'test:restore', scope: SCOPE });
    first.unmount();
    renderHarness({ tableKey: 'test:restore', scope: SCOPE });
    expect(readHeaders()).toEqual(['业务单号', '客户名称', '操作']);
  });

  it('结构列不进入设置且保持在原锚点位置', async () => {
    renderHarness({
      tableKey: 'test:structural',
      structuralKeys: ['option'],
    });
    await openSettings();
    expect(
      screen.getByRole('checkbox', { name: '业务单号' }),
    ).toBeInTheDocument();
    expect(
      screen.queryByRole('checkbox', { name: '操作' }),
    ).not.toBeInTheDocument();
    fireEvent.click(screen.getByRole('checkbox', { name: '备注' }));
    const headers = readHeaders();
    expect(headers[headers.length - 1]).toBe('操作');
  });

  it('损坏的存储偏好回退为默认列，未知 key 被忽略', () => {
    window.localStorage.setItem(
      'roncin:column-settings:v1:test:broken:u1:o1',
      '{"order":"bad","hidden":null}',
    );
    renderHarness({ tableKey: 'test:broken', scope: SCOPE });
    expect(readHeaders()).toEqual(['业务单号', '客户名称', '备注', '操作']);
  });

  it('存储写入失败时提示一次且页面内保持生效、浮层不关闭', async () => {
    const setItemSpy = vi
      .spyOn(window.localStorage, 'setItem')
      .mockImplementation(() => {
        throw new Error('quota');
      });
    try {
      renderHarness({ tableKey: 'test:failure', scope: SCOPE });
      await openSettings();
      fireEvent.click(screen.getByRole('checkbox', { name: '备注' }));
      expect(
        await screen.findByText(
          '列设置保存到本地浏览器失败，本次设置仅当前页面生效',
        ),
      ).toBeInTheDocument();
      // 表格已应用本次设置
      expect(readHeaders()).toEqual(['业务单号', '客户名称', '操作']);
      // 浮层保持打开，可继续调整
      expect(
        screen.getByRole('checkbox', { name: '客户名称' }),
      ).toBeInTheDocument();
      // 连续失败只提示一次
      fireEvent.click(screen.getByRole('checkbox', { name: '客户名称' }));
      expect(
        screen.getAllByText(
          '列设置保存到本地浏览器失败，本次设置仅当前页面生效',
        ),
      ).toHaveLength(1);
    } finally {
      setItemSpy.mockRestore();
    }
  });

  it('恢复默认即时生效并持久化默认配置', async () => {
    saveColumnSettingsPreference('test:clear', SCOPE, {
      order: ['note', 'code', 'customer', 'option'],
      hidden: [],
    });
    renderHarness({ tableKey: 'test:clear', scope: SCOPE });
    await openSettings();
    fireEvent.click(screen.getByRole('button', { name: '恢复默认' }));
    expect(readHeaders()).toEqual(['业务单号', '客户名称', '备注', '操作']);
    expect(
      JSON.parse(
        window.localStorage.getItem(
          'roncin:column-settings:v1:test:clear:u1:o1',
        ) as string,
      ),
    ).toEqual({
      order: ['code', 'customer', 'note', 'option'],
      hidden: [],
    });
    clearColumnSettingsPreference('test:clear', SCOPE);
    expect(
      window.localStorage.getItem('roncin:column-settings:v1:test:clear:u1:o1'),
    ).toBeNull();
  });

  it('提供高级内容时浮层出现「更多设置」并打开二级弹窗', async () => {
    renderHarness({
      tableKey: 'test:advanced',
      advanced: <div>行配色设置内容</div>,
    });
    await openSettings();
    fireEvent.click(screen.getByRole('button', { name: '更多设置' }));
    expect(await screen.findByText('行配色设置内容')).toBeInTheDocument();
  });
});
