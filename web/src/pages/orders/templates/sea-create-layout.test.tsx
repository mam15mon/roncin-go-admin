import { ProForm } from '@ant-design/pro-components';
import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { App, Form, type FormInstance } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import {
  SeaDocumentStructure,
  SeaHouseBillIssuerSource,
} from '@/enums.generated';
import { buildSeaExportCreatePayload } from '../order-kinds/sea-export/form-adapter';
import { SeaCreateDocumentModeField } from './components/sea/SeaDocumentSection';
import { getSeaTemplateSections } from './sea-template';
import type { TemplateProps } from './types';

vi.mock('@umijs/max', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@umijs/max')>()),
  useAccess: () => ({ canOrder: () => true }),
}));

const search = async () => [];
const props: TemplateProps = {
  serviceTypeOptions: [],
  cargoCategoryOptions: [],
  locationOptions: [],
  currencyOptions: [],
  containerSpecOptions: [],
  personnelOptions: [],
  searchLocations: search,
  searchCustomers: search,
  searchShippingLines: search,
  searchBookingAgents: search,
  searchForeignAgents: search,
  searchShippingAgents: search,
  setCustomerCode: () => {},
  checkCustomerReferenceNo: async () => {},
  checkInternalReferenceNo: async () => {},
};

function CreateDocuments({
  exposeForm,
  initialValues,
}: {
  exposeForm: (form: FormInstance) => void;
  initialValues?: Record<string, unknown>;
}) {
  const [form] = Form.useForm();
  exposeForm(form);
  const sections = getSeaTemplateSections(props).filter((section) =>
    ['masterBillContent', 'houseBillContent'].includes(section.key),
  );
  return (
    <App>
      <ProForm
        form={form}
        submitter={false}
        layout="vertical"
        initialValues={initialValues}
      >
        <SeaCreateDocumentModeField />
        {sections.map((section) => (
          <section key={section.key} aria-label={section.title}>
            {section.content}
          </section>
        ))}
      </ProForm>
    </App>
  );
}

const HOUSE = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE;

describe('SE 新建 MBL 连续录入', () => {
  it('新建六区依录入顺序展示，详情继续原五区', () => {
    expect(getSeaTemplateSections(props).map(({ title }) => title)).toEqual([
      '业务归属',
      '订舱与主单识别',
      '航线与船期',
      'MBL 主单内容',
      'HBL 分单内容（HOUSE）',
      '补充与内部信息',
    ]);
    expect(
      getSeaTemplateSections({ ...props, isDetail: true }).map(
        ({ key }) => key,
      ),
    ).toEqual([
      'basicInfo',
      'transportInfo',
      'cargoAndDocumentInfo',
      'remarks',
      'internalInfo',
    ]);
  });

  it('HOUSE 同时挂载主分单且委托字段唯一，带入只改目标实际值并保持提交口径', async () => {
    let form!: FormInstance;
    const { container } = render(
      <CreateDocuments
        exposeForm={(value) => {
          form = value;
        }}
        initialValues={{
          customerId: 'customer-1',
          paymentTerm: 1,
          seaDocumentStructure: HOUSE,
          totalPackages: 10,
          totalPackageUnit: 'CTNS',
          totalGrossWeightKg: 100,
          totalVolumeCbm: 2,
          seaMasterBillContent: {
            packageCount: 11,
            grossWeightKg: 110,
            volumeCbm: 3,
          },
          seaHouseBill: {
            houseNo: 'HBL001',
            issuerSource:
              SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_SELF_ORGANIZATION,
            content: { packageCount: 12, grossWeightKg: 120, volumeCbm: 4 },
          },
        }}
      />,
    );
    const master = screen.getByRole('region', { name: 'MBL 主单内容' });
    const house = screen.getByRole('region', { name: 'HBL 分单内容（HOUSE）' });
    expect(
      within(master).getByPlaceholderText('请输入发货人英文名称与详细地址'),
    ).toBeVisible();
    expect(within(house).getByPlaceholderText('请输入分单号')).toBeVisible();
    expect(screen.queryByRole('tab')).not.toBeInTheDocument();
    for (const id of [
      'totalPackages',
      'totalPackageUnit',
      'totalGrossWeightKg',
      'totalVolumeCbm',
    ]) {
      expect(container.querySelectorAll(`[id="${id}"]`)).toHaveLength(1);
    }
    fireEvent.click(
      within(house).getByRole('button', { name: /带入委托件重尺/ }),
    );
    await waitFor(() =>
      expect(
        form.getFieldValue(['seaHouseBill', 'content', 'packageCount']),
      ).toBe(10),
    );
    expect(form.getFieldValue(['seaMasterBillContent', 'packageCount'])).toBe(
      11,
    );
    expect(form.getFieldValue('totalPackages')).toBe(10);
    const payload = buildSeaExportCreatePayload(form.getFieldsValue(true));
    expect(payload.totalPackages).toBe(10);
    expect(payload.seaDocument?.masterBillContent?.packageCount).toBe(11);
    expect(payload.seaDocument?.houseBill?.content?.packageCount).toBe(10);
    expect(payload.seaDocument?.houseBill?.houseNo).toBe('HBL001');
  });

  it('HOUSE 必填生效，切 DIRECT 清空分单并卸载必填，切回沿用委托默认值', async () => {
    let form!: FormInstance;
    render(
      <CreateDocuments
        exposeForm={(value) => {
          form = value;
        }}
        initialValues={{
          seaDocumentStructure: HOUSE,
          totalPackages: 8,
          seaMasterBillContent: { packageCount: 9 },
          seaHouseBill: { content: { packageCount: 7 } },
        }}
      />,
    );
    await act(async () => {
      await expect(form.validateFields()).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({ name: ['seaHouseBill', 'houseNo'] }),
        ]),
      });
    });
    fireEvent.click(
      screen.getByRole('radio', { name: 'DIRECT（直接交付 MBL）' }),
    );
    await waitFor(() =>
      expect(
        screen.queryByPlaceholderText('请输入分单号'),
      ).not.toBeInTheDocument(),
    );
    expect(form.getFieldValue('seaHouseBill')).toBeUndefined();
    await act(async () => {
      await expect(form.validateFields()).resolves.toBeDefined();
    });
    expect(
      buildSeaExportCreatePayload(form.getFieldsValue(true)).seaDocument
        ?.houseBill,
    ).toBeUndefined();
    fireEvent.click(screen.getByRole('radio', { name: 'HOUSE（签发 HBL）' }));
    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入分单号')).toBeVisible(),
    );
    expect(
      form.getFieldValue(['seaHouseBill', 'content', 'packageCount']),
    ).toBe(8);
    expect(form.getFieldValue(['seaMasterBillContent', 'packageCount'])).toBe(
      9,
    );
  });
});
