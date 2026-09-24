import { fireEvent, render, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { FEE_LEDGER_FIELDS, getDefaultRowColors } from './fields-meta';
import { TableColumnConfigModal } from './TableColumnConfigModal';

const updatePreferenceMock = vi.fn();

vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceUpdateFeeLedgerPreference: (...args: unknown[]) =>
    updatePreferenceMock(...args),
}));

function buildPreference(
  overrides: Partial<API.FeeLedgerPreference> = {},
): API.FeeLedgerPreference {
  return {
    columns: FEE_LEDGER_FIELDS.map((field) => ({
      fieldKey: field.key,
      visible: field.defaultVisible,
    })),
    pageSize: 40,
    rowColors: getDefaultRowColors(),
    version: '3',
    customized: true,
    ...overrides,
  };
}

function renderModal(currentPreference?: API.FeeLedgerPreference) {
  const onSaved = vi.fn();
  const onClose = vi.fn();
  const view = render(
    <App>
      <TableColumnConfigModal
        open
        onClose={onClose}
        currentPreference={currentPreference}
        onSaved={onSaved}
      />
    </App>,
  );
  return { onSaved, onClose, ...view };
}

/** FEE_LEDGER_FIELDS 之外的扩展字段：早期元数据遗留，现无对应表格列。 */
const REMOVED_FIELD_KEY = 'creatorName';

describe('TableColumnConfigModal 保存载荷', () => {
  beforeEach(() => {
    updatePreferenceMock.mockReset();
    updatePreferenceMock.mockResolvedValue({
      success: true,
      data: buildPreference(),
    });
  });

  it('未选择排序字段时保存不携带排序方向，并回传版本号', async () => {
    const { getByRole, onSaved, onClose } = renderModal(buildPreference());
    fireEvent.click(getByRole('button', { name: /^保\s*存$/ }));
    await waitFor(() => expect(onSaved).toHaveBeenCalled());
    expect(updatePreferenceMock).toHaveBeenCalledTimes(1);
    const payload = updatePreferenceMock.mock.calls[0][0];
    expect(payload.sortField).toBeUndefined();
    expect(payload.sortDirection).toBeUndefined();
    expect(payload.version).toBe('3');
    expect(
      payload.columns.map((item: { fieldKey: string }) => item.fieldKey),
    ).toEqual(FEE_LEDGER_FIELDS.map((field) => field.key));
    expect(onClose).toHaveBeenCalled();
  });

  it('存量排序字段指向已删除字段时清洗为无排序', async () => {
    const { getByRole } = renderModal(
      buildPreference({
        sortField: REMOVED_FIELD_KEY,
        sortDirection: 'ASC',
      }),
    );
    fireEvent.click(getByRole('button', { name: /^保\s*存$/ }));
    await waitFor(() => expect(updatePreferenceMock).toHaveBeenCalled());
    const payload = updatePreferenceMock.mock.calls[0][0];
    expect(payload.sortField).toBeUndefined();
    expect(payload.sortDirection).toBeUndefined();
  });

  it('存量排序字段为旧 key 时规范化为新 key 并保留方向', async () => {
    const { getByRole } = renderModal(
      buildPreference({
        sortField: 'settlementPartyName',
        sortDirection: 'ASC',
      }),
    );
    fireEvent.click(getByRole('button', { name: /^保\s*存$/ }));
    await waitFor(() => expect(updatePreferenceMock).toHaveBeenCalled());
    const payload = updatePreferenceMock.mock.calls[0][0];
    expect(payload.sortField).toBe('settlementPartyId');
    expect(payload.sortDirection).toBe('ASC');
  });

  it('有效排序字段与方向原样提交', async () => {
    const { getByRole } = renderModal(
      buildPreference({ sortField: 'expenseDate', sortDirection: 'DESC' }),
    );
    fireEvent.click(getByRole('button', { name: /^保\s*存$/ }));
    await waitFor(() => expect(updatePreferenceMock).toHaveBeenCalled());
    const payload = updatePreferenceMock.mock.calls[0][0];
    expect(payload.sortField).toBe('expenseDate');
    expect(payload.sortDirection).toBe('DESC');
  });
});
