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
import * as orderService from '@/services/roncin/orderService';
import { buildSeaExportCreatePayload } from '../order-kinds/sea-export/form-adapter';
import { SeaCreateDocumentModeField } from './components/sea/SeaDocumentSection';
import { getSeaTemplateSections } from './sea-template';
import type { TemplateProps } from './types';

vi.mock('@umijs/max', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@umijs/max')>()),
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOrder: () => true,
  }),
  useModel: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: 'org-1', name: '测试组织' },
      },
    },
  }),
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
  sectionKeys = ['masterBillContent', 'houseBillContent'],
}: {
  exposeForm: (form: FormInstance) => void;
  initialValues?: Record<string, unknown>;
  sectionKeys?: string[];
}) {
  const [form] = Form.useForm();
  exposeForm(form);
  const sections = getSeaTemplateSections(props).filter((section) =>
    sectionKeys.includes(section.key),
  );
  // bookingInfo 分节内含 SeaCreateDocumentModeField，避免同名字段重复注册。
  const includesBookingInfo = sectionKeys.includes('bookingInfo');
  return (
    <App>
      <ProForm
        form={form}
        submitter={false}
        layout="vertical"
        initialValues={initialValues}
      >
        {!includesBookingInfo && <SeaCreateDocumentModeField />}
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
      'HBL 分单内容',
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
    const house = screen.getByRole('region', { name: 'HBL 分单内容' });
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

const DIRECT = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT;

function candidateResponse(overrides?: {
  members?: API.SeaMasterBillMemberSummary[];
  conflicts?: API.SeaVoyageConflict[];
}) {
  return {
    matched: true,
    candidate: {
      id: 'mbl-candidate-1',
      version: '5',
      masterNo: 'COSCOAUTO1',
      shippingLineId: 'carrier-1',
      memberCount: 2,
      members:
        overrides?.members ??
        ([
          {
            orderId: 'order-a',
            orderNo: 'SE-A',
            documentStructure: HOUSE,
          },
          {
            orderId: 'order-b',
            orderNo: 'SE-B',
            documentStructure: HOUSE,
          },
        ] satisfies API.SeaMasterBillMemberSummary[]),
      batchNormalizedHouseNos: ['HBL-EXIST-A', 'HBL-EXIST-B'],
      transportExecutions: [
        {
          id: 'te-1',
          version: '3',
          vesselName: 'EVER TEST',
          voyageNo: '001W',
        },
      ],
    },
    conflicts: overrides?.conflicts ?? [],
  } as Awaited<
    ReturnType<typeof orderService.orderServiceMatchSeaMasterBillCandidate>
  >;
}

describe('SE 新建共享主单自动关联与批次排重', () => {
  it('命中全 HOUSE 批次自动写入候选确认参数并展示自动关联横幅', async () => {
    const matchSpy = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValue(candidateResponse());
    let form!: FormInstance;
    render(
      <CreateDocuments
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingInfo', 'houseBillContent']}
        initialValues={{
          seaDocumentStructure: HOUSE,
          shippingLineId: 'carrier-1',
          seaMasterBillMasterNo: 'COSCOAUTO1',
        }}
      />,
    );

    await waitFor(() =>
      expect(screen.getByText(/已自动关联共享主单批次/)).toBeInTheDocument(),
    );
    expect(matchSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        masterNo: 'COSCOAUTO1',
        shippingLineId: 'carrier-1',
      }),
    );
    // 候选确认参数（候选 ID + 版本 + 唯一航次）自动带入提交载荷。
    expect(form.getFieldValue('seaMasterBillCandidateId')).toBe(
      'mbl-candidate-1',
    );
    expect(form.getFieldValue('seaMasterBillExpectedCandidateVersion')).toBe(
      '5',
    );
    expect(form.getFieldValue('seaMasterBillCandidateTeId')).toBe('te-1');
    const payload = buildSeaExportCreatePayload(form.getFieldsValue(true));
    expect(payload.seaMasterBill?.candidateId).toBe('mbl-candidate-1');
    expect(payload.seaMasterBill?.expectedCandidateVersion).toBe('5');
  });

  it('批次含直单成员时展示红色阻断横幅且不自动关联', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(
      candidateResponse({
        members: [
          { orderId: 'order-a', orderNo: 'SE-A', documentStructure: DIRECT },
        ],
      }),
    );
    let form!: FormInstance;
    render(
      <CreateDocuments
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingInfo', 'houseBillContent']}
        initialValues={{
          seaDocumentStructure: HOUSE,
          shippingLineId: 'carrier-1',
          seaMasterBillMasterNo: 'COSCOAUTO1',
        }}
      />,
    );

    await waitFor(() =>
      expect(
        screen.getByText('该主单已被直单订单占用，如需拼单请先将其转为分单'),
      ).toBeInTheDocument(),
    );
    await waitFor(() => {
      const payload = buildSeaExportCreatePayload(form.getFieldsValue(true));
      expect(payload.seaMasterBill?.candidateId).toBeUndefined();
    });
  });

  it('分单号与批次内已用分单号重复时即时提示', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(candidateResponse());
    let form!: FormInstance;
    render(
      <CreateDocuments
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingInfo', 'houseBillContent']}
        initialValues={{
          seaDocumentStructure: HOUSE,
          shippingLineId: 'carrier-1',
          seaMasterBillMasterNo: 'COSCOAUTO1',
        }}
      />,
    );

    // 等待候选命中写入批次分单号清单。
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillBatchHouseNos')).toEqual([
        'HBL-EXIST-A',
        'HBL-EXIST-B',
      ]),
    );
    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: ' hbl-exist-a ' },
    });
    await waitFor(() =>
      expect(
        screen.getByText(/在该主单批次内已存在（含作废）/),
      ).toBeInTheDocument(),
    );
    await act(async () => {
      await expect(
        form.validateFields([
          ['seaHouseBill', 'houseNo'],
          ['seaHouseBill', 'issuerSource'],
        ]),
      ).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({ name: ['seaHouseBill', 'houseNo'] }),
        ]),
      });
    });
  });

  it('分单号先于候选结果录入时，批次清单到达后仍即时补检重复', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(candidateResponse());
    render(
      <CreateDocuments
        exposeForm={() => {}}
        sectionKeys={['bookingInfo', 'houseBillContent']}
        initialValues={{
          seaDocumentStructure: HOUSE,
          shippingLineId: 'carrier-1',
        }}
      />,
    );

    // 候选尚未返回（批次清单未写入）时先录分单号，此时不报错。
    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: 'hbl-exist-a' },
    });
    expect(
      screen.queryByText(/在该主单批次内已存在（含作废）/),
    ).not.toBeInTheDocument();

    // 后录主单号触发候选命中写入批次清单，已录入的分单号随依赖变更自动重校验。
    fireEvent.change(screen.getByPlaceholderText('请输入主单号'), {
      target: { value: 'COSCOAUTO1' },
    });
    await waitFor(() =>
      expect(screen.getByText(/已自动关联共享主单批次/)).toBeInTheDocument(),
    );
    await waitFor(() =>
      expect(
        screen.getByText(/在该主单批次内已存在（含作废）/),
      ).toBeInTheDocument(),
    );
  });
});
