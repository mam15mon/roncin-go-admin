import { ProForm } from '@ant-design/pro-components';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { expect, it, vi } from 'vitest';
import type { TemplateProps } from '../../types';
import { getSeaBaseInfoFields } from './SeaBasicInfoSection';

it('船司英文别名命中的中文标签不被二次过滤', async () => {
  const searchShippingLines = vi.fn(async (keyword?: string) =>
    keyword === 'english-alias'
      ? [{ label: '甲船公司 (AAA)', value: 'shipping-line-1' }]
      : [],
  );
  const onFinish = vi.fn();
  const field = getSeaBaseInfoFields({
    searchShippingLines,
  } as unknown as TemplateProps).carrier;
  render(
    <ProForm submitter={false} onFinish={onFinish}>
      {field}
      <button type="submit">保存船司</button>
    </ProForm>,
  );
  fireEvent.mouseDown(screen.getByRole('combobox'));
  fireEvent.change(screen.getByRole('combobox'), {
    target: { value: 'english-alias' },
  });
  fireEvent.click(await screen.findByText('甲船公司 (AAA)'));
  fireEvent.click(screen.getByRole('button', { name: '保存船司' }));
  await waitFor(() =>
    expect(onFinish).toHaveBeenCalledWith({
      shippingLineId: 'shipping-line-1',
    }),
  );
});
