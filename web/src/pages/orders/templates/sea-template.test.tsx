import { ProForm } from '@ant-design/pro-components';
import { fireEvent, render, screen } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import { getSeaTemplateSections } from './sea-template';

describe('海运订单新增模板', () => {
  it('仅生成需求约定的五个业务区块', () => {
    const sections = getSeaTemplateSections({
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      currencyOptions: [],
      containerSpecOptions: [{ label: '20GP', value: 'spec-20gp' }],
      searchCustomers: vi.fn().mockResolvedValue([]),
      searchCarriers: vi.fn().mockResolvedValue([]),
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
      { key: 'cargoInfo', title: '提单信息' },
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
    expect(transportSection).toHaveTextContent('主单 (MBL)');
    expect(transportSection).toHaveTextContent('分单号 (HBL)');
    expect(transportSection).toHaveTextContent('一主多分');
    expect(transportSection).toHaveTextContent('计划箱型箱量');
    expect(
      screen.getByRole('button', { name: /加拼主单 \(MBL\)/ }),
    ).toBeTruthy();
    expect(
      screen.getByRole('button', { name: /新增计划箱型箱量/ }),
    ).toBeTruthy();

    const cargoSection = screen.getByTestId('section-cargoInfo');
    expect(cargoSection).not.toHaveTextContent('主单号');
    expect(cargoSection).not.toHaveTextContent('分单号');
  });

  it('支持多组主分单的展开和删除', async () => {
    const sections = getSeaTemplateSections({
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      currencyOptions: [],
      containerSpecOptions: [],
      searchCustomers: vi.fn().mockResolvedValue([]),
      searchCarriers: vi.fn().mockResolvedValue([]),
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

    expect(
      screen.getAllByPlaceholderText('请输入主单号 (如 MBL-001)'),
    ).toHaveLength(1);

    const addMasterBtn = screen.getByRole('button', {
      name: /加拼主单 \(MBL\)/,
    });
    fireEvent.click(addMasterBtn);

    expect(
      screen.getAllByPlaceholderText('请输入主单号 (如 MBL-001)'),
    ).toHaveLength(2);
    expect(
      screen.getAllByRole('button', { name: /删除该主单组/ }),
    ).toHaveLength(2);

    fireEvent.click(screen.getAllByRole('button', { name: /删除该主单组/ })[1]);
    expect(
      screen.getAllByPlaceholderText('请输入主单号 (如 MBL-001)'),
    ).toHaveLength(1);
  });

  it('散杂托运隐藏箱型箱量并要求显式清理已有计划', () => {
    const sections = getSeaTemplateSections({
      serviceTypeOptions: [],
      cargoCategoryOptions: [],
      locationOptions: [],
      currencyOptions: [],
      containerSpecOptions: [{ label: '40HQ', value: 'spec-40hq' }],
      searchCustomers: vi.fn().mockResolvedValue([]),
      searchCarriers: vi.fn().mockResolvedValue([]),
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
      currencyOptions: [
        { label: 'CNY - 人民币', value: 'CNY' },
        { label: 'USD - 美元', value: 'USD' },
      ],
      containerSpecOptions: [],
      searchCustomers: vi.fn().mockResolvedValue([]),
      searchCarriers: vi.fn().mockResolvedValue([]),
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

    const cargoInputs = screen.getAllByPlaceholderText('金额');
    expect(cargoInputs.length).toBe(2);

    const currencySelects = screen.getAllByRole('combobox');
    expect(currencySelects.length).toBeGreaterThanOrEqual(2);
  });
});
