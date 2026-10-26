import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  SeaDocumentStructure,
  SeaHouseBillIssuerSource,
} from '@/enums.generated';
import {
  fillExternalConfirmation,
  mockDocuments,
  previewMode,
  setupDocumentTests,
  TestForm,
} from './SeaDocumentSection.fixtures';

setupDocumentTests();
describe('SeaDocumentSectionComponent', () => {
  it('DIRECT→HOUSE 提交唯一 HBL，确认信息变化后要求重新预览', async () => {
    mockDocuments(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT);
    render(
      <TestForm isDetail initialValues={{ id: 'order-1', version: '11' }} />,
    );

    fireEvent.click(
      await screen.findByRole('button', { name: /切换为有货代分单（HOUSE）/ }),
    );
    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: 'HBL-NEW' },
    });
    fireEvent.click(screen.getByRole('radio', { name: '本公司' }));
    fireEvent.change(
      screen.getByPlaceholderText('说明客户请求及本次 HOUSE/DIRECT 切换原因'),
      {
        target: { value: '客户要求签发分单' },
      },
    );
    fillExternalConfirmation();
    fireEvent.click(screen.getByRole('button', { name: '预览切换影响' }));

    await screen.findByRole('button', { name: '确认执行' });
    expect(previewMode).toHaveBeenLastCalledWith(
      { orderId: 'order-1' },
      expect.objectContaining({
        targetMode: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
        newHouseBill: expect.objectContaining({
          houseNo: 'HBL-NEW',
          issuerSource:
            SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION,
        }),
      }),
    );

    fireEvent.change(
      screen.getByPlaceholderText('记录确认渠道、联系人和确认结论'),
      {
        target: { value: '船代再次邮件确认可以变更' },
      },
    );
    expect(
      screen.queryByRole('button', { name: '确认执行' }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: '预览切换影响' })).toBeEnabled();
  }, 60000);
});
