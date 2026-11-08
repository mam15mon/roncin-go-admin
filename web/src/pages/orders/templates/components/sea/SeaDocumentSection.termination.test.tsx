import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { Form } from 'antd';
import { expect, it, vi } from 'vitest';
import {
  SeaDocumentStructure,
  SeaHouseBillIssuerSource,
} from '@/enums.generated';
import { SeaDocumentSectionComponent } from './SeaDocumentSection';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: () => false,
    canOperateOrganization: () => false,
  }),
}));
vi.mock('./SeaBillContentFormFields', () => ({
  SeaBillContentFormFields: () => null,
}));

const documents: API.SeaOrderDocuments = {
  orderId: 'order-1',
  documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
  linkVersion: '2',
  masterBill: { id: 'mbl-1', masterNo: 'MBL001', version: '1', memberCount: 0 },
  houseBill: {
    id: 'hbl-1',
    houseNo: 'HBL001',
    version: '1',
    issuerSource:
      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION,
  },
};
function Harness({ terminated }: { terminated: boolean }) {
  const [form] = Form.useForm();
  return (
    <Form
      form={form}
      initialValues={{
        id: 'order-1',
        version: '3',
        seaDocumentStructure: documents.documentStructure,
      }}
    >
      <SeaDocumentSectionComponent
        isDetail
        disabled
        documents={documents}
        masterHistoryDisabled={terminated}
      />
    </Form>
  );
}
it('终态明确关闭主单历史入口，保留分单历史和一般只读订单的主单历史', async () => {
  const view = renderWithApp(<Harness terminated />);
  const main = await screen.findByRole('tabpanel', { name: /主单/ });
  const mainHistory = within(main).getByRole('button', { name: /版本与事件/ });
  expect(mainHistory).toBeDisabled();
  expect(mainHistory).toHaveAttribute(
    'title',
    '退关后暂不支持查看主单版本与事件',
  );
  fireEvent.click(screen.getByRole('tab', { name: /分单/ }));
  const house = await screen.findByRole('tabpanel', { name: /分单/ });
  await waitFor(() =>
    expect(
      within(house).getByRole('button', { name: /版本与事件/ }),
    ).toBeEnabled(),
  );
  view.rerender(<Harness terminated={false} />);
  fireEvent.click(screen.getByRole('tab', { name: /主单/ }));
  expect(
    within(main).getByRole('button', { name: /版本与事件/ }),
  ).toBeEnabled();
});
