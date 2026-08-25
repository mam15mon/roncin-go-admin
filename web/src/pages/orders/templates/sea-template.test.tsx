import { describe, expect, it, vi } from 'vitest';
import { ProForm } from '@ant-design/pro-components';
import { render, screen } from '@testing-library/react';
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
    expect(transportSection).toHaveTextContent('主单号 / 分单号');
    expect(transportSection).toHaveTextContent('箱型箱量');
    expect(screen.getByRole('button', { name: /新增主分单/ })).toBeTruthy();
    expect(screen.getByRole('button', { name: /新增箱型箱量/ })).toBeTruthy();

    const cargoSection = screen.getByTestId('section-cargoInfo');
    expect(cargoSection).not.toHaveTextContent('主单号 / 分单号');
  });
});
