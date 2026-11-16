import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, screen, waitFor } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { describe, expect, it } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import {
  getDocuments,
  mockDocuments,
  setupDocumentTests,
  TestForm,
} from './SeaDocumentSection.fixtures';

setupDocumentTests();
describe('海运单证查询与编辑基线', () => {
  it('背景重查成功不覆盖正文或提交基线', async () => {
    mockDocuments(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT);
    let form!: FormInstance;
    const view = renderWithClient(
      <TestForm
        isDetail
        initialValues={{ id: 'order-1', version: '11' }}
        exposeForm={(next) => {
          form = next;
        }}
      />,
    );
    await waitFor(() =>
      expect(form.getFieldValue('seaDocumentLinkVersion')).toBe('5'),
    );
    act(() =>
      form.setFieldValue(['seaMasterBillContent', 'shipperText'], '未保存输入'),
    );
    getDocuments.mockResolvedValue({
      success: true,
      data: {
        orderId: 'order-1',
        documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
        linkVersion: '6',
        masterBill: {
          id: 'mbl-id',
          masterNo: 'MBL001',
          version: '8',
          content: { shipperText: '他人更新' },
        },
      },
    });
    await act(async () => {
      await view.queryClient.invalidateQueries({
        queryKey: ['orders', 'sea-documents'],
      });
    });
    expect(form.getFieldValue(['seaMasterBillContent', 'shipperText'])).toBe(
      '未保存输入',
    );
    expect(form.getFieldValue('seaDocumentLinkVersion')).toBe('5');
    expect(form.getFieldValue('seaMasterBillContentVersion')).toBe('7');
  });

  it('全局消费后的 undefined 保持失败而非正常空单证', async () => {
    getDocuments.mockResolvedValue(undefined as never);
    renderWithClient(
      <TestForm isDetail initialValues={{ id: 'order-1', version: '11' }} />,
    );
    expect(await screen.findByText('海运单证加载失败，请重试')).toBeVisible();
    expect(
      screen.getByRole('button', { name: /切换为有货代分单/ }),
    ).toBeDisabled();
  });
});
