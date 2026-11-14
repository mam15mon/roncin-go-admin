import { act, fireEvent, waitFor } from '@testing-library/react';
import { beforeEach, expect, it } from 'vitest';
import { PAYABLE, RECEIVABLE } from './feeConstants';
import {
  auditButton,
  mountForAuditRefresh,
  noteOf,
  resetAuditRefresh,
} from './OrderFeeTableTabs.audit-refresh.fixtures';
import { saveFees } from './OrderFeeTableTabs.bulkActions.fixtures';

beforeEach(resetAuditRefresh);

it('保存一侧后审核状态同步往返保留另一侧现有行输入及原提交版本', async () => {
  const { props, synchronize } = await mountForAuditRefresh();
  fireEvent.change(noteOf(`fee-${PAYABLE}`), {
    target: { value: '应付尚未保存' },
  });
  fireEvent.change(noteOf(`fee-${RECEIVABLE}`), {
    target: { value: '应收本次保存' },
  });
  await act(async () => fireEvent.click(auditButton('保存应收费用')));
  await waitFor(() => expect(props.onFeeSaved).toHaveBeenCalledWith('7'));
  await synchronize();
  expect(noteOf(`fee-${PAYABLE}`)).toHaveValue('应付尚未保存');
  await act(async () => fireEvent.click(auditButton('保存应付费用')));
  await waitFor(() => expect(saveFees).toHaveBeenCalledTimes(2));
  expect(saveFees.mock.calls[1][1].rows).toEqual([
    expect.objectContaining({
      feeId: `fee-${PAYABLE}`,
      expectedVersion: '2',
      note: '应付尚未保存',
    }),
  ]);
});
