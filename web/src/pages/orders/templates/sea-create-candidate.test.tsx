import { renderWithClient as render } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import type { FormInstance } from 'antd';
import { describe, expect, it, vi } from 'vitest';
import * as orderService from '@/services/roncin/orderService';
import { buildSeaExportCreatePayload } from '../order-kinds/sea-export/form-adapter';
import { DIRECT, HOUSE, SectionCards } from './sea-create-layout.fixtures';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOrder: () => true,
  }),
}));

vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user-1',
        currentOrganization: { id: 'org-1', name: '测试组织' },
      },
    },
  }),
}));

vi.mock('@/services/roncin/seaDocumentService', () => ({
  seaDocumentServiceGetSeaOrderDocuments: vi.fn(),
  seaDocumentServicePreviewChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceExecuteChangeSeaDocumentMode: vi.fn(),
  seaDocumentServiceUpdateSeaHouseBill: vi.fn(),
  seaDocumentServiceUpdateSeaMasterBillContent: vi.fn(),
}));

vi.mock('@/services/roncin/orderReleasePodService', () => ({
  orderReleasePodServiceListReleasePods: vi
    .fn()
    .mockResolvedValue({ data: [] }),
}));

function candidateResponse(overrides?: {
  members?: API.SeaMasterBillMemberSummary[];
  conflicts?: API.SeaVoyageConflict[];
}) {
  return {
    matched: true,
    candidate: {
      id: 'mbl-candidate-1',
      version: '5',
      masterNo: 'COSCOAUTO1',
      shippingLineId: 'carrier-1',
      memberCount: 2,
      members:
        overrides?.members ??
        ([
          {
            orderId: 'order-a',
            orderNo: 'SE-A',
            documentStructure: HOUSE,
          },
          {
            orderId: 'order-b',
            orderNo: 'SE-B',
            documentStructure: HOUSE,
          },
        ] satisfies API.SeaMasterBillMemberSummary[]),
      batchNormalizedHouseNos: ['HBL-EXIST-A', 'HBL-EXIST-B'],
      transportExecutions: [
        {
          id: 'te-1',
          version: '3',
          vesselName: 'EVER TEST',
          voyageNo: '001W',
        },
      ],
    },
    conflicts: overrides?.conflicts ?? [],
  } as Awaited<
    ReturnType<typeof orderService.orderServiceMatchSeaMasterBillCandidate>
  >;
}

describe('SE 新建共享主单自动关联与批次排重', () => {
  it('命中全 HOUSE 批次写入候选确认并展示保存后加入提示', async () => {
    const matchSpy = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValue(candidateResponse());
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport']}
        initialValues={{
          seaDocumentStructure: HOUSE,
          shippingLineId: 'carrier-1',
          seaMasterBillMasterNo: 'COSCOAUTO1',
        }}
      />,
    );

    await waitFor(() =>
      expect(
        screen.getByText(/已找到主单 COSCOAUTO1，已有 2 票订单/),
      ).toBeInTheDocument(),
    );
    expect(matchSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        masterNo: 'COSCOAUTO1',
        shippingLineId: 'carrier-1',
      }),
      expect.objectContaining({ skipErrorHandler: true }),
    );
    // 候选确认参数（候选 ID + 版本 + 唯一航次）自动带入提交载荷。
    expect(form.getFieldValue('seaMasterBillCandidateId')).toBe(
      'mbl-candidate-1',
    );
    expect(form.getFieldValue('seaMasterBillExpectedCandidateVersion')).toBe(
      '5',
    );
    expect(form.getFieldValue('seaMasterBillCandidateTeId')).toBe('te-1');
    const payload = buildSeaExportCreatePayload(form.getFieldsValue(true));
    expect(payload.seaMasterBill?.candidateId).toBe('mbl-candidate-1');
    expect(payload.seaMasterBill?.expectedCandidateVersion).toBe('5');
  });

  it('批次含直单成员时展示红色阻断横幅且不自动关联', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(
      candidateResponse({
        members: [
          { orderId: 'order-a', orderNo: 'SE-A', documentStructure: DIRECT },
        ],
      }),
    );
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport']}
        initialValues={{
          seaDocumentStructure: HOUSE,
          shippingLineId: 'carrier-1',
          seaMasterBillMasterNo: 'COSCOAUTO1',
        }}
      />,
    );

    await waitFor(() =>
      expect(
        screen.getByText('该主单已被直单订单占用，如需拼单请先将其转为分单'),
      ).toBeInTheDocument(),
    );
    await waitFor(() => {
      const payload = buildSeaExportCreatePayload(form.getFieldsValue(true));
      expect(payload.seaMasterBill?.candidateId).toBeUndefined();
    });
  });

  it('分单号与批次内已用分单号重复时即时提示', async () => {
    vi.spyOn(
      orderService,
      'orderServiceMatchSeaMasterBillCandidate',
    ).mockResolvedValue(candidateResponse());
    let form!: FormInstance;
    render(
      <SectionCards
        exposeForm={(value) => {
          form = value;
        }}
        sectionKeys={['bookingTransport', 'seaDocument']}
        initialValues={{
          seaDocumentStructure: HOUSE,
          shippingLineId: 'carrier-1',
          seaMasterBillMasterNo: 'COSCOAUTO1',
        }}
      />,
    );

    fireEvent.click(screen.getByRole('tab', { name: /分单 \(HBL\)/ }));
    // 等待候选命中写入批次分单号清单。
    await waitFor(() =>
      expect(form.getFieldValue('seaMasterBillBatchHouseNos')).toEqual([
        'HBL-EXIST-A',
        'HBL-EXIST-B',
      ]),
    );
    fireEvent.change(await screen.findByPlaceholderText('请输入分单号'), {
      target: { value: ' hbl-exist-a ' },
    });
    await waitFor(() =>
      expect(
        screen.getByText(/在该主单批次内已存在（含作废）/),
      ).toBeInTheDocument(),
    );
    await act(async () => {
      await expect(
        form.validateFields([
          ['seaHouseBill', 'houseNo'],
          ['seaHouseBill', 'issuerSource'],
        ]),
      ).rejects.toMatchObject({
        errorFields: expect.arrayContaining([
          expect.objectContaining({ name: ['seaHouseBill', 'houseNo'] }),
        ]),
      });
    });
  });
});
