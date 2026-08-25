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
    expect(transportSection).toHaveTextContent('自动归集拼载批次');
    expect(transportSection).toHaveTextContent('箱型箱量');
    expect(
      screen.getByRole('button', { name: /添加主单分组 \(MBL\)/ }),
    ).toBeTruthy();
    expect(screen.getByRole('button', { name: /新增箱型箱量/ })).toBeTruthy();

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
      name: /添加主单分组 \(MBL\)/,
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
