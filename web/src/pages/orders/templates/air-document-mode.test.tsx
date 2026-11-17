import { ProForm } from '@ant-design/pro-components';
import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
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
      <ConfigProvider theme={{ token: { motion: false } }}>
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
      </ConfigProvider>
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
      expect(readForm().getFieldValue('airDocumentMode')).toBe('DIRECT'),
    );
    await waitFor(() =>
      expect(screen.queryByRole('tab', { name: /^分运单 \(HAWB\)/ })).toBeNull(),
    );
    expect(
      screen.queryByRole('textbox', { name: /本票分运单号（HAWB）/ }),
    ).toBeNull();
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
    expect(screen.getByText('主分单（HOUSE）')).toBeVisible();
    expect(
      screen.queryByRole('radio', { name: '直单（DIRECT，无 HAWB）' }),
    ).toBeNull();
    expect(
      screen.getByRole('textbox', { name: /本票分运单号（HAWB）/ }),
    ).toBeDisabled();
    expect(
      screen.getByRole('textbox', { name: /已签发分单修改原因/ }),
    ).toBeEnabled();
  });

  const sharedMaster = {
    id: 'master',
    organizationId: 'company',
    businessType: 'AE',
    documentMode: 'HOUSE',
    canAttach: true,
    memberCount: 2,
    version: '3',
    airlineName: '国航',
    content: { masterNo: '78112345675', flightNo: 'CA100' },
  };
  const masterNoBox = () =>
    screen.getByRole('textbox', { name: /主运单号（MAWB）/ });
  const validateMasterNo = (form: FormInstance) =>
    form.validateFields([['airMaster', 'masterNo']]);

  it('输入同号主单后自动核对：模式不符先拦截，切换后自动关联，改号即解除', async () => {
    vi.mocked(airExportServiceListAirMasterWaybills).mockResolvedValue({
      data: [sharedMaster],
    });
    const readForm = setup({
      airDocumentMode: 'DIRECT',
      airMaster: { masterNo: '78112345675' },
    });
    expect(
      await screen.findByText('本票保存后将加入该主单，共 3 票。'),
    ).toBeVisible();
    expect(airExportServiceListAirMasterWaybills).toHaveBeenCalledWith(
      expect.objectContaining({ businessType: 'AE', keyword: '78112345675' }),
      expect.anything(),
    );
    expect(readForm().getFieldValue('airMasterId')).toBeUndefined();
    await act(async () => {
      await expect(validateMasterNo(readForm())).rejects.toMatchObject({
        errorFields: [
          {
            errors: ['该主单为主分单（HOUSE），请切换单证模式后再保存'],
          },
        ],
      });
    });
    fireEvent.click(screen.getByRole('button', { name: '切换为主单模式' }));
    await waitFor(() =>
      expect(readForm().getFieldValue('airMasterId')).toBe('master'),
    );
    expect(readForm().getFieldValue('airDocumentMode')).toBe('HOUSE');
    expect(readForm().getFieldValue('airMasterMode')).toBe('EXISTING');
    expect(readForm().getFieldValue('airMasterExpectedVersion')).toBe('3');
    // 空字段自动带入主单值，关联后共同资料只读，主运单号仍可修改。
    expect(readForm().getFieldValue(['airMaster', 'flightNo'])).toBe('CA100');
    expect(screen.getByRole('textbox', { name: '航班号' })).toBeDisabled();
    expect(masterNoBox()).toBeEnabled();
    await act(async () => {
      await expect(validateMasterNo(readForm())).resolves.toBeTruthy();
    });

    fireEvent.change(masterNoBox(), { target: { value: '78100000000' } });
    expect(
      await screen.findByText('未找到同号主单，保存后将新建主单'),
    ).toBeVisible();
    expect(readForm().getFieldValue('airMasterMode')).toBe('NEW');
    expect(readForm().getFieldValue('airMasterId')).toBeUndefined();
    expect(screen.getByRole('textbox', { name: '航班号' })).toBeEnabled();
    expect(readForm().getFieldValue(['airMaster', 'flightNo'])).toBe('CA100');
  });

  it('本票已填资料与主单不同时列出差异，采用前拦截保存', async () => {
    vi.mocked(airExportServiceListAirMasterWaybills).mockResolvedValue({
      data: [sharedMaster],
    });
    const readForm = setup({
      airDocumentMode: 'HOUSE',
      airMaster: { masterNo: '78112345675', flightNo: 'MU200' },
    });
    expect(
      await screen.findByText('航班号：本票 MU200；主单 CA100'),
    ).toBeVisible();
    expect(readForm().getFieldValue('airMasterId')).toBeUndefined();
    await act(async () => {
      await expect(validateMasterNo(readForm())).rejects.toMatchObject({
        errorFields: [
          { errors: ['本票主单资料与已有主单不同，请确认采用主单资料'] },
        ],
      });
    });
    fireEvent.click(screen.getByRole('button', { name: '采用主单资料' }));
    await waitFor(() =>
      expect(readForm().getFieldValue('airMasterId')).toBe('master'),
    );
    expect(readForm().getFieldValue(['airMaster', 'flightNo'])).toBe('CA100');
  });

  it('Proto省略canAttach时视为不可关联，显示服务端原因并拦截保存', async () => {
    const { canAttach: _omitted, ...blocked } = sharedMaster;
    vi.mocked(airExportServiceListAirMasterWaybills).mockResolvedValue({
      data: [{ ...blocked, attachBlockedReason: '直单已有当前操作票' }],
    });
    const readForm = setup({
      airDocumentMode: 'HOUSE',
      airMaster: { masterNo: '78112345675' },
    });
    expect(await screen.findByText('直单已有当前操作票')).toBeVisible();
    expect(readForm().getFieldValue('airMasterId')).toBeUndefined();
    await act(async () => {
      await expect(validateMasterNo(readForm())).rejects.toMatchObject({
        errorFields: [{ errors: ['直单已有当前操作票'] }],
      });
    });
  });
});
