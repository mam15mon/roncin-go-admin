import { render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import {
  executeMode,
  getDocuments,
  mockDocuments,
  previewMode,
  setupDocumentTests,
  setWorkspaceCanOperate,
  TestForm,
} from './SeaDocumentSection.fixtures';

setupDocumentTests();
describe('SeaDocumentSectionComponent', () => {
  it('外公司单证详情可查看，但不能切换提单模式', async () => {
    setWorkspaceCanOperate(false);
    mockDocuments(SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE);
    render(
      <TestForm
        isDetail
        disabled
        initialValues={{ id: 'order-1', version: '11' }}
      />,
    );
    await waitFor(() => expect(getDocuments).toHaveBeenCalled());
    await screen.findByText('MBL001');
    expect(
      screen.queryByRole('button', { name: /切换为仅船公司主单（DIRECT）/ }),
    ).not.toBeInTheDocument();
    expect(previewMode).not.toHaveBeenCalled();
    expect(executeMode).not.toHaveBeenCalled();
  });
});
