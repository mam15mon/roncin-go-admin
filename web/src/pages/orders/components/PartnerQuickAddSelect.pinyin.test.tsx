import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { expect, it } from 'vitest';
import { searchPartners, TestHost } from './PartnerQuickAddSelect.fixtures';

it('客商别名拼音命中的官方名称保持可选并提交真实ID', async () => {
  searchPartners.mockImplementation(async (keyword) =>
    keyword === 'haiyundaili'
      ? [{ label: '甲公司', value: 'partner-alias', code: 'P-ALIAS' }]
      : [],
  );
  render(<TestHost />);
  fireEvent.mouseDown(screen.getByRole('combobox'));
  fireEvent.change(screen.getByRole('combobox'), {
    target: { value: 'haiyundaili' },
  });
  const option = await screen.findByText('甲公司');
  fireEvent.click(option);
  await waitFor(() =>
    expect(screen.getByTestId('customer-id')).toHaveTextContent(
      'partner-alias',
    ),
  );
  expect(searchPartners).toHaveBeenCalledWith(
    'haiyundaili',
    expect.objectContaining({
      signal: expect.any(AbortSignal),
      skipErrorHandler: true,
    }),
  );
});
