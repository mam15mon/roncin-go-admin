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
import { revealSeaFormErrors } from './sea-form-error-reveal';
import { getSeaTemplateSections } from './sea-template';
import type { TemplateProps } from './types';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOrder: () => true,
  }),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: 'org-1', name: '测试组织' },
      },
    },
  }),
}));

vi.mock('@/services/roncin/seaDocumentService', () => ({
  seaDocumentServiceGetSeaOrderDocuments: vi.fn(),
  seaDocumentServicePreviewChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceExecuteChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceUpdateSeaHouseBill: vi.fn(),
  seaDocumentServiceUpdateSeaMasterBillContent: vi.fn(),
}));

vi.mock('@/services/roncin/orderReleasePodService', () => ({
  orderReleasePodServiceListReleasePods: vi
    .fn()
    .mockResolvedValue({ data: [] }),
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

const HOUSE = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE;
const DIRECT = SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT;

/** 当前前台提单页签正文区域（页签强制渲染，仅激活正文带 active 类）。 */
function activeBillPane() {
  const pane = document.querySelector<HTMLElement>('.ant-tabs-content-active');
  if (!pane) throw new Error('提单页签正文未渲染');
  return pane;
}

function SectionCards({
  exposeForm,
  initialValues,
  sectionKeys,
  isDetail = false,
}: {
  exposeForm: (form: FormInstance) => void;
  initialValues?: Record<string, unknown>;
  sectionKeys?: string[];
  isDetail?: boolean;
}) {
  const [form] = Form.useForm();
  exposeForm(form);
  const sections = getSeaTemplateSections({ ...props, isDetail }).filter(
    (section) => !sectionKeys || sectionKeys.includes(section.key),
  );
  return (
    <App>
      <ProForm
        form={form}
        submitter={false}
        layout="vertical"
        initialValues={initialValues}
      >
        {sections.map((section) => (
          <section key={section.key} aria-label={section.title}>
            {section.content}
          </section>
        ))}
      </ProForm>
    </App>
  );
}

describe('SE 新建与详情五卡片合同', () => {
  it('新建与详情返回同一套五卡片 key、标题与顺序', () => {
    const expected = [
      { key: 'businessCustomer', title: '业务与客户' },
      { key: 'bookingTransport', title: '订舱与运输' },
      { key: 'cargoCommercial', title: '货物与商业' },
      { key: 'seaDocument', title: '提单信息' },
      { key: 'internalPersonnel', title: '内部人员' },
    ];
    expect(
      getSeaTemplateSections(props).map(({ key, title }) => ({ key, title })),
    ).toEqual(expected);
    expect(
      getSeaTemplateSections({ ...props, isDetail: true }).map(
        ({ key, title }) => ({ key, title }),
      ),
    ).toEqual(expected);
  });

  it('第二卡内部按订舱与主单、航线与船期、箱量与截关、操作补充编排', () => {
    let _form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          _form = value;
        }}
        sectionKeys={['bookingTransport']}
      />,
    );
    const bookingSection = screen.getByRole('region', {
      name: '订舱与运输',
    });
    expect(bookingSection).toHaveTextContent('订舱与主单');
    expect(bookingSection).toHaveTextContent('航线与船期');
    expect(bookingSection).toHaveTextContent('箱量与截关');
    expect(bookingSection).toHaveTextContent('操作补充');
    // 订舱号、船公司、MBL 主单号、提单模式同在第一小节。
    expect(bookingSection).toHaveTextContent('订舱号');
    expect(bookingSection).toHaveTextContent('船公司');
    expect(bookingSection).toHaveTextContent('MBL 主单号');
    expect(bookingSection).toHaveTextContent('提单模式');
  });

  it('委托件重尺只出现在货物与商业卡，提单页签只保留实际侧', () => {
    let _form!: FormInstance;
    const { container } = render(
      <SectionCards
        exposeForm={(value) => {
          _form = value;
        }}
        sectionKeys={['cargoCommercial', 'seaDocument']}
        initialValues={{ seaDocumentStructure: HOUSE }}
      />,
    );
    for (const id of [
      'totalPackages',
      'totalPackageUnit',
      'totalGrossWeightKg',
      'totalVolumeCbm',
    ]) {
      expect(container.querySelectorAll(`[id="${id}"]`)).toHaveLength(1);
    }
    const cargoSection = screen.getByRole('region', { name: '货物与商业' });
    expect(cargoSection).toHaveTextContent('委托件数');
    expect(cargoSection).toHaveTextContent('委托毛重');
    expect(cargoSection).toHaveTextContent('委托体积');
    // 提单卡 MBL 页签展示实际件重尺标题，不再出现委托对照列。
    expect(screen.getByText('MBL 实际件重尺')).toBeVisible();
    expect(screen.queryByText('委托 vs 实际件重尺对照')).toBeNull();
  });

  it('散杂托运在货物与商业卡展示散杂计费吨提示', () => {
    render(
      <SectionCards
        exposeForm={() => {}}
        sectionKeys={[
          'businessCustomer',
          'cargoCommercial',
          'bookingTransport',
        ]}
        initialValues={{
          shipmentType: 3,
          totalGrossWeightKg: 2500,
          totalVolumeCbm: 1.8,
          containerRequests: [],
        }}
      />,
    );
    expect(screen.getByText('散杂计费吨 (RT)：2.500')).toBeVisible();
    expect(
      screen.getByText('散杂货不使用箱型箱量、箱号或封号配置'),
    ).toBeVisible();
  });
});

