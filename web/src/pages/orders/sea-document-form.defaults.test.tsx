import { ProForm } from '@ant-design/pro-components';
import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, waitFor } from '@testing-library/react';
import { App, ConfigProvider, Form, type FormInstance } from 'antd';
import { expect, it, vi } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import {
  buildSeaDocumentUpdate,
  seaDocumentFormValues,
} from './sea-document-form';
import { SeaDocumentSectionComponent } from './templates/components/sea/SeaDocumentSection';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOrder: () => true,
    canOperateOrganization: () => true,
  }),
}));
vi.mock('@/services/roncin/orderReleasePodService', () => ({
  orderReleasePodServiceListReleasePods: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));
vi.mock('./templates/components/sea/SeaDocumentHistoryActions', () => ({
  default: () => null,
}));

it.each([undefined, { billForm: 'SEAWAY BILL', releaseType: '正本' }])(
  '真实单证表单默认值%s不会把未改MBL/HBL判为变更',
  async (terms) => {
    const documents: API.SeaOrderDocuments = {
      documentStructure: SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
      linkVersion: '3',
      masterBill: {
        id: 'm',
        version: '5',
        content: { shipperText: 'MBL', ...terms },
      },
      houseBill: {
        id: 'h',
        version: '7',
        houseNo: 'HBL',
        issuerSource: 1,
        status: 1,
        content: { shipperText: 'HBL', ...terms },
      },
    };
    let form!: FormInstance;
    function Harness() {
      [form] = Form.useForm();
      return (
        <ConfigProvider theme={{ token: { motion: false } }}>
          <App>
            <ProForm
              form={form}
              submitter={false}
              initialValues={{
                id: 'order',
                version: '1',
                ...seaDocumentFormValues(documents),
              }}
            >
              <SeaDocumentSectionComponent isDetail documents={documents} />
            </ProForm>
          </App>
        </ConfigProvider>
      );
    }
    renderWithClient(<Harness />);
    await act(async () => {});
    await waitFor(() =>
      expect(
        form.getFieldValue(['seaMasterBillContent', 'billForm']),
      ).toBeTruthy(),
    );
    expect(
      buildSeaDocumentUpdate(form.getFieldsValue(true), documents),
    ).toBeUndefined();
    act(() =>
      form.setFieldValue(['seaHouseBill', 'content', 'shipperText'], '修改HBL'),
    );
    const update = buildSeaDocumentUpdate(form.getFieldsValue(true), documents);
    expect(update?.masterBillContent).toBeUndefined();
    expect(update?.houseBill?.content?.shipperText).toBe('修改HBL');
  },
);
