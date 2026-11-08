import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { Form } from 'antd';
import { expect, it, vi } from 'vitest';
import {
  SeaDocumentEventType,
  SeaDocumentStructure,
  SeaDocumentType,
  SeaHouseBillIssuerSource,
} from '@/enums.generated';
import { SeaDocumentSectionComponent } from './SeaDocumentSection';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: () => true,
    canOperateOrganization: () => true,
  }),
}));
vi.mock('./SeaBillContentFormFields', () => ({
  SeaBillContentFormFields: () => null,
}));

const historyMocks = vi.hoisted(() => ({ versions: vi.fn(), events: vi.fn() }));
vi.mock('@/services/roncin/seaDocumentService', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/services/roncin/seaDocumentService')
  >()),
  seaDocumentServiceListSeaMasterBillVersions: historyMocks.versions,
  seaDocumentServiceListSeaDocumentEvents: historyMocks.events,
}));

const documents: API.SeaOrderDocuments = {
  orderId: 'order-1',
  documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
  linkVersion: '2',
  masterBill: {
    id: 'mbl-1',
    masterNo: 'MBL001',
    version: '2',
    memberCount: 0,
    currentVersionId: 'v2',
  },
  houseBill: {
    id: 'hbl-1',
    houseNo: 'HBL001',
    version: '1',
    issuerSource:
      SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION,
  },
};
function Harness() {
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
      <SeaDocumentSectionComponent isDetail disabled documents={documents} />
    </Form>
  );
}
it('退关只读入口可查看v1、v2和改单事件，写操作仍禁用', async () => {
  historyMocks.versions.mockResolvedValue({
    data: [
      {
        id: 'v2',
        documentType: SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL,
        versionNo: '2',
        documentNo: 'MBL001',
        content: { shipperText: '新版发货人' },
      },
      {
        id: 'v1',
        documentType: SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL,
        versionNo: '1',
        documentNo: 'MBL001',
        content: { shipperText: '原发货人' },
      },
    ],
    total: '2',
  });
  historyMocks.events.mockResolvedValue({
    data: [
      {
        id: 'amend-1',
        documentId: 'mbl-1',
        documentNo: 'MBL001',
        eventType: SeaDocumentEventType.SEA_DOCUMENT_EVENT_TYPE_AMENDMENT,
        reason: '退关前更正发货人',
      },
    ],
    total: '1',
  });
  renderWithApp(<Harness />);
  const main = await screen.findByRole('tabpanel', { name: /主单/ });
  const mainHistory = within(main).getByRole('button', { name: /版本与事件/ });
  expect(mainHistory).toBeEnabled();
  expect(within(main).getByRole('button', { name: /单\s*改/ })).toBeDisabled();
  expect(within(main).getByRole('button', { name: /作废/ })).toBeDisabled();
  fireEvent.click(mainHistory);
  const drawer = await screen.findByRole('dialog', {
    name: /MBL001.*不可变版本与事件/,
  });
  expect(await within(drawer).findByText('v2')).toBeVisible();
  expect(await within(drawer).findByText('v1')).toBeVisible();
  expect(await within(drawer).findByText('退关前更正发货人')).toBeVisible();
  await waitFor(() =>
    expect(historyMocks.versions).toHaveBeenCalledWith({
      orderId: 'order-1',
      page: 1,
      pageSize: 200,
    }),
  );
  fireEvent.click(within(drawer).getAllByRole('button', { name: '展开行' })[1]);
  expect(await within(drawer).findByText(/原发货人/)).toBeVisible();
});
