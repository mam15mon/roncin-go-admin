import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import { SearchFilterTemplate } from './SearchFilterTemplate';

it('同一筛选模板的经营身份就绪后重新加载候选，关闭身份后隐藏旧结果', async () => {
  const request = vi.fn(async () => [{ label: '公司客户', value: 'customer' }]);
  const view = (enabled: boolean) => (
    <SearchFilterTemplate
      collapsible={false}
      items={[
        {
          name: 'settlementPartyId',
          label: '结算单位',
          type: 'searchable-select',
          requestKey: [enabled],
          request: () => (enabled ? request() : Promise.resolve([])),
        },
      ]}
    />
  );
  const rendered = renderWithClient(view(false));
  expect(request).not.toHaveBeenCalled();
  rendered.rerender(view(true));
  await waitFor(() => expect(request).toHaveBeenCalledOnce());
  fireEvent.mouseDown(screen.getByRole('combobox'));
  expect(await screen.findByText('公司客户')).toBeInTheDocument();
  rendered.rerender(view(false));
  expect(screen.queryByText('公司客户')).toBeNull();
  expect(request).toHaveBeenCalledOnce();
});
