import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import {
  act,
  fireEvent,
  screen,
  waitFor,
  within,
} from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { getAirTemplateSections } from './air-template';
import type { TemplateProps } from './types';

vi.mock('@/services/roncin/masterDataService', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/services/roncin/masterDataService')
  >()),
  masterDataServiceListAirlines: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@/services/roncin/airExportService', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/services/roncin/airExportService')
  >()),
  airExportServiceGetAirMasterWaybill: vi.fn().mockResolvedValue(undefined),
  airExportServiceListAirMasterWaybills: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

const search = async () => [];
const props: TemplateProps = {
  businessType: OrderBusinessType.BUSINESS_TYPE_AE,
  orderKind: 'air-export',
  cargoCategoryOptions: [],
  locationOptions: [],
  searchLocations: search,
  currencyOptions: [],
  containerSpecOptions: [],
  searchCustomers: search,
  searchShippingLines: search,
  searchBookingAgents: search,
  searchForeignAgents: search,
  searchShippingAgents: search,
  setCustomerCode: () => {},
  personnelOptions: [],
};

const user = {
  id: 'user-1',
  currentOrganization: { id: 'org-1', name: '测试组织' },
};

function SectionCards({
  initialValues,
  isDetail = false,
  cargoCategoryOptions = props.cargoCategoryOptions,
  exposeForm,
}: {
  initialValues?: Record<string, unknown>;
  isDetail?: boolean;
  cargoCategoryOptions?: TemplateProps['cargoCategoryOptions'];
  exposeForm?: (form: FormInstance) => void;
}) {
  const [form] = Form.useForm();
  exposeForm?.(form);
  const sections = getAirTemplateSections({
    ...props,
    cargoCategoryOptions,
    isDetail,
  });
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
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
    </ConfigProvider>
  );
}

// 挂载期 request 型下拉会发起异步选项查询；在 act 内冲刷微任务，
// 确保用例结束前已触发的异步流全部落地，避免迟到 setState 触发 act 警告。
async function flushMountRequests() {
  await act(async () => {
    await Promise.resolve();
    await Promise.resolve();
    await Promise.resolve();
  });
}

