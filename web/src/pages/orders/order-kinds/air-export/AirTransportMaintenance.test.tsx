import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider, Form, Input } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import {
  airExportServiceGetAirMasterWaybill,
  airExportServicePreviewAirMasterWaybillUpdate,
  airExportServiceUpdateAirMasterWaybill,
} from '@/services/roncin/airExportService';
import type { OrderDetailFeaturesContext } from '../types';
import AirTransportMaintenance from './AirTransportMaintenance';

vi.mock('@/services/roncin/airExportService', () => ({
  airExportServiceGetAirMasterWaybill: vi.fn(),
  airExportServicePreviewAirMasterWaybillUpdate: vi.fn(),
  airExportServiceUpdateAirMasterWaybill: vi.fn(),
}));
// 航空字段组件有独立真实字段测试；此处保留官方表单绑定，专验独立命令的基线与输入保护。
vi.mock('../../templates/components/air/AirMasterFields', async () => {
  const { ProFormText } = await import('@ant-design/pro-components');
  return {
    AirMasterFields: () => (
      <ProFormText name={['input', 'flightNo']} label="运输航班" />
    ),
  };
});
const get = vi.mocked(airExportServiceGetAirMasterWaybill);
const preview = vi.mocked(airExportServicePreviewAirMasterWaybillUpdate);
const update = vi.mocked(airExportServiceUpdateAirMasterWaybill);
const master: API.AirMasterWaybill = {
  id: 'master',
  organizationId: 'company',
  businessType: 'AI',
  version: '4',
  content: {
    masterNo: '78112345675',
    flightNo: 'CA123',
    waybillContent: { remarks: '原单证正文', shipperText: '原发货人' },
  },
};
const refresh = vi.fn();
function context(
  overrides: Partial<OrderDetailFeaturesContext> = {},
): OrderDetailFeaturesContext {
  return {
    kind: 'air-import',
    orderId: 'order',
    orderFormIdentity: 'air-import:order',
    order: {
      id: 'order',
      organizationId: 'company',
      businessType: OrderBusinessType.BUSINESS_TYPE_AI,
      airExport: { master },
    },
    businessWritesDisabled: false,
    canOrder: () => true,
    searchShippingLines: async () => [],
    searchLocations: async () => [],
    containerSpecOptions: [],
    refreshOrderAndLock: refresh,
    ensureBusinessWriteAllowed: () => true,
    ...overrides,
  };
}
function mount(value = context()) {
  return renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Form initialValues={{ notes: '订单未保存输入' }}>
        <Form.Item name="notes" label="订单备注">
          <Input />
        </Form.Item>
        <AirTransportMaintenance context={value} />
      </Form>
    </ConfigProvider>,
    { routerProps: { initialEntries: ['/orders/air-import/order'] } },
  );
}
async function openEditor() {
  fireEvent.click(screen.getByRole('button', { name: '修改共同运输资料' }));
  const flight = await screen.findByLabelText('运输航班');
  await waitFor(() => expect(flight).toBeVisible());
  fireEvent.change(screen.getByLabelText('修改原因'), {
    target: { value: '调整航班' },
  });
  return flight;
}
describe('订单内修改共同航空运输资料', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    vi.spyOn(document, 'hasFocus').mockReturnValue(false);
    get.mockResolvedValue({ success: true, data: structuredClone(master) });
    preview.mockResolvedValue({
      success: true,
      inputFingerprint: 'fingerprint',
      canExecute: true,
      members: [
        { orderId: 'order', orderNo: 'AI001' },
        { orderId: 'other', orderNo: 'AI002' },
      ],
    });
    update.mockResolvedValue(
      undefined as unknown as API.UpdateAirMasterWaybillResponse,
    );
    refresh.mockResolvedValue(undefined);
  });

  it('预览和执行使用打开版本，保留完整正文；失败保留输入，成功只刷新摘要', async () => {
    mount();
    const flight = await openEditor();
    fireEvent.change(flight, { target: { value: 'CA456' } });
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    const submit = await screen.findByRole('button', {
      name: '确认影响并提交',
    });
    expect(screen.getByText('AI001、AI002')).toBeVisible();
    expect(preview).toHaveBeenLastCalledWith(
      { id: 'master' },
      expect.objectContaining({
        expectedVersion: '4',
        input: expect.objectContaining({
          flightNo: 'CA456',
          waybillContent: master.content?.waybillContent,
        }),
      }),
    );
    fireEvent.click(submit);
    await waitFor(() => expect(update).toHaveBeenCalledTimes(1));
    expect(flight).toHaveValue('CA456');
    expect(refresh).not.toHaveBeenCalled();
    expect(screen.getByLabelText('订单备注')).toHaveValue('订单未保存输入');
    update.mockResolvedValue({
      success: true,
      data: { ...master, version: '5' },
    });
    fireEvent.click(submit);
    await waitFor(() => expect(refresh).toHaveBeenCalledWith());
    await waitFor(() =>
      expect(screen.queryByLabelText('运输航班')).not.toBeInTheDocument(),
    );
    expect(screen.getByLabelText('订单备注')).toHaveValue('订单未保存输入');
  });

  it('预览在途继续编辑使迟到预览失效，不允许借旧指纹提交', async () => {
    mount();
    const flight = await openEditor();
    let resolve!: (value: API.PreviewAirMasterWaybillUpdateResponse) => void;
    preview.mockImplementationOnce(
      () =>
        new Promise((done) => {
          resolve = done;
        }),
    );
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    await waitFor(() => expect(preview).toHaveBeenCalled());
    fireEvent.change(flight, { target: { value: '新航班' } });
    await act(async () =>
      resolve({ inputFingerprint: 'old', canExecute: true }),
    );
    expect(
      screen.queryByRole('button', { name: '确认影响并提交' }),
    ).not.toBeInTheDocument();
    expect(flight).toHaveValue('新航班');
    expect(update).not.toHaveBeenCalled();
  });

  it('共享成员阻止原因可见，且只读/审核关闭入口不请求主单', async () => {
    const view = mount(context({ businessWritesDisabled: true }));
    expect(
      screen.getByRole('button', { name: '修改共同运输资料' }),
    ).toBeDisabled();
    expect(get).not.toHaveBeenCalled();
    view.unmount();
    preview.mockResolvedValue({
      inputFingerprint: 'blocked',
      blockedReasons: ['关联订单已审核'],
    });
    mount();
    await openEditor();
    fireEvent.click(screen.getByRole('button', { name: '预览影响' }));
    expect(await screen.findByText('共享修改被阻止')).toBeVisible();
    expect(screen.getByText('关联订单已审核')).toBeVisible();
    expect(update).not.toHaveBeenCalled();
  });
});
