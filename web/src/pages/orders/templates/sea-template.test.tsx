import { ProForm, ProFormText } from '@ant-design/pro-components';
import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import * as orderService from '@/services/roncin/orderService';
import {
  SeaAssociatedHouseBillsField,
  SeaMasterBillFields,
  splitSeaVesselVoyage,
} from './components/sea/SeaTransportSection';
import { getSeaTemplateSections } from './sea-template';

vi.mock('@umijs/max', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@umijs/max')>()),
  useAccess: () => ({ canOrder: () => true }),
}));

describe('海运订单新增模板', () => {
  it('候选查询按后端相同规则拆分船名航次', () => {
    expect(splitSeaVesselVoyage('EVER GIVEN / 001W')).toEqual({
      vesselName: 'EVER GIVEN',
      voyageNo: '001W',
    });
    expect(splitSeaVesselVoyage('EVER GIVEN 001W')).toEqual({
      vesselName: 'EVER GIVEN',
      voyageNo: '001W',
    });
    expect(splitSeaVesselVoyage('EVERGIVEN')).toEqual({
      vesselName: 'EVERGIVEN',
      voyageNo: undefined,
    });
  });

  it('候选匹配使用 ShippingLine 作为唯一船公司身份', async () => {
    const matchCandidate = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValue({ matched: false });

    render(
      <ProForm
        submitter={false}
        initialValues={{
          shippingLineId: 'carrier-1',
          seaMasterBillMasterNo: 'COSCO123456',
        }}
      >
        <ProFormText name="shippingLineId" hidden />
        <SeaMasterBillFields />
      </ProForm>,
    );

    await waitFor(
      () => {
        expect(matchCandidate).toHaveBeenCalledWith(
          expect.objectContaining({
            masterNo: 'COSCO123456',
            shippingLineId: 'carrier-1',
          }),
        );
      },
      { timeout: 2_000 },
    );
  });

  it('共享主单关联多票时回填当前船公司名称并同时禁止修改船公司和 MBL 主单号', async () => {
    const sections = getSeaTemplateSections({
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      searchLocations: vi.fn().mockResolvedValue([]),
      currencyOptions: [],
      containerSpecOptions: [],
      searchCustomers: vi.fn().mockResolvedValue([]),
      searchShippingLines: vi.fn().mockResolvedValue([]),
      searchBookingAgents: vi.fn().mockResolvedValue([]),
      searchForeignAgents: vi.fn().mockResolvedValue([]),
      searchShippingAgents: vi.fn().mockResolvedValue([]),
      setCustomerCode: vi.fn(),
      checkCustomerReferenceNo: vi.fn().mockResolvedValue(undefined),
      checkInternalReferenceNo: vi.fn().mockResolvedValue(undefined),
      personnelOptions: [],
      isDetail: true,
    });

    render(
      <ProForm
        submitter={false}
        initialValues={{
          shippingLineId: 'carrier-1',
          seaMasterBillMasterNo: 'COSCO123456',
          seaMasterBill: {
            masterNo: 'COSCO123456',
            shippingLineId: 'carrier-1',
            shippingLineName: '中远海运 / COSCO SHIPPING (COSU)',
            memberCount: 2,
          },
        }}
      >
        {sections.map((section) => (
          <div key={section.key}>{section.content}</div>
        ))}
      </ProForm>,
    );

    const carrierItem = screen.getByText('船公司').closest('.ant-form-item');
    expect(carrierItem?.querySelector('input')).toBeDisabled();
    await waitFor(() => {
      expect(
        screen.getByText('中远海运 / COSCO SHIPPING (COSU)'),
      ).toBeInTheDocument();
    });
    expect(
      screen.getByPlaceholderText('请输入主单号 (仅大写字母与数字)'),
    ).toBeDisabled();
  });

  it('按配舱、提单、货物顺序生成海运业务区块', () => {
    const sections = getSeaTemplateSections({
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      searchLocations: vi.fn().mockResolvedValue([]),
      currencyOptions: [],
      containerSpecOptions: [{ label: '20GP', value: 'spec-20gp' }],
      searchCustomers: vi.fn().mockResolvedValue([]),
      searchShippingLines: vi.fn().mockResolvedValue([]),
      searchBookingAgents: vi.fn().mockResolvedValue([]),
      searchForeignAgents: vi.fn().mockResolvedValue([]),
      searchShippingAgents: vi.fn().mockResolvedValue([]),
      setCustomerCode: vi.fn(),
      checkCustomerReferenceNo: vi.fn().mockResolvedValue(undefined),
      checkInternalReferenceNo: vi.fn().mockResolvedValue(undefined),
      personnelOptions: [],
    });

    expect(sections.map(({ key, title }) => ({ key, title }))).toEqual([
      { key: 'basicInfo', title: '业务信息' },
      { key: 'transportInfo', title: '配舱信息' },
      { key: 'sea-document', title: '提单信息' },
      { key: 'cargoInfo', title: '货物信息' },
      { key: 'remarks', title: '备注' },
      { key: 'internalInfo', title: '内部信息' },
    ]);

    render(
      <ProForm submitter={false}>
        {sections.map((section) => (
          <div key={section.key} data-testid={`section-${section.key}`}>
            {section.content}
          </div>
        ))}
      </ProForm>,
    );
    const transportSection = screen.getByTestId('section-transportInfo');
    expect(transportSection).toHaveTextContent('MBL 主单号');
    expect(transportSection).not.toHaveTextContent('实际签发/承运主体');
    expect(transportSection).not.toHaveTextContent('主单签发方');
    expect(transportSection).not.toHaveTextContent('分单信息 (HBL)');
    expect(transportSection).toHaveTextContent('计划箱型箱量');
    expect(screen.getByRole('button', { name: /添加首张分单/ })).toBeTruthy();
    expect(
      screen.getByRole('button', { name: /新增计划箱型箱量/ }),
    ).toBeTruthy();

    const cargoSection = screen.getByTestId('section-cargoInfo');
    expect(cargoSection).not.toHaveTextContent('主单号');
    expect(cargoSection).not.toHaveTextContent('分单号');

    const carrierLabel = screen.getByText('船公司').closest('label');
    expect(carrierLabel).toHaveClass('ant-form-item-required');
  });

  it('从未确定状态添加首张及多张分单', async () => {
    const sections = getSeaTemplateSections({
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      searchLocations: vi.fn().mockResolvedValue([]),
      currencyOptions: [],
      containerSpecOptions: [],
      searchCustomers: vi.fn().mockResolvedValue([]),
      searchShippingLines: vi.fn().mockResolvedValue([]),
      searchBookingAgents: vi.fn().mockResolvedValue([]),
      searchForeignAgents: vi.fn().mockResolvedValue([]),
      searchShippingAgents: vi.fn().mockResolvedValue([]),
      setCustomerCode: vi.fn(),
      checkCustomerReferenceNo: vi.fn().mockResolvedValue(undefined),
      checkInternalReferenceNo: vi.fn().mockResolvedValue(undefined),
      personnelOptions: [],
    });

    render(
      <ProForm submitter={false}>
        {sections.map((section) => (
          <div key={section.key} data-testid={`section-${section.key}`}>
            {section.content}
          </div>
        ))}
      </ProForm>,
    );

    expect(screen.queryAllByPlaceholderText('请输入分单号')).toHaveLength(0);

    const addFirstHouseBtn = screen.getByRole('button', {
      name: /添加首张分单/,
    });
    fireEvent.click(addFirstHouseBtn);

    await waitFor(() => {
      expect(screen.getAllByPlaceholderText('请输入分单号')).toHaveLength(1);
      expect(screen.getAllByText('签发主体')).toHaveLength(1);
    });

    const addHouseBtn = screen.getByRole('button', {
      name: /添加分单 \(HBL\)/,
    });
    fireEvent.click(addHouseBtn);

    await waitFor(() => {
      expect(screen.getAllByPlaceholderText('请输入分单号')).toHaveLength(2);
    });
  }, 30_000);

  it('散杂托运隐藏箱型箱量并要求显式清理已有计划', () => {
    const sections = getSeaTemplateSections({
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      searchLocations: vi.fn().mockResolvedValue([]),
      currencyOptions: [],
      containerSpecOptions: [{ label: '40HQ', value: 'spec-40hq' }],
      searchCustomers: vi.fn().mockResolvedValue([]),
      searchShippingLines: vi.fn().mockResolvedValue([]),
      searchBookingAgents: vi.fn().mockResolvedValue([]),
      searchForeignAgents: vi.fn().mockResolvedValue([]),
      searchShippingAgents: vi.fn().mockResolvedValue([]),
      setCustomerCode: vi.fn(),
      checkCustomerReferenceNo: vi.fn().mockResolvedValue(undefined),
      checkInternalReferenceNo: vi.fn().mockResolvedValue(undefined),
      personnelOptions: [],
    });

    render(
      <ProForm
        submitter={false}
        initialValues={{
          shipmentType: 3,
          totalGrossWeightKg: 2500,
          totalVolumeCbm: 1.8,
          containerRequests: [{ containerSpecId: 'spec-40hq', quantity: 1 }],
        }}
      >
        {sections.map((section) => (
          <div key={section.key}>{section.content}</div>
        ))}
      </ProForm>,
    );

    expect(screen.queryByText('计划箱型箱量')).toBeNull();
    expect(
      screen.getByText('散杂货不使用箱型箱量、箱号或封号配置'),
    ).toBeTruthy();
    expect(screen.getByRole('button', { name: '清空箱量计划' })).toBeTruthy();
    expect(screen.getByText('散杂计费吨 (RT)：2.500')).toBeTruthy();
  });

  it('渲染货值与保费的金额及币种选择框', () => {
    const sections = getSeaTemplateSections({
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      searchLocations: vi.fn().mockResolvedValue([]),
      currencyOptions: [
        { label: 'CNY - 人民币', value: 'CNY' },
        { label: 'USD - 美元', value: 'USD' },
      ],
      containerSpecOptions: [],
      searchCustomers: vi.fn().mockResolvedValue([]),
      searchShippingLines: vi.fn().mockResolvedValue([]),
      searchBookingAgents: vi.fn().mockResolvedValue([]),
      searchForeignAgents: vi.fn().mockResolvedValue([]),
      searchShippingAgents: vi.fn().mockResolvedValue([]),
      setCustomerCode: vi.fn(),
      checkCustomerReferenceNo: vi.fn().mockResolvedValue(undefined),
      checkInternalReferenceNo: vi.fn().mockResolvedValue(undefined),
      personnelOptions: [],
    });

    const basicInfo = sections.find((s) => s.key === 'basicInfo');
    render(
      <ProForm submitter={false}>
        <div data-testid="section-basicInfo">{basicInfo?.content}</div>
      </ProForm>,
    );

    const cargoInputs = screen.getAllByPlaceholderText('金额');
    expect(cargoInputs.length).toBe(2);

    const currencySelects = screen.getAllByRole('combobox');
    expect(currencySelects.length).toBeGreaterThanOrEqual(2);
  });

  describe('配舱信息关联分单号只读展示', () => {
    it('尚未建立分单时显示“暂未录入分单号”', () => {
      render(
        <ProForm
          submitter={false}
          initialValues={{
            seaDocumentStructure:
              SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
          }}
        >
          <SeaAssociatedHouseBillsField />
        </ProForm>,
      );

      const display = screen.getByTestId('associated-hbl-display');
      expect(display.textContent).toContain('暂未录入分单号');
      expect(screen.queryByRole('textbox')).toBeNull();
    });

    it('直单模式下显示“直单，无HBL”', () => {
      render(
        <ProForm
          submitter={false}
          initialValues={{
            seaDocumentStructure:
              SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
          }}
        >
          <SeaAssociatedHouseBillsField />
        </ProForm>,
      );

      const display = screen.getByTestId('associated-hbl-display');
      expect(display.textContent).toContain('直单，无HBL');
      expect(screen.queryByRole('textbox')).toBeNull();
    });

    it('录入当前订单的唯一分单时以标签形式展示且不可就地编辑', () => {
      render(
        <ProForm
          submitter={false}
          initialValues={{
            seaDocumentStructure:
              SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
            seaHouseBill: { houseNo: 'HBL-001' },
          }}
        >
          <SeaAssociatedHouseBillsField />
        </ProForm>,
      );

      expect(screen.getByText('HBL-001')).toBeTruthy();
      expect(screen.queryByRole('textbox')).toBeNull();
    });

    it('回显已有订单的 seaDocumentSummary 分单号', () => {
      render(
        <ProForm
          submitter={false}
          initialValues={{
            seaDocumentSummary: {
              documentStructure:
                SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
              houseNo: 'HBL-HIST-1',
            },
          }}
        >
          <SeaAssociatedHouseBillsField />
        </ProForm>,
      );

      expect(screen.getByText('HBL-HIST-1')).toBeTruthy();
      expect(screen.queryByRole('textbox')).toBeNull();
    });
  });
});
