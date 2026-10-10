import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
  activeBillPane,
  DIRECT,
  HOUSE,
  setup,
} from './sea-document-visibility.fixtures';
import { revealSeaFormErrors } from './sea-form-error-reveal';

describe('提单页签校验失败定位链路', () => {
  it('提交校验命中 HBL 首个错误时自动切到 HBL 页签', async () => {
    const { formRef } = setup(HOUSE);
    // 默认 MBL 页签在前台，HBL 输入不可见。
    expect(
      within(activeBillPane()).queryByPlaceholderText('请输入分单号'),
    ).toBeNull();

    await act(async () => {
      formRef.current?.submit();
    });
    await waitFor(() =>
      expect(
        within(activeBillPane()).getByPlaceholderText('请输入分单号'),
      ).toBeVisible(),
    );
  });

  it('DIRECT 无 HBL 页签时提交直接通过，不触发隐藏校验', async () => {
    const onFinish = vi.fn().mockResolvedValue(true);
    const { formRef } = setup(DIRECT, onFinish);
    await act(async () => {
      formRef.current?.submit();
    });
    await waitFor(() => expect(onFinish).toHaveBeenCalled());
  });

  it('revealSeaFormErrors 按首个错误切换前台页签', async () => {
    setup(HOUSE);
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() =>
      expect(
        within(activeBillPane()).getByPlaceholderText('请输入分单号'),
      ).toBeVisible(),
    );
    act(() => {
      revealSeaFormErrors([
        { name: ['seaMasterBillContent', 'shipperText'], errors: ['必填'] },
      ]);
    });
    await waitFor(() =>
      expect(
        within(activeBillPane()).getByPlaceholderText(
          '请输入发货人英文名称与详细地址',
        ),
      ).toBeVisible(),
    );
  });
});
