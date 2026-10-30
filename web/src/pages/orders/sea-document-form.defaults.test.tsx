import { ProForm } from '@ant-design/pro-components';
import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App, ConfigProvider, Form, type FormInstance } from 'antd';
import { expect, it, vi } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import {
  buildSeaDocumentUpdate,
  seaDocumentFormValues,
} from './sea-document-form';
import { SeaCargoDescriptionFields } from './templates/components/sea/SeaCargoSection';
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
    const consoleError = vi.spyOn(console, 'error');
    const consoleWarn = vi.spyOn(console, 'warn');
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
                paymentTerm: 2,
                ...seaDocumentFormValues(documents),
              }}
            >
              <SeaCargoDescriptionFields />
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
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() => {
      expect(
        document.querySelector('#seaHouseBill_content_billForm'),
      ).not.toBeNull();
    });
    expect(form.getFieldValue('paymentTerm')).toBe(2);
    for (const prefix of [
      ['seaMasterBillContent'],
      ['seaHouseBill', 'content'],
    ]) {
      expect(form.getFieldValue([...prefix, 'billForm'])).toBe(
        terms?.billForm ?? 'ORIGINAL',
      );
      expect(form.getFieldValue([...prefix, 'releaseType'])).toBe(
        terms?.releaseType ?? '电放',
      );
    }
    const warnings = [...consoleError.mock.calls, ...consoleWarn.mock.calls]
      .flat()
      .join('\n');
    expect(warnings).not.toContain("Form already set 'initialValues'");
    consoleError.mockRestore();
    consoleWarn.mockRestore();
    act(() =>
      form.setFieldValue(['seaHouseBill', 'content', 'shipperText'], '修改HBL'),
    );
    const update = buildSeaDocumentUpdate(form.getFieldsValue(true), documents);
    expect(update?.masterBillContent).toBeUndefined();
    expect(update?.houseBill?.content?.shipperText).toBe('修改HBL');
  },
);
