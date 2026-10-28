import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import {
  executeMode,
  fillExternalConfirmation,
  mockDocuments,
  previewMode,
  setupDocumentTests,
  TestForm,
} from './SeaDocumentSection.fixtures';

setupDocumentTests();
describe('SeaDocumentSectionComponent', () => {
  it('HOUSE→DIRECT 预览后携带外部确认和当前 HBL 版本执行', async () => {
    mockDocuments(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE);
    render(
      <TestForm isDetail initialValues={{ id: 'order-1', version: '11' }} />,
    );

    const switchButton = await screen.findByRole('button', {
      name: /切换为仅船公司主单（DIRECT）/,
    });
    expect(switchButton).toBeEnabled();
    expect(screen.getAllByText('关联放货记录').length).toBeGreaterThan(0);
    expect(screen.getAllByText('暂无关联放货记录').length).toBeGreaterThan(0);
    fireEvent.click(switchButton);
    fireEvent.change(
      screen.getByPlaceholderText('说明客户请求及本次 HOUSE/DIRECT 切换原因'),
      {
        target: { value: '客户要求改为直单' },
      },
    );
    fillExternalConfirmation();
    fireEvent.click(screen.getByRole('button', { name: '预览切换影响' }));

    await waitFor(() => {
      expect(previewMode).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        expect.objectContaining({
          orderId: 'order-1',
          targetMode: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
          reason: '客户要求改为直单',
          newHouseBill: undefined,
        }),
      );
    });
    fireEvent.click(await screen.findByRole('button', { name: '确认执行' }));
    await waitFor(() => {
      expect(executeMode).toHaveBeenCalledWith(
        { orderId: 'order-1' },
        expect.objectContaining({
          expectedOrderVersion: '11',
          expectedLinkVersion: '5',
          expectedHouseBillVersion: '3',
          expectedCurrentVersionId: 'hbl-version-id',
          targetMode: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
          confirmation: expect.objectContaining({
            confirmedByParty: '测试船代',
            confirmationNote: '船代邮件确认可以变更',
          }),
        }),
      );
    });
  });
});
