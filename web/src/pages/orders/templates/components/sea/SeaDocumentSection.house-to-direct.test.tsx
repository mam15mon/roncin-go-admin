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

    const switchButton = (
      await screen.findByText('切换为仅船公司主单（DIRECT）')
    ).closest('button') as HTMLButtonElement;
    await waitFor(() => {
      expect(switchButton).toBeVisible();
      expect(switchButton).toBeEnabled();
    });
    expect(switchButton).toHaveAccessibleName(/切换为仅船公司主单（DIRECT）/);
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
    // Modal 的 prepare 阶段可能已挂载内容；只等待目标按钮的实际可见性。
    const previewButton = (await screen.findByText('预览切换影响')).closest(
      'button',
    ) as HTMLButtonElement;
    await waitFor(() => {
      expect(previewButton).toBeVisible();
      expect(previewButton).toBeEnabled();
    });
    expect(previewButton).toHaveAccessibleName('预览切换影响');
    fireEvent.click(previewButton);

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
    const executeButton = (await screen.findByText('确认执行')).closest(
      'button',
    ) as HTMLButtonElement;
    await waitFor(() => {
      expect(executeButton).toBeVisible();
      expect(executeButton).toBeEnabled();
    });
    expect(executeButton).toHaveAccessibleName('确认执行');
    fireEvent.click(executeButton);
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