describe('SE 提单卡 MBL/HBL 页签', () => {
  it('HOUSE 一次只展示一个正文，切换页签不丢已填数据', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['seaDocument']}
        initialValues={{ seaDocumentStructure: HOUSE }}
      />,
    );

    // 默认展示 MBL 正文，HBL 正文隐藏但页签存在（HBL 自身也有发货人字段，
    // 以字段 id 断言正文归属）。
    expect(
      activeBillPane().querySelector('#seaMasterBillContent_shipperText'),
    ).not.toBeNull();
    expect(activeBillPane().querySelector('#seaHouseBill_houseNo')).toBeNull();

    fireEvent.change(
      within(activeBillPane()).getByPlaceholderText(
        '请输入发货人英文名称与详细地址',
      ),
      { target: { value: 'SHIPPER TEXT' } },
    );
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() =>
      expect(
        activeBillPane().querySelector('#seaHouseBill_houseNo'),
      ).not.toBeNull(),
    );
    expect(
      activeBillPane().querySelector('#seaMasterBillContent_shipperText'),
    ).toBeNull();
    fireEvent.change(
      within(activeBillPane()).getByPlaceholderText('请输入分单号'),
      {
        target: { value: 'HBL001' },
      },
    );

    // 切回 MBL：两侧草稿均保留。
    fireEvent.click(screen.getByRole('tab', { name: /主单 \(MBL\)/ }));
    await waitFor(() =>
      expect(
        activeBillPane().querySelector('#seaMasterBillContent_shipperText'),
      ).not.toBeNull(),
    );
    expect(form.getFieldValue(['seaMasterBillContent', 'shipperText'])).toBe(
      'SHIPPER TEXT',
    );
    expect(form.getFieldValue(['seaHouseBill', 'houseNo'])).toBe('HBL001');
  });

  it('HOUSE 必填在从未打开 HBL 页签时同样拦截提交', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['seaDocument']}
        initialValues={{ seaDocumentStructure: HOUSE }}
      />,
    );
    expect(screen.queryByPlaceholderText('请输入分单号')).not.toBeVisible();

    await act(async () => {
      await expect(
        form.validateFields([
          ['seaHouseBill', 'houseNo'],
          ['seaHouseBill', 'issuerSource'],
        ]),
      ).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({ name: ['seaHouseBill', 'houseNo'] }),
          expect.objectContaining({ name: ['seaHouseBill', 'issuerSource'] }),
        ]),
      });
    });
  });

  it('DIRECT 不出现 HBL 页签、不校验不提交隐藏草稿，切回 HOUSE 恢复已填值', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport', 'seaDocument']}
        initialValues={{ seaDocumentStructure: HOUSE }}
      />,
    );

    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    const houseNoInput = await waitFor(() =>
      within(activeBillPane()).getByPlaceholderText('请输入分单号'),
    );
    fireEvent.change(houseNoInput, { target: { value: 'HBL-KEEP' } });
    fireEvent.click(
      screen.getByRole('radio', { name: '仅船公司主单（DIRECT）' }),
    );
    await waitFor(() =>
      expect(
        screen.queryByRole('tab', { name: /分单 \(HBL\)/ }),
      ).not.toBeInTheDocument(),
    );
    // 隐藏草稿保留在 Form store，但提交口径不含 HBL。
    expect(form.getFieldValue(['seaHouseBill', 'houseNo'])).toBe('HBL-KEEP');
    await act(async () => {
      await expect(
        form.validateFields([
          ['seaHouseBill', 'houseNo'],
          ['seaHouseBill', 'issuerSource'],
        ]),
      ).resolves.toBeDefined();
    });
    const payload = buildSeaExportCreatePayload(form.getFieldsValue(true));
    expect(payload.seaDocument?.houseBill).toBeUndefined();

    // 切回 HOUSE：HBL 页签恢复，草稿值原样保留。
    fireEvent.click(screen.getByRole('radio', { name: '有货代分单（HOUSE）' }));
    fireEvent.click(await screen.findByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() =>
      expect(
        within(activeBillPane()).getByPlaceholderText('请输入分单号'),
      ).toBeVisible(),
    );
    expect(form.getFieldValue(['seaHouseBill', 'houseNo'])).toBe('HBL-KEEP');
    const housePayload = buildSeaExportCreatePayload(form.getFieldsValue(true));
    expect(housePayload.seaDocument?.houseBill?.houseNo).toBe('HBL-KEEP');
  });

  it('带入委托件重尺只改当前提单实际值并保持提交口径', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['cargoCommercial', 'seaDocument']}
        initialValues={{
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
    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    await waitFor(() =>
      expect(screen.getByPlaceholderText('请输入分单号')).toBeVisible(),
    );
    const house = screen.getByRole('region', { name: '提单信息' });
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
});

