import { ProForm, ProFormText } from '@ant-design/pro-components';
import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider, Form, type FormInstance } from 'antd';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import * as orderService from '@/services/roncin/orderService';
import { candidateResponse } from './SeaMasterBillFields.test-utils';
import { SeaMasterBillFields } from './SeaTransportSection';

type MatchResult = Awaited<
  ReturnType<typeof orderService.orderServiceMatchSeaMasterBillCandidate>
>;
const houseField = ['seaHouseBill', 'houseNo'];
const successLabel = 'HBL 分单号核对通过';
const loadingLabel = 'HBL 分单号核对中';

function setup(initialValues: Record<string, unknown> = {}, isDetail = false) {
  let form!: FormInstance;
  function Harness() {
    const [instance] = Form.useForm();
    form = instance;
    return (
      <ProForm
        form={instance}
        submitter={false}
        initialValues={{
          seaDocumentStructure:
            SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_HOUSE,
          shippingLineId: 'carrier-1',
          seaMasterBillMasterNo: 'MBL123',
          seaHouseBill: { houseNo: 'HBL-NEW' },
          ...initialValues,
        }}
      >
        <ProFormText name="shippingLineId" label="船公司" />
        <SeaMasterBillFields isDetail={isDetail} />
      </ProForm>
    );
  }
  const result = renderWithClient(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <Harness />
    </ConfigProvider>,
  );
  return {
    ...result,
    get form() {
      return form;
    },
  };
}

afterEach(() => vi.restoreAllMocks());

