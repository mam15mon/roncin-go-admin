import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { Form, type FormInstance } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { airExportServiceListAirMasterWaybills } from '@/services/roncin/airExportService';
import { getAirTemplateSections } from './air-template';
import type { TemplateProps } from './types';

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListAirlines: vi.fn().mockResolvedValue({ data: [] }),
}));
vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: vi.fn(),
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
const user = { id: 'operator', currentOrganization: { id: 'company' } };
function setup(initialValues: Record<string, unknown>, isDetail = false) {
  let current!: FormInstance;
  function Editor() {
    const [form] = Form.useForm();
    current = form;
    return (
      <ProForm form={form} submitter={false} initialValues={initialValues}>
        {getAirTemplateSections({ ...props, isDetail })
          .filter((item) =>
            ['bookingTransport', 'airDocuments'].includes(item.key),
          )
          .map((item) => (
            <section key={item.key} aria-label={item.title}>
              {item.content}
            </section>
          ))}
      </ProForm>
    );
  }
  renderWithApp(<Editor />, { user });
  return () => current;
}

describe('航空直单与主分单表单', () => {
  beforeEach(() =>
    vi
      .mocked(airExportServiceListAirMasterWaybills)
      .mockResolvedValue({ data: [] }),
  );

  it('DIRECT没有HAWB字段，创建切回HOUSE保留原输入，主单正文仍独立存在', async () => {
    const readForm = setup({
      airDocumentMode: 'HOUSE',
      airHouseNo: 'HAWB-001',
      airMasterMode: 'NEW',
    });
    expect(
      screen.getByRole('textbox', { name: /本票分运单号（HAWB）/ }),
    ).toHaveValue('HAWB-001');
    fireEvent.click(
      screen.getByRole('radio', { name: '直单（DIRECT，无 HAWB）' }),
    );
    await waitFor(() =>
      expect(
        screen.queryByRole('textbox', { name: /本票分运单号（HAWB）/ }),
      ).toBeNull(),
    );
    expect(screen.getByText('直单：本票使用主运单，无 HAWB')).toBeVisible();
    expect(screen.getByText('主运单正文')).toBeVisible();
    expect(readForm().getFieldValue('airHouseNo')).toBe('HAWB-001');
    fireEvent.click(screen.getByRole('radio', { name: '主分单（HOUSE）' }));
    expect(
      await screen.findByRole('textbox', { name: /本票分运单号（HAWB）/ }),
    ).toHaveValue('HAWB-001');
  });

  it('已保存HOUSE的模式和号码不可普通修改，已签发状态从真实house消息读取', async () => {
    setup(
      {
        airDocumentMode: 'HOUSE',
        airHouseNo: 'HAWB-001',
        airExport: { documentMode: 'HOUSE', house: { status: 'RELEASED' } },
      },
      true,
    );
    expect(
      screen.getByRole('radio', { name: '直单（DIRECT，无 HAWB）' }),
    ).toBeDisabled();
    expect(
      screen.getByRole('textbox', { name: /本票分运单号（HAWB）/ }),
    ).toBeDisabled();
    expect(
      screen.getByRole('textbox', { name: /已签发分单修改原因/ }),
    ).toBeEnabled();
  });

  it('同号候选只按服务端资格与固定模式关联，不根据票数推断', async () => {
    vi.mocked(airExportServiceListAirMasterWaybills).mockResolvedValue({
      data: [
        {
          id: 'master',
          organizationId: 'company',
          businessType: 'AE',
          documentMode: 'HOUSE',
          canAttach: true,
          memberCount: 0,
          version: '3',
          content: { masterNo: '78112345675' },
        },
      ],
    });
    const readForm = setup({
      airDocumentMode: 'HOUSE',
      airMasterMode: 'NEW',
      airMaster: { masterNo: '78112345675' },
    });
    const select = await screen.findByRole('button', { name: '选择此主单' });
    expect(select).toBeEnabled();
    fireEvent.click(
      screen.getByRole('radio', { name: '直单（DIRECT，无 HAWB）' }),
    );
    await waitFor(() => expect(select).toBeDisabled());
    fireEvent.click(screen.getByRole('radio', { name: '主分单（HOUSE）' }));
    await waitFor(() => expect(select).toBeEnabled());
    fireEvent.click(select);
    await waitFor(() =>
      expect(readForm().getFieldValue('airMasterId')).toBe('master'),
    );
    expect(readForm().getFieldValue('airMasterExpectedVersion')).toBe('3');
    expect(screen.queryByText('主运单正文')).toBeNull();
  });

  it('Proto省略canAttach=false时拒绝关联，保留服务端阻断原因', async () => {
    const readForm = setup({
      airDocumentMode: 'DIRECT',
      airMasterMode: 'EXISTING',
      airMasterId: 'blocked',
      airMasterSelection: {
        id: 'blocked',
        organizationId: 'company',
        businessType: 'AE',
        version: '1',
        documentMode: 'DIRECT',
        attachBlockedReason: '直单已有当前操作票',
      },
    });
    expect(screen.getByText('直单已有当前操作票')).toBeVisible();
    await act(async () => {
      await expect(
        readForm().validateFields(['airMasterId']),
      ).rejects.toMatchObject({
        errorFields: [{ errors: ['直单已有当前操作票'] }],
      });
    });
  });
});
