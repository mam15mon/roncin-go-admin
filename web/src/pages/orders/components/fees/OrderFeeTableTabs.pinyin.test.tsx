import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, expect, it } from 'vitest';
import {
  makeProps,
  renderTabs,
  resetBulkActionMocks,
} from './OrderFeeTableTabs.bulkActions.fixtures';

beforeEach(resetBulkActionMocks);
it('费用名称行内编辑可按全拼、首字母、英文名与别名选择真实费目', async () => {
  const { props } = makeProps('order-pinyin');
  const feeSettings: API.OrderFeeSettingOption[] = [
    {
      id: 'setting-of',
      feeCode: 'OF',
      nameZh: '海运费',
      nameEn: 'Ocean Freight',
      aliasName: '运费别名',
    },
    { id: 'setting-tc', feeCode: 'TC', nameZh: '拖车费' },
  ];
  await renderTabs({ ...props, feeSettings });
  await act(async () => {
    fireEvent.click(screen.getByRole('button', { name: /新增应收费用/ }));
  });
  const input = screen.getAllByRole('combobox')[0];
  fireEvent.mouseDown(input);
  for (const keyword of ['haiyunfei', 'hyf', 'yfbm', 'freight']) {
    fireEvent.change(input, { target: { value: keyword } });
    await waitFor(() => {
      expect(
        document.querySelector('.ant-select-item-option-content')?.textContent,
      ).toBe('海运费 (OF)');
      expect(document.querySelectorAll('.ant-select-item-option')).toHaveLength(
        1,
      );
    });
  }
});
