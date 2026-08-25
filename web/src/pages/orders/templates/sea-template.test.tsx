import { describe, expect, it, vi } from 'vitest';
import { ProForm } from '@ant-design/pro-components';
import { fireEvent, render, screen } from '@testing-library/react';
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
    expect(transportSection).toHaveTextContent('主单号');
    expect(transportSection).toHaveTextContent('分单号');
    expect(transportSection).toHaveTextContent(
      '复用其他订单的主单号，即加入同一拼载批次',
    );
    expect(transportSection).toHaveTextContent('箱型箱量');
    expect(
      screen.getAllByRole('button', { name: '新增一组主分单' }),
    ).toHaveLength(2);
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

    // Initial state: no appended lists
    expect(screen.queryByText('其他主单号')).toBeNull();
    expect(screen.queryByText('其他分单号')).toBeNull();

    // Click append button for master bill
    const addMasterBtn = screen.getAllByRole('button', {
      name: '新增一组主分单',
    })[0];
    fireEvent.click(addMasterBtn);

    // Both master and house appended boxes should show row 1
    expect(screen.getByText('其他主单号')).toBeTruthy();
    expect(screen.getByText('其他分单号')).toBeTruthy();
    expect(
      screen.getAllByRole('button', { name: '删除主分单组合1' }),
    ).toHaveLength(2);

    // Click delete button
    fireEvent.click(
      screen.getAllByRole('button', { name: '删除主分单组合1' })[0],
    );
    expect(screen.queryByText('其他主单号')).toBeNull();
    expect(screen.queryByText('其他分单号')).toBeNull();
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
