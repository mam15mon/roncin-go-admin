import { cleanup, fireEvent, render, screen } from '@testing-library/react';
import { App, Table } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { useColumnSettings } from './useColumnSettings';
import {
  clearColumnSettingsPreference,
  saveColumnSettingsPreference,
} from './preference';

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
  persist?: false;
  scope?: typeof SCOPE | undefined;
}) {
  const settings = useColumnSettings({
    tableKey: props.tableKey,
    columns: BASE_COLUMNS,
    structuralKeys: props.structuralKeys,
    lockVisibleKeys: props.lockVisibleKeys,
    defaultHiddenKeys: props.defaultHiddenKeys,
    scope: props.scope,
    persist: props.persist,
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
      {settings.modal}
    </>
  );
}

function Harness(props: {
  tableKey: string;
  structuralKeys?: string[];
  lockVisibleKeys?: string[];
  defaultHiddenKeys?: string[];
  persist?: false;
  scope?: typeof SCOPE | undefined;
}) {
  return (
    <App>
      <Inner {...props} />
    </App>
  );
}

function renderHarness(props: Parameters<typeof Harness>[0]) {
  return render(<Harness {...props} />);
}

function openSettings() {
  fireEvent.click(screen.getByRole('button', { name: '列设置' }));
}

function save() {
  fireEvent.click(screen.getByRole('button', { name: /^保\s*存$/ }));
}

describe('useColumnSettings', () => {
  afterEach(() => {
    cleanup();
    window.localStorage.clear();
  });

  it('默认全部可见，隐藏列与排序保存后生效并持久化', () => {
    const storageSpy = vi.spyOn(window.localStorage, 'setItem');
    renderHarness({ tableKey: 'test:demo', scope: SCOPE });
    openSettings();
    fireEvent.click(screen.getByRole('checkbox', { name: '客户名称' }));
    fireEvent.click(screen.getByRole('button', { name: '下移业务单号' }));
    save();
    // 客户名称被隐藏，备注上移到第二列：业务单号、备注、操作
    const headers = screen
      .getAllByRole('columnheader')
      .map((node) => node.textContent);
    expect(headers).toEqual(['业务单号', '备注', '操作']);
    expect(storageSpy).toHaveBeenCalled();
    const stored = window.localStorage.getItem(
      'roncin:column-settings:v1:test:demo:u1:o1',
    );
    expect(stored).not.toBeNull();
    expect(JSON.parse(stored as string)).toEqual({
      order: ['customer', 'code', 'note', 'option'],
      hidden: ['customer'],
    });
    storageSpy.mockRestore();
  });

  it('重挂载后恢复已保存的偏好', () => {
    saveColumnSettingsPreference('test:restore', SCOPE, {
      order: ['note', 'code', 'customer', 'option'],
      hidden: ['note'],
    });
    const first = renderHarness({ tableKey: 'test:restore', scope: SCOPE });
    first.unmount();
    renderHarness({ tableKey: 'test:restore', scope: SCOPE });
    const headers = screen
      .getAllByRole('columnheader')
      .map((node) => node.textContent);
    expect(headers).toEqual(['业务单号', '客户名称', '操作']);
  });

  it('结构列不进入设置且保持在原锚点位置', () => {
    renderHarness({
      tableKey: 'test:structural',
      structuralKeys: ['option'],
    });
    openSettings();
    expect(screen.getByRole('checkbox', { name: '业务单号' })).toBeInTheDocument();
    expect(
      screen.queryByRole('checkbox', { name: '操作' }),
    ).not.toBeInTheDocument();
    save();
    const headers = screen
      .getAllByRole('columnheader')
      .map((node) => node.textContent);
    expect(headers[headers.length - 1]).toBe('操作');
  });

  it('损坏的存储偏好回退为默认列，未知 key 被忽略', () => {
    window.localStorage.setItem(
      'roncin:column-settings:v1:test:broken:u1:o1',
      '{"order":"bad","hidden":null}',
    );
    renderHarness({ tableKey: 'test:broken', scope: SCOPE });
    const headers = screen
      .getAllByRole('columnheader')
      .map((node) => node.textContent);
    expect(headers).toEqual(['业务单号', '客户名称', '备注', '操作']);
  });

  it('persist=false 时不读写本地存储', () => {
    renderHarness({ tableKey: 'test:nopersist', persist: false });
    openSettings();
    fireEvent.click(screen.getByRole('checkbox', { name: '备注' }));
    save();
    const headers = screen
      .getAllByRole('columnheader')
      .map((node) => node.textContent);
    expect(headers).toEqual(['业务单号', '客户名称', '操作']);
    expect(
      window.localStorage.getItem(
        'roncin:column-settings:v1:test:nopersist:anonymous:default',
      ),
    ).toBeNull();
  });

  it('存储写入失败时报错且弹窗保持打开', async () => {
    const setItemSpy = vi
      .spyOn(window.localStorage, 'setItem')
      .mockImplementation(() => {
        throw new Error('quota');
      });
    try {
      renderHarness({ tableKey: 'test:failure', scope: SCOPE });
      openSettings();
      fireEvent.click(screen.getByRole('checkbox', { name: '备注' }));
      save();
      expect(
        await screen.findByText(
          '列设置保存到本地浏览器失败，本次设置仅当前页面生效',
        ),
      ).toBeInTheDocument();
      // 弹窗未关闭，草稿仍可继续调整
      expect(
        screen.getByRole('button', { name: /^保\s*存$/ }),
      ).toBeInTheDocument();
      // 表格已应用本次设置
      const headers = screen
        .getAllByRole('columnheader')
        .map((node) => node.textContent);
      expect(headers).toEqual(['业务单号', '客户名称', '操作']);
    } finally {
      setItemSpy.mockRestore();
    }
  });

  it('恢复默认后保存写回默认配置', () => {
    saveColumnSettingsPreference('test:clear', SCOPE, {
      order: ['note', 'code', 'customer', 'option'],
      hidden: [],
    });
    renderHarness({ tableKey: 'test:clear', scope: SCOPE });
    openSettings();
    fireEvent.click(screen.getByRole('button', { name: '恢复默认' }));
    save();
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
      window.localStorage.getItem(
        'roncin:column-settings:v1:test:clear:u1:o1',
      ),
    ).toBeNull();
  });
});