describe('SE 就近折叠备注', () => {
  it('空备注初始收起、有内容初始展开，收起显示有备注且值保留', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport']}
        initialValues={{ bookingNotes: '已有订舱备注', operationNotes: '' }}
      />,
    );

    const bookingNotes = screen.getByTestId('sea-notes-bookingNotes');
    expect(
      within(bookingNotes).getByPlaceholderText('请输入订舱备注'),
    ).toBeVisible();
    // 空备注初始收起：输入框挂载但不可见，值不丢失、可展开编辑。
    const operationNotes = screen.getByTestId('sea-notes-operationNotes');
    expect(
      within(operationNotes).getByPlaceholderText('请输入操作备注'),
    ).not.toBeVisible();

    fireEvent.click(within(bookingNotes).getByRole('button', { name: '收起' }));
    expect(within(bookingNotes).getByText('有备注')).toBeVisible();
    expect(
      within(bookingNotes).queryByPlaceholderText('请输入订舱备注'),
    ).not.toBeVisible();
    expect(form.getFieldValue('bookingNotes')).toBe('已有订舱备注');

    fireEvent.click(within(bookingNotes).getByRole('button', { name: '展开' }));
    expect(
      within(bookingNotes).getByPlaceholderText('请输入订舱备注'),
    ).toBeVisible();
    // 手动展开空备注后可直接录入。
    fireEvent.click(
      within(operationNotes).getByRole('button', { name: '展开' }),
    );
    fireEvent.change(
      within(operationNotes).getByPlaceholderText('请输入操作备注'),
      { target: { value: '新操作备注' } },
    );
    expect(form.getFieldValue('operationNotes')).toBe('新操作备注');
  });

  it('异步回填到达后自动展开；用户手动收起后数据更新不强制重新展开', async () => {
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport']}
      />,
    );
    const allocationNotes = screen.getByTestId('sea-notes-allocationNotes');
    expect(
      within(allocationNotes).getByPlaceholderText('请输入配舱备注'),
    ).not.toBeVisible();

    // 模拟详情回填/草稿恢复：数据到达后有内容自动展开。
    act(() => {
      form.setFieldsValue({ allocationNotes: '回填的配舱备注' });
    });
    await waitFor(() =>
      expect(
        within(allocationNotes).getByPlaceholderText('请输入配舱备注'),
      ).toBeVisible(),
    );

    // 用户手动收起后，普通重绘（再次回填同值）不强制重新展开。
    fireEvent.click(
      within(allocationNotes).getByRole('button', { name: '收起' }),
    );
    act(() => {
      form.setFieldsValue({ allocationNotes: '回填的配舱备注-新' });
    });
    expect(
      within(allocationNotes).queryByPlaceholderText('请输入配舱备注'),
    ).not.toBeVisible();
    expect(within(allocationNotes).getByText('有备注')).toBeVisible();
    expect(form.getFieldValue('allocationNotes')).toBe('回填的配舱备注-新');
  });

  it('校验失败定位链路按首个错误展开折叠备注', async () => {
    render(
      <SectionCards
        exposeForm={() => {}}
        sectionKeys={['bookingTransport']}
        initialValues={{ bookingNotes: '已有订舱备注' }}
      />,
    );
    const bookingNotes = screen.getByTestId('sea-notes-bookingNotes');
    fireEvent.click(within(bookingNotes).getByRole('button', { name: '收起' }));
    expect(
      within(bookingNotes).queryByPlaceholderText('请输入订舱备注'),
    ).not.toBeVisible();

    act(() => {
      revealSeaFormErrors([{ name: ['bookingNotes'], errors: ['校验失败'] }]);
    });
    await waitFor(() =>
      expect(
        within(bookingNotes).getByPlaceholderText('请输入订舱备注'),
      ).toBeVisible(),
    );
  });
});

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
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport']}
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
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport']}
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
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport', 'seaDocument']}
        initialValues={{
          seaDocumentStructure: HOUSE,
          shippingLineId: 'carrier-1',
          seaMasterBillMasterNo: 'COSCOAUTO1',
        }}
      />,
    );

    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    // 等待候选命中写入批次分单号清单。
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillBatchHouseNos')).toEqual([
        'HBL-EXIST-A',
        'HBL-EXIST-B',
      ]),
    );
    fireEvent.change(await screen.findByPlaceholderText('请输入分单号'), {
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
});
