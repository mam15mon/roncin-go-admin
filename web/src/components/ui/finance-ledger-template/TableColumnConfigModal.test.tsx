import { fireEvent, render, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FEE_LEDGER_FIELDS, getDefaultRowColors } from './fields-meta';
import { TableColumnConfigModal } from './TableColumnConfigModal';
import type { FinanceLedgerViewConfig } from './types';

function buildConfig(
  overrides: Partial<FinanceLedgerViewConfig> = {},
): FinanceLedgerViewConfig {
  return {
    columns: {
      order: FEE_LEDGER_FIELDS.map((field) => field.key),
      hidden: [],
    },
    rowColors: getDefaultRowColors(),
    ...overrides,
  };
}

function renderModal(
  currentConfig?: FinanceLedgerViewConfig,
  onSave = vi.fn((_config: FinanceLedgerViewConfig) => true),
) {
  const onClose = vi.fn();
  const view = render(
    <App>
      <TableColumnConfigModal
        open
        onClose={onClose}
        currentConfig={currentConfig}
        onSave={onSave}
      />
    </App>,
  );
  return { onClose, onSave, ...view };
}

/** FEE_LEDGER_FIELDS 之外的扩展字段：早期元数据遗留，现无对应表格列。 */
const REMOVED_FIELD_KEY = 'creatorName';

describe('TableColumnConfigModal 本地保存', () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  it('无存量配置时保存提交默认列顺序与默认配色', async () => {
    const onSave = vi.fn((_config: FinanceLedgerViewConfig) => true);
    const { getByRole, onClose } = renderModal(undefined, onSave);
    fireEvent.click(getByRole('button', { name: /^保\s*存$/ }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    const config = onSave.mock.calls[0]?.[0] as FinanceLedgerViewConfig;
    expect(config.columns.order).toEqual(
      FEE_LEDGER_FIELDS.map((field) => field.key),
    );
    expect(config.columns.hidden).toEqual([]);
    expect(config.rowColors).toEqual(getDefaultRowColors());
    expect(onClose).toHaveBeenCalled();
  });

  it('存量配置中的未知 key 被清洗，缺失字段按默认顺序补齐', async () => {
    const onSave = vi.fn((_config: FinanceLedgerViewConfig) => true);
    const { getByRole } = renderModal(
      buildConfig({
        columns: {
          order: ['note', REMOVED_FIELD_KEY, 'totalAmount'],
          hidden: ['totalAmount'],
        },
      }),
      onSave,
    );
    fireEvent.click(getByRole('button', { name: /^保\s*存$/ }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    const config = onSave.mock.calls[0]?.[0] as FinanceLedgerViewConfig;
    // 已知列保持用户顺序（note 在前），未知 key 不进入配置，
    // 未提及字段按默认顺序补齐且无重复。
    expect(config.columns.order[0]).toBe('note');
    expect(config.columns.order).not.toContain(REMOVED_FIELD_KEY);
    expect(config.columns.order).toHaveLength(FEE_LEDGER_FIELDS.length);
    expect(new Set(config.columns.order).size).toBe(FEE_LEDGER_FIELDS.length);
    expect(config.columns.hidden).toEqual(['totalAmount']);
  });

  it('本地写入失败时保持弹窗打开且不关闭', async () => {
    const onSave = vi.fn((_config: FinanceLedgerViewConfig) => false);
    const { getByRole, onClose } = renderModal(undefined, onSave);
    fireEvent.click(getByRole('button', { name: /^保\s*存$/ }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(onClose).not.toHaveBeenCalled();
    expect(getByRole('button', { name: /^保\s*存$/ })).toBeTruthy();
  });

  it('修改后的行配色随保存一并提交', async () => {
    const onSave = vi.fn((_config: FinanceLedgerViewConfig) => true);
    const customColors = {
      ...getDefaultRowColors(),
      unbilled: '#FF0000',
    };
    const { getByRole } = renderModal(
      buildConfig({ rowColors: customColors }),
      onSave,
    );
    fireEvent.click(getByRole('button', { name: /^保\s*存$/ }));
    await waitFor(() => expect(onSave).toHaveBeenCalledTimes(1));
    expect(onSave.mock.calls[0]?.[0]?.rowColors.unbilled).toBe('#FF0000');
  });
});
