import { fireEvent, screen } from '@testing-library/react';
import { beforeEach, expect, it } from 'vitest';
import {
  makeProps,
  renderTabs,
  resetBulkActionMocks,
} from './OrderFeeTableTabs.bulkActions.fixtures';

beforeEach(resetBulkActionMocks);
it.each([0, 1])(
  '费用表 %i 保留批量结算和删除入口，不提供业务标签操作',
  async (tableIndex) => {
    const { props } = makeProps('order-1');
    props.selectedReceivableFeeIds = ['fee-a'];
    props.selectedPayableFeeIds = ['fee-b'];
    await renderTabs(props);
    const buttons = screen.getAllByRole('button', { name: /批量操作/ });
    const button = buttons[tableIndex];
    fireEvent.click(button);
    expect(await screen.findByText('批量改结算单位')).toBeInTheDocument();
    expect(screen.getByText('批量删除')).toBeInTheDocument();
    expect(screen.queryByText('添加标签')).not.toBeInTheDocument();
    expect(screen.queryByText('删除标签')).not.toBeInTheDocument();
  },
);
