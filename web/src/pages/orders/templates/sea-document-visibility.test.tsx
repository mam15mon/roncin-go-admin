import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  DIRECT,
  expectHblTabPresent,
  HOUSE,
  setup,
} from './sea-document-visibility.fixtures';

describe('海运提单卡页签可见性', () => {
  it('初始 DIRECT 无 HBL 页签，切换 HOUSE 同步出现并注册必填校验', async () => {
    const { formRef } = setup(DIRECT);
    expectHblTabPresent(false);

    await act(async () => {
      fireEvent.click(
        screen.getByRole('radio', { name: '有货代分单（HOUSE）' }),
      );
    });
    await waitFor(() => {
      expect(formRef.current?.getFieldValue('seaDocumentStructure')).toBe(
        HOUSE,
      );
      expect(
        screen.getByRole('radio', { name: '有货代分单（HOUSE）' }),
      ).toBeChecked();
      expectHblTabPresent(true);
    });
    // 页签强制渲染：HBL 必填在从未点击页签时同样注册。
    await act(async () => {
      await expect(formRef.current?.validateFields()).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({ name: ['seaHouseBill', 'houseNo'] }),
        ]),
      });
    });

    // 切回操作经异步 act 收敛，同时检查 store、选中态与页签。
    await act(async () => {
      fireEvent.click(
        screen.getByRole('radio', { name: '仅船公司主单（DIRECT）' }),
      );
    });
    await waitFor(() => {
      expect(formRef.current?.getFieldValue('seaDocumentStructure')).toBe(
        DIRECT,
      );
      expect(
        screen.getByRole('radio', { name: '仅船公司主单（DIRECT）' }),
      ).toBeChecked();
      expectHblTabPresent(false);
    });
    await act(async () => {
      await expect(formRef.current?.validateFields()).resolves.toBeDefined();
    });
  });

  it('程序回填与 resetTo 同步更新页签可见性', async () => {
    const { formRef, actionsRef } = setup(HOUSE);
    await waitFor(() => expectHblTabPresent(true));
    act(() =>
      formRef.current?.setFieldsValue({ seaDocumentStructure: DIRECT }),
    );
    await waitFor(() => expectHblTabPresent(false));
    act(() => actionsRef.current?.resetTo({ seaDocumentStructure: HOUSE }));
    await waitFor(() => expectHblTabPresent(true));
  });
});
