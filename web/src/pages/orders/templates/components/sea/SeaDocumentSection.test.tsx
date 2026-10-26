import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { describe, expect, it } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import { setupDocumentTests, TestForm } from './SeaDocumentSection.fixtures';

setupDocumentTests();
describe('SeaDocumentSectionComponent', () => {
  it('新建默认 HOUSE 分单制，切 DIRECT 隐藏 HBL 但保留草稿，切回恢复', async () => {
    let form: FormInstance | undefined;
    render(
      <TestForm
        includeModeField
        exposeForm={(value) => {
          form = value;
        }}
      />,
    );

    // 全员分单制：新建默认 HOUSE，出现 HBL 分单页签且无「请先选择模式」提示。
    await waitFor(() => {
      expect(
        screen.getByRole('tab', { name: /分单 \(HBL\)/ }),
      ).toBeInTheDocument();
      expect(form?.getFieldValue('seaDocumentStructure')).toBe(
        SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
      );
    });
    expect(screen.queryByText('请先选择提单模式')).not.toBeInTheDocument();

    // 新建号码在主单旁独立注册，无需切换 HBL 页签。
    expect(screen.getByPlaceholderText('请输入分单号')).toBeInTheDocument();
    expect(screen.queryByText(/添加分单/)).not.toBeInTheDocument();
    expect(screen.queryByText('删除分单')).not.toBeInTheDocument();
    expect(screen.queryByText('箱货分配')).not.toBeInTheDocument();

    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: 'HBL-NEW' },
    });
    fireEvent.click(
      screen.getByRole('radio', { name: '仅船公司主单（DIRECT）' }),
    );
    await waitFor(() =>
      expect(
        screen.queryByPlaceholderText('请输入分单号'),
      ).not.toBeInTheDocument(),
    );
    // 隐藏草稿保留在 Form store，仅页签与提交口径排除 HBL。
    expect(form?.getFieldValue(['seaHouseBill', 'houseNo'])).toBe('HBL-NEW');
    expect(form?.getFieldValue('seaDocumentStructure')).toBe(
      SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
    );

    // 切回 HOUSE 恢复分单录入与草稿。
    fireEvent.click(screen.getByRole('radio', { name: '有货代分单（HOUSE）' }));
    fireEvent.click(await screen.findByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入分单号')).toHaveValue(
        'HBL-NEW',
      ),
    );
  });
});