describe('AE 新建与详情五卡片合同', () => {
  it('新建与详情返回同一套五卡片 key、标题与顺序（对齐海运出口节奏）', () => {
    const expected = [
      { key: 'businessCustomer', title: '业务与客户' },
      { key: 'bookingTransport', title: '订舱与运输' },
      { key: 'cargoCommercial', title: '货物与商业' },
      { key: 'airDocuments', title: '单证信息' },
      { key: 'internalPersonnel', title: '内部人员' },
    ];
    expect(
      getAirTemplateSections(props).map(({ key, title }) => ({ key, title })),
    ).toEqual(expected);
    expect(
      getAirTemplateSections({ ...props, isDetail: true }).map(
        ({ key, title }) => ({ key, title }),
      ),
    ).toEqual(expected);
  });

  it('订舱与运输卡按小节分组编排：主单与订舱 / 航线与班期 / 时刻与申报 / 备注', async () => {
    renderWithApp(<SectionCards />, { user });
    const booking = screen.getByRole('region', { name: '订舱与运输' });
    // textContent 顺序即 DOM 顺序：三个小节标题按固定顺序出现。
    const groupIndexes = ['主单与订舱', '航线与班期', '时刻与申报', '备注'].map(
      (title) => booking.textContent?.indexOf(title) ?? -1,
    );
    expect(groupIndexes.every((index) => index >= 0)).toBe(true);
    expect([...groupIndexes].sort((a, b) => a - b)).toEqual(groupIndexes);
    // 主单备注与操作备注为可折叠备注（NEW 模式两者都挂载，主单备注在前）。
    const masterNotes = within(booking).getByTestId(
      'sea-notes-airMaster.notes',
    );
    const operationNotes = within(booking).getByTestId(
      'sea-notes-operationNotes',
    );
    expect(masterNotes).toBeInTheDocument();
    expect(operationNotes).toBeInTheDocument();
    expect(
      masterNotes.compareDocumentPosition(operationNotes) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    // 与海运同节奏：订舱号、订舱/国外代理归入主单小节，四类本票备注集中在「备注」小节。
    for (const label of [
      '订舱号',
      '订舱代理',
      '国外代理',
      '主运单号（MAWB）',
      '航班号',
    ]) {
      expect(booking).toHaveTextContent(label);
    }
    const noteIds = [
      'bookingNotes',
      'allocationNotes',
      'operationNotes',
      'notes',
    ];
    const noteNodes = noteIds.map((id) =>
      within(booking).getByTestId(`sea-notes-${id}`),
    );
    noteNodes.slice(1).forEach((node, index) => {
      expect(
        noteNodes[index].compareDocumentPosition(node) &
          Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy();
    });
    await flushMountRequests();
  });

  it('业务与客户卡不再包含订舱号，分单信息卡承接 HAWB 与单证资料', async () => {
    renderWithApp(<SectionCards />, { user });
    const business = screen.getByRole('region', { name: '业务与客户' });
    expect(business).toHaveTextContent('订单编号');
    expect(business).toHaveTextContent('委托单位');
    expect(business).not.toHaveTextContent('订舱号');
    expect(business).not.toHaveTextContent('本票分运单号');

    const documents = screen.getByRole('region', { name: '单证信息' });
    expect(documents).toHaveTextContent('本票分运单号（HAWB）');
    expect(documents).toHaveTextContent('本票分单状态');
    expect(documents).toHaveTextContent('本票单证资料');
    // 详情已签发才展示的修改原因在新建模式不出现。
    expect(documents).not.toHaveTextContent('已签发分单修改原因');
    await flushMountRequests();
  });

  it('详情模式展示共享主单摘要与订舱号，主单编辑字段收起', async () => {
    renderWithApp(
      <SectionCards
        isDetail
        initialValues={{
          airExport: {
            documentMode: 'HOUSE',
            house: { status: 'RELEASED' },
            master: {
              id: 'master-1',
              airlineName: '测试航空',
              content: { masterNo: '78112345675' },
            },
          },
        }}
      />,
      { user },
    );
    const booking = screen.getByRole('region', { name: '订舱与运输' });
    expect(booking).toHaveTextContent('共享主单 78112345675');
    expect(booking).toHaveTextContent('订舱号');
    expect(booking).not.toHaveTextContent('主单关联方式');
    expect(booking).toHaveTextContent('主运单号（MAWB）');
    expect(booking).toHaveTextContent('共同航线与班期');

    const documents = screen.getByRole('region', { name: '单证信息' });
    expect(documents).toHaveTextContent('已签发分单修改原因');
    await flushMountRequests();
  });

  it('本票货物卡在品类与货物描述之间渲染危险品联动区块，勾选危险品后展开', async () => {
    let form!: FormInstance;
    renderWithApp(
      <SectionCards
        cargoCategoryOptions={[
          { label: '危险品', value: 'cat-dg', code: 'DANGEROUS' },
        ]}
        exposeForm={(instance) => {
          form = instance;
        }}
      />,
      { user },
    );
    const cargo = screen.getByRole('region', { name: '货物与商业' });
    // 渲染初始即收起（未勾选任何品类）。
    let motion = within(cargo).getByTestId('sea-dangerous-goods-motion');
    expect(motion).toHaveAttribute('data-open', 'false');
    expect(motion).toHaveAttribute('aria-hidden', 'true');
    // 顺序即模板编排：货物品类 → 危险品联动 → 货物描述。
    const categoryLabel = within(cargo).getByText('货物品类');
    const descriptionLabel = within(cargo).getByText(/委托品名/);
    expect(
      categoryLabel.compareDocumentPosition(motion) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    expect(
      motion.compareDocumentPosition(descriptionLabel) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    // 勾选危险品品类后联动展开：options 接线正确（误传服务类型等候选时用例失败）。
    await act(async () => {
      form.setFieldsValue({ cargoCategoryIds: ['cat-dg'] });
    });
    await flushMountRequests();
    motion = within(cargo).getByTestId('sea-dangerous-goods-motion');
    expect(motion).toHaveAttribute('data-open', 'true');
    expect(motion).toHaveAttribute('aria-hidden', 'false');
    // 空运动效区比海运多第四字段：包装类别（IATA DGR PG）。
    await waitFor(() => {
      expect(within(cargo).getByText('包装类别')).toBeVisible();
    });
    expect(
      within(cargo).getByRole('combobox', { name: /包装类别/ }),
    ).toBeInTheDocument();
  });

  it('新建页只有主运单号入口，不再要求先选新建或关联主单', async () => {
    renderWithApp(<SectionCards />, { user });
    const booking = screen.getByRole('region', { name: '订舱与运输' });
    expect(booking).not.toHaveTextContent('主单关联方式');
    expect(booking).not.toHaveTextContent('选择共享主单');
    expect(
      within(booking).getByRole('textbox', { name: /主运单号（MAWB）/ }),
    ).toBeEnabled();
    expect(booking).toHaveTextContent('航线与班期');
    expect(booking).toHaveTextContent('时刻与申报');
    expect(
      within(booking).getByTestId('sea-notes-airMaster.notes'),
    ).toBeInTheDocument();
    await flushMountRequests();
  });
  it('经营字段在本票卡片中可编辑，付款方式只注册一次且不写入主单', async () => {
    let form!: FormInstance;
    renderWithApp(
      <SectionCards
        initialValues={{
          paymentTerm: 2,
          factoryName: '工厂A',
          contractNo: '合同A',
          cargoCurrency: 'USD',
          insuranceCurrency: 'CNY',
        }}
        exposeForm={(value) => {
          form = value;
        }}
      />,
      { user },
    );
    const business = screen.getByRole('region', { name: '业务与客户' });
    const cargo = screen.getByRole('region', { name: '货物与商业' });
    // 与海运同归属：业务卡只留客户、单号与集运/跨境；经营条款与金额随货物卡，备注不在货物卡。
    expect(business).toHaveTextContent('集运/跨境');
    for (const label of ['贸易条款', '业务付款方式', '合约号', '订舱代理']) {
      expect(business).not.toHaveTextContent(label);
    }
    for (const label of [
      '贸易条款',
      '业务付款方式',
      '合约号',
      '货值',
      '保费',
      '工厂',
      '接单时间',
      '货好时间',
    ]) {
      expect(cargo).toHaveTextContent(label);
    }
    for (const label of ['订舱备注', '配舱备注', '普通备注']) {
      expect(cargo).not.toHaveTextContent(label);
    }
    expect(
      screen.getAllByRole('combobox', { name: '业务付款方式' }),
    ).toHaveLength(1);
    expect(screen.getByRole('textbox', { name: '工厂' })).toHaveValue('工厂A');
    fireEvent.change(screen.getByRole('textbox', { name: '工厂' }), {
      target: { value: '' },
    });
    fireEvent.change(screen.getByRole('textbox', { name: '合约号' }), {
      target: { value: '合同B' },
    });
    expect(form.getFieldValue('factoryName')).toBe('');
    expect(form.getFieldValue('contractNo')).toBe('合同B');
    expect(form.getFieldValue('paymentTerm')).toBe(2);
    expect(form.getFieldValue(['airMaster', 'contractNo'])).toBeUndefined();
    const amountFields = [
      'cargoValue',
      'cargoCurrency',
      'insurancePremium',
      'insuranceCurrency',
    ];
    await act(async () => {
      await expect(form.validateFields(amountFields)).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({
            name: ['cargoValue'],
            errors: ['请输入货值'],
          }),
          expect.objectContaining({
            name: ['insurancePremium'],
            errors: ['请输入保费'],
          }),
        ]),
      });
      form.setFieldsValue({ cargoValue: '0', insurancePremium: '123.4567' });
      await expect(form.validateFields(amountFields)).resolves.toMatchObject({
        cargoValue: '0',
        cargoCurrency: 'USD',
        insurancePremium: '123.4567',
        insuranceCurrency: 'CNY',
      });
      form.setFieldsValue({
        cargoValue: '',
        cargoCurrency: undefined,
        insurancePremium: '',
        insuranceCurrency: undefined,
      });
      await expect(form.validateFields(amountFields)).resolves.toMatchObject({
        cargoValue: '',
        cargoCurrency: undefined,
        insurancePremium: '',
        insuranceCurrency: undefined,
      });
    });
    await flushMountRequests();
  });
});