describe('新建订单分单号可见核对反馈', () => {
  it('身份未齐提示等待，核对中新主单不能通过，成功未命中仅展示绿色勾', async () => {
    const pending = Promise.withResolvers<MatchResult>();
    const match = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockReturnValue(pending.promise);
    const { form } = setup({
      shippingLineId: undefined,
      seaMasterBillMasterNo: undefined,
      // 恢复的草稿不能把旧核对结果冒充本次查询成功。
      seaHouseNumberCheck: { identity: 'carrier-1:MBL123', status: 'checked' },
    });
    expect(
      await screen.findByText('待选择船公司和填写主单号后核对分单号'),
    ).toBeVisible();
    expect(match).not.toHaveBeenCalled();
    expect(screen.queryByLabelText(successLabel)).toBeNull();
    await act(async () =>
      form.setFieldsValue({
        shippingLineId: 'carrier-1',
        seaMasterBillMasterNo: 'MBL123',
      }),
    );
    expect(await screen.findByLabelText(loadingLabel)).toBeVisible();
    expect(screen.getByText('正在核对当前主单内分单号')).toBeVisible();
    expect(screen.queryByLabelText(successLabel)).toBeNull();
    await waitFor(() => expect(match).toHaveBeenCalledOnce());
    await act(async () => pending.resolve({ matched: false }));
    expect(await screen.findByLabelText(successLabel)).toBeVisible();
    expect(screen.queryByText('当前主单内分单号可用')).toBeNull();
    expect(form.getFieldValue(houseField)).toBe('HBL-NEW');
  });

  it('命中主单展示可用勾，同主单重复报错且修改分单号后清错变绿', async () => {
    const match = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValue(candidateResponse());
    const { form } = setup();
    expect(await screen.findByLabelText(successLabel)).toBeVisible();
    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: ' hbl-old ' },
    });
    expect(
      await screen.findByText(/分单号 HBL-OLD 在该主单批次内已存在（含作废）/),
    ).toBeVisible();
    expect(screen.queryByLabelText(successLabel)).toBeNull();
    expect(screen.queryByText('当前主单内分单号重复')).toBeNull();
    fireEvent.change(screen.getByPlaceholderText('请输入分单号'), {
      target: { value: 'HBL-FREE' },
    });
    await waitFor(() => expect(form.getFieldError(houseField)).toEqual([]));
    expect(screen.getByLabelText(successLabel)).toBeVisible();
    expect(form.getFieldValue(houseField)).toBe('HBL-FREE');
    expect(match).toHaveBeenCalledOnce();
  });

  it('更换主单立刻清旧重复，空批次成功后原分单号可用', async () => {
    const pending = Promise.withResolvers<MatchResult>();
    const match = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValueOnce(candidateResponse())
      .mockReturnValueOnce(pending.promise);
    const { form } = setup({ seaHouseBill: { houseNo: 'HBL-OLD' } });
    expect(
      await screen.findByText(/分单号 HBL-OLD 在该主单批次内已存在/),
    ).toBeVisible();
    fireEvent.change(screen.getByPlaceholderText('请输入主单号'), {
      target: { value: 'MBL456' },
    });
    expect(await screen.findByLabelText(loadingLabel)).toBeVisible();
    await waitFor(() => expect(form.getFieldError(houseField)).toEqual([]));
    expect(screen.queryByLabelText(successLabel)).toBeNull();
    await waitFor(() => expect(match).toHaveBeenCalledTimes(2));
    await act(async () =>
      pending.resolve(
        candidateResponse({ masterNo: 'MBL456', batchNormalizedHouseNos: [] }),
      ),
    );
    expect(await screen.findByLabelText(successLabel)).toBeVisible();
    expect(form.getFieldError(houseField)).toEqual([]);
    expect(form.getFieldValue(houseField)).toBe('HBL-OLD');
  });

  it('切换船公司撤销旧通过，旧候选迟到不污染新主单核对', async () => {
    const pending = Promise.withResolvers<MatchResult>();
    const match = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValueOnce(candidateResponse())
      .mockReturnValueOnce(pending.promise)
      .mockResolvedValueOnce({ matched: false });
    const { form } = setup({ seaHouseBill: { houseNo: 'HBL-LATE' } });
    expect(await screen.findByLabelText(successLabel)).toBeVisible();
    fireEvent.change(screen.getByLabelText('船公司'), {
      target: { value: 'carrier-2' },
    });
    expect(await screen.findByLabelText(loadingLabel)).toBeVisible();
    expect(screen.queryByLabelText(successLabel)).toBeNull();
    await waitFor(() => expect(match).toHaveBeenCalledTimes(2));
    expect(match.mock.calls[1][0]).toMatchObject({
      shippingLineId: 'carrier-2',
      masterNo: 'MBL123',
    });
    await act(async () =>
      form.setFieldValue('seaMasterBillMasterNo', 'MBL789'),
    );
    expect(await screen.findByLabelText(successLabel)).toBeVisible();
    await act(async () =>
      pending.resolve(
        candidateResponse({ batchNormalizedHouseNos: ['HBL-LATE'] }),
      ),
    );
    expect(screen.getByLabelText(successLabel)).toBeVisible();
    expect(form.getFieldError(houseField)).toEqual([]);
    expect(form.getFieldValue(houseField)).toBe('HBL-LATE');
    expect(match).toHaveBeenCalledTimes(3);
  });

  it('请求失败显示核对失败，复用主单重新核对后恢复绿色勾', async () => {
    const match = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockRejectedValueOnce(new Error('核对请求失败'))
      .mockResolvedValueOnce({ matched: false });
    setup();
    expect(
      await screen.findByText('分单号核对失败，请在主单号处点击「重新核对」'),
    ).toBeVisible();
    expect(screen.queryByLabelText(successLabel)).toBeNull();
    fireEvent.click(screen.getByRole('button', { name: '重新核对' }));
    expect(await screen.findByLabelText(successLabel)).toBeVisible();
    expect(match).toHaveBeenCalledTimes(2);
  });

  it('详情主单身份未改变时不将未核对分单标为通过', async () => {
    const match = vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    );
    const { form } = setup(
      {
        seaMasterBill: {
          masterNo: 'MBL123',
          shippingLineId: 'carrier-1',
          memberCount: 1,
        },
      },
      true,
    );
    await act(async () => {
      await form.validateFields(['seaMasterBillMasterNo']);
    });
    expect(match).not.toHaveBeenCalled();
    expect(screen.queryByLabelText(successLabel)).toBeNull();
    expect(screen.queryByLabelText(loadingLabel)).toBeNull();
  });
});
