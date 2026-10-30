import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { SeaDocumentStructure } from '@/enums.generated';
import * as orderService from '@/services/roncin/orderService';
import {
  candidateResponse,
  masterField,
  renderVoyageForm,
  voyage,
} from './SeaMasterBillFields.test-utils';

type MatchResult = Awaited<
  ReturnType<typeof orderService.orderServiceMatchSeaMasterBillCandidate>
>;
const confirmationFields = [
  'seaMasterBillCandidateId',
  'seaMasterBillExpectedCandidateVersion',
  'seaMasterBillCandidateTeId',
  'seaMasterBillExpectedCandidateTeVersion',
  'seaMasterBillBatchHouseNos',
  'seaMasterBillSharedVoyage',
];
afterEach(() => vi.restoreAllMocks());

describe('主单候选身份与异常路径', () => {
  it('缺船公司提示待核对，保存同时收集独立字段错误', async () => {
    const match = vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    );
    const { form } = renderVoyageForm({ shippingLineId: undefined });
    expect(
      await screen.findByText('待选择船公司后核对主单占用情况'),
    ).toBeInTheDocument();
    await act(async () => {
      await expect(form.validateFields()).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({
            name: ['shippingLineId'],
            errors: ['请选择船公司'],
          }),
          expect.objectContaining({ name: ['etd'] }),
          expect.objectContaining({ name: ['eta'] }),
        ]),
      });
    });
    expect(match).not.toHaveBeenCalled();
  });

  it('切换身份同步阻止旧确认提交，并清空关联元数据保留用户输入', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockImplementation(async ({ masterNo }) =>
      masterNo === 'MBL123' ? candidateResponse() : { matched: false },
    );
    const { form } = renderVoyageForm({
      destinationLocationId: 'destination',
      seaHouseBill: { houseNo: 'HBL-NEW' },
    });
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateId')).toBe('mbl-1'),
    );
    await act(async () => {
      form.setFieldValue(masterField, 'NEW456');
      // 同一调用栈内校验，覆盖 useWatch/Effect 尚未同步的窗口。
      await expect(form.validateFields([masterField])).rejects.toMatchObject({
        errorFields: [
          expect.objectContaining({ errors: ['正在核对已有主单，请稍候'] }),
        ],
      });
    });
    for (const field of confirmationFields)
      expect(form.getFieldValue(field)).toBeUndefined();
    expect(form.getFieldValue('vesselVoyage')).toBe('EVER TEST / 001W');
    expect(form.getFieldValue('destinationLocationId')).toBe('destination');
    expect(form.getFieldValue(['seaHouseBill', 'houseNo'])).toBe('HBL-NEW');
  });

  it('船公司切换和非法号码立即移除旧关联，失败响应不恢复旧候选', async () => {
    const late = Promise.withResolvers<MatchResult>();
    const match = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValueOnce(candidateResponse())
      .mockReturnValueOnce(late.promise);
    const { form } = renderVoyageForm();
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateId')).toBe('mbl-1'),
    );
    await act(async () => form.setFieldValue('shippingLineId', 'carrier-2'));
    await waitFor(() => expect(match).toHaveBeenCalledTimes(2));
    expect(match.mock.calls[1][0]).toMatchObject({
      masterNo: 'MBL123',
      shippingLineId: 'carrier-2',
    });
    expect(form.getFieldValue('seaMasterBillCandidateId')).toBeUndefined();
    await act(async () => form.setFieldValue(masterField, 'INVALID-1'));
    await act(async () => late.resolve(candidateResponse()));
    for (const field of confirmationFields)
      expect(form.getFieldValue(field)).toBeUndefined();
    await act(async () => {
      await expect(form.validateFields([masterField])).rejects.toMatchObject({
        errorFields: [
          expect.objectContaining({
            errors: ['主单号仅允许包含英文字母和数字，禁止包含空格或特殊字符'],
          }),
        ],
      });
    });
  });

  it('迟到的旧主单响应不覆盖当前未命中的新主单', async () => {
    const old = Promise.withResolvers<MatchResult>();
    const match = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockReturnValueOnce(old.promise)
      .mockResolvedValueOnce({ matched: false });
    const { form } = renderVoyageForm();
    await waitFor(() => expect(match).toHaveBeenCalledOnce());
    await act(async () => form.setFieldValue(masterField, 'NEW456'));
    await waitFor(() => expect(match).toHaveBeenCalledTimes(2));
    await screen.findByLabelText('MBL 主单号核对通过');
    await act(async () => old.resolve(candidateResponse()));
    expect(form.getFieldValue(masterField)).toBe('NEW456');
    expect(form.getFieldValue('seaMasterBillCandidateId')).toBeUndefined();
    expect(form.getFieldValue('vesselVoyage')).toBeUndefined();
    expect(screen.queryByText(/已找到主单 MBL123/)).not.toBeInTheDocument();
  });

  it('候选批次含直单或本票未选HOUSE均不继承航程', async () => {
    const match = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValue(
        candidateResponse({
          members: [
            {
              orderId: 'direct',
              documentStructure:
                SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
            },
          ],
        }),
      );
    const first = renderVoyageForm();
    await screen.findByText('该主单已被直单订单占用，如需拼单请先将其转为分单');
    expect(
      first.form.getFieldValue('seaMasterBillCandidateId'),
    ).toBeUndefined();
    expect(first.form.getFieldValue('vesselVoyage')).toBeUndefined();
    first.unmount();
    match.mockResolvedValue(candidateResponse());
    const second = renderVoyageForm({ seaDocumentStructure: undefined });
    await screen.findByText(
      '共享主单仅支持有货代分单（HOUSE），请选择分单模式',
    );
    expect(
      second.form.getFieldValue('seaMasterBillCandidateId'),
    ).toBeUndefined();
    expect(second.form.getFieldValue('vesselVoyage')).toBeUndefined();
  });

  it('已确认候选切换后查询失败不保留旧确认，并显示可重试错误', async () => {
    vi.spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValueOnce(candidateResponse())
      .mockRejectedValueOnce(new Error('核对请求失败'));
    const { form } = renderVoyageForm();
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateId')).toBe('mbl-1'),
    );
    await act(async () => form.setFieldValue(masterField, 'NEW456'));
    // 先按文本定位，避免全局角色查询的样式计算阻塞身份防抖回调。
    const retryButton = (await screen.findByText('重新核对')).closest('button');
    await waitFor(() => {
      expect(retryButton).toBeVisible();
      expect(retryButton).toBeEnabled();
    });
    expect(retryButton).toHaveAccessibleName('重新核对');
    for (const field of confirmationFields)
      expect(form.getFieldValue(field)).toBeUndefined();
    await act(async () => {
      await expect(form.validateFields([masterField])).rejects.toMatchObject({
        errorFields: [expect.objectContaining({ errors: ['核对请求失败'] })],
      });
    });
  });

  it('同步改为DIRECT后不允许旧HOUSE确认通过校验', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(candidateResponse());
    const { form } = renderVoyageForm();
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateId')).toBe('mbl-1'),
    );
    await act(async () => {
      form.setFieldValue(
        'seaDocumentStructure',
        SeaDocumentStructure.SEA_DOCUMENT_STRUCTURE_DIRECT,
      );
      await expect(form.validateFields([masterField])).rejects.toMatchObject({
        errorFields: [
          expect.objectContaining({
            errors: ['共享主单仅支持有货代分单（HOUSE），请选择分单模式'],
          }),
        ],
      });
    });
    expect(form.getFieldValue('seaMasterBillCandidateId')).toBeUndefined();
  });

  it('多航程不默认第一条，明确选择后继承且换身份后不复用旧选择', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(
      candidateResponse({
        transportExecutions: [
          voyage,
          { ...voyage, id: 'te-2', vesselName: 'SECOND', voyageNo: '002W' },
        ],
      }),
    );
    const { form } = renderVoyageForm();
    const selector = await screen.findByRole(
      'combobox',
      { name: '本票共用航程' },
      // 真正的 300ms 身份防抖在全量并发负载下仍需等待完成。
      { timeout: 5000 },
    );
    expect(form.getFieldValue('seaMasterBillCandidateTeId')).toBeUndefined();
    expect(form.getFieldValue('vesselVoyage')).toBeUndefined();
    fireEvent.mouseDown(selector);
    fireEvent.click(await screen.findByText(/SECOND \/ 002W/));
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillCandidateTeId')).toBe('te-2'),
    );
    expect(form.getFieldValue('vesselVoyage')).toBe('SECOND / 002W');
    await act(async () => form.setFieldValue(masterField, 'OTHER456'));
    await screen.findByRole('combobox', { name: '本票共用航程' });
    expect(form.getFieldValue('seaMasterBillCandidateTeId')).toBeUndefined();
  });
});
