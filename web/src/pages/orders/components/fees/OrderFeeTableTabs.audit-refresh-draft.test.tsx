import { act, fireEvent, waitFor } from '@testing-library/react';
import { beforeEach, expect, it } from 'vitest';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import { RECEIVABLE } from './feeConstants';
import {
  auditButton,
  mountForAuditRefresh,
  noteOf,
  resetAuditRefresh,
} from './OrderFeeTableTabs.audit-refresh.fixtures';
import { saveFees } from './OrderFeeTableTabs.bulkActions.fixtures';

beforeEach(resetAuditRefresh);

it('审核状态同步不移除另一侧新增行和未保存提示', async () => {
  const { props, synchronize } = await mountForAuditRefresh();
  await act(async () => {
    fireEvent.click(auditButton('新增应付费用'));
  });
  const draft = await waitFor(() => {
    const row = document.querySelector('tr[data-row-key^="new_"]');
    expect(row).not.toBeNull();
    return row as HTMLElement;
  });
  const draftId = draft.dataset.rowKey as string;
  fireEvent.change(noteOf(draftId), { target: { value: '应付新增草稿' } });
  fireEvent.change(noteOf(`fee-${RECEIVABLE}`), {
    target: { value: '保存应收后刷新审核' },
  });
  await act(async () => fireEvent.click(auditButton('保存应收费用')));
  await waitFor(() => expect(props.onFeeSaved).toHaveBeenCalledWith('7'));
  await synchronize();
  expect(noteOf(draftId)).toHaveValue('应付新增草稿');
  expect(getAnyDirtyTabKey()).toBe('/orders/sea-export/order-1/fees');
  expect(saveFees.mock.calls[0][1].direction).toBe(RECEIVABLE);
});
