import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  SeaDocumentEventType,
  SeaDocumentType,
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import * as service from '@/services/roncin/seaDocumentService';
import SeaDocumentHistoryActions from './SeaDocumentHistoryActions';

vi.mock('@/services/roncin/orderAttachmentService', () => ({
  orderAttachmentServiceListAttachments: vi.fn().mockResolvedValue({ data: [] }),
}));

vi.mock('@umijs/max', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@umijs/max')>();
  return {
    ...actual,
    useAccess: () => ({ canOrder: () => true }),
  };
});

const historyServiceMocks = vi.hoisted(() => ({
  listMasterBillVersions: vi.fn(),
  listDocumentEvents: vi.fn(),
}));

vi.mock('@/services/roncin/seaDocumentService', async (importOriginal) => {
  const actual =
    await importOriginal<
      typeof import('@/services/roncin/seaDocumentService')
    >();
  return {
    ...actual,
    seaDocumentServiceListSeaMasterBillVersions:
      historyServiceMocks.listMasterBillVersions,
    seaDocumentServiceListSeaDocumentEvents:
      historyServiceMocks.listDocumentEvents,
  };
});

describe('SeaDocumentHistoryActions', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
    historyServiceMocks.listMasterBillVersions.mockReset();
    historyServiceMocks.listDocumentEvents.mockReset();
  });

  it('只有 Preview 成功并展示最终差异后才允许 Execute', async () => {
    let currentShipper = '新发货人';
    const preview = vi
      .spyOn(service, 'seaDocumentServicePreviewSeaDocumentAmendment')
      .mockResolvedValue({
        data: {
          executable: true,
          baseVersion: { documentNo: 'HBL-001', versionNo: '2' },
          differences: [
            {
              field: 'shipper_text',
              label: '发货人',
              beforeValue: '旧发货人',
              afterValue: '新发货人',
            },
          ],
          impacts: [],
        },
      } as Awaited<
        ReturnType<typeof service.seaDocumentServicePreviewSeaDocumentAmendment>
      >);
    const execute = vi
      .spyOn(service, 'seaDocumentServiceExecuteSeaDocumentAmendment')
      .mockResolvedValue({ data: { id: 'version-3' } } as Awaited<
        ReturnType<typeof service.seaDocumentServiceExecuteSeaDocumentAmendment>
      >);

    render(
      <App>
        <SeaDocumentHistoryActions
          orderId="00000000-0000-0000-0000-000000000001"
          orderVersion="5"
          documentType={SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL}
          documentId="00000000-0000-0000-0000-000000000002"
          documentNo="HBL-001"
          documentVersion="2"
          currentVersionId="00000000-0000-0000-0000-000000000003"
          currentHouseBill={{
            id: '00000000-0000-0000-0000-000000000002',
            houseNo: 'HBL-001',
            issuerSource:
              SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_CUSTOMER_PARTNER,
            status: SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_DRAFT,
            version: '2',
            currentVersionId: '00000000-0000-0000-0000-000000000003',
            content: { shipperText: '新发货人' },
          }}
          getAmendmentInput={() => ({
            houseBill: {
              houseNo: 'HBL-001',
              issuerSource:
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_CUSTOMER_PARTNER,
              content: { shipperText: currentShipper },
            },
          })}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    fireEvent.click(screen.getByRole('button', { name: /单\s*改/ }));
    const executeButton = screen.getByRole('button', {
      name: /确认执行/,
    });
    expect(executeButton).toBeDisabled();

    fireEvent.change(screen.getByLabelText('原因'), {
      target: { value: '客户书面更正' },
    });
    fireEvent.change(screen.getByLabelText('外部确认方'), {
      target: { value: '测试船代' },
    });
    fireEvent.change(screen.getByLabelText('确认说明'), {
      target: { value: '船代已邮件确认可改' },
    });
    fireEvent.click(screen.getByRole('button', { name: /重新预览最终差异/ }));

    await waitFor(() => {
      expect(preview).toHaveBeenCalledTimes(1);
      expect(screen.getByText('旧发货人')).toBeInTheDocument();
      expect(screen.getByText('新发货人')).toBeInTheDocument();
      expect(executeButton).toBeEnabled();
    });

    currentShipper = '预览后外部表单发生变化';
    fireEvent.click(executeButton);
    await waitFor(() => expect(execute).toHaveBeenCalledTimes(1));
    expect(
      execute.mock.calls[0][1].input?.houseBill?.content?.shipperText,
    ).toBe('新发货人');
    expect(execute.mock.calls[0][1].confirmation).toMatchObject({
      confirmedByParty: '测试船代',
      confirmationNote: '船代已邮件确认可改',
    });
    expect(preview.mock.invocationCallOrder[0]).toBeLessThan(
      execute.mock.invocationCallOrder[0],
    );
  });

  it('订单锁定时保留历史入口但关闭所有写命令', () => {
    render(
      <App>
        <SeaDocumentHistoryActions
          orderId="00000000-0000-0000-0000-000000000001"
          orderVersion="5"
          documentType={SeaDocumentType.SEA_DOCUMENT_TYPE_HOUSE_BILL}
          documentId="00000000-0000-0000-0000-000000000002"
          documentNo="HBL-LOCKED"
          documentVersion="2"
          currentVersionId="00000000-0000-0000-0000-000000000003"
          currentHouseBill={{
            status: SeaHouseBillStatus.SEA_HOUSE_BILL_STATUS_DRAFT,
          }}
          getAmendmentInput={() => ({
            houseBill: {
              houseNo: 'HBL-LOCKED',
              issuerSource:
                SeaHouseBillIssuerSource.SEA_HOUSE_BILL_ISSUER_SOURCE_CUSTOMER_PARTNER,
              content: {},
            },
          })}
          onSuccess={vi.fn()}
          disabled
        />
      </App>,
    );

    expect(screen.getByRole('button', { name: /版本与事件/ })).toBeEnabled();
    expect(screen.getByRole('button', { name: /单\s*改/ })).toBeDisabled();
    expect(screen.getByRole('button', { name: /作废/ })).toBeDisabled();
    expect(screen.queryByRole('button', { name: /Switch B\/L/ })).toBeNull();
  });

  it('MBL 不可变版本展开后显示船公司名称', async () => {
    historyServiceMocks.listMasterBillVersions.mockResolvedValue({
      data: [
        {
          id: 'version-1',
          documentType: SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL,
          documentNo: 'COSU123456',
          versionNo: '1',
          sourceEntityVersion: '1',
          shippingLineId: 'line-1',
          shippingLineName: '中远海运 / COSCO SHIPPING (COSU)',
          content: {},
        },
      ],
    });
    historyServiceMocks.listDocumentEvents.mockResolvedValue({
      data: [],
    });

    render(
      <App>
        <SeaDocumentHistoryActions
          orderId="00000000-0000-0000-0000-000000000001"
          orderVersion="5"
          documentType={SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL}
          documentId="00000000-0000-0000-0000-000000000002"
          documentNo="COSU123456"
          documentVersion="1"
          currentVersionId="00000000-0000-0000-0000-000000000003"
          getAmendmentInput={() => ({ masterBillContent: {} })}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    fireEvent.click(screen.getByRole('button', { name: /版本与事件/ }));
    await waitFor(() => {
      expect(historyServiceMocks.listMasterBillVersions).toHaveBeenCalledWith({
        orderId: '00000000-0000-0000-0000-000000000001',
        page: 1,
        pageSize: 200,
      });
      expect(screen.getByText('v1')).toBeInTheDocument();
    });
    fireEvent.click(screen.getByRole('button', { name: /Expand row/i }));
    expect(
      screen.getByText('中远海运 / COSCO SHIPPING (COSU)'),
    ).toBeInTheDocument();
  });

  it('版本总数超过一页时展示加载更多并追加第二页，刷新时重置回第一页', async () => {
    historyServiceMocks.listMasterBillVersions.mockImplementation(
      ({ page }: { page: number }) =>
        Promise.resolve({
          data: [
            {
              id: `version-${page}`,
              documentType: SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL,
              documentNo: 'COSU123456',
              versionNo: `${page}`,
              sourceEntityVersion: `${page}`,
              content: {},
            },
          ],
          total: 2,
        }),
    );
    historyServiceMocks.listDocumentEvents.mockResolvedValue({
      data: [],
      total: 0,
    });

    render(
      <App>
        <SeaDocumentHistoryActions
          orderId="00000000-0000-0000-0000-000000000001"
          orderVersion="5"
          documentType={SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL}
          documentId="00000000-0000-0000-0000-000000000002"
          documentNo="COSU123456"
          documentVersion="1"
          currentVersionId="00000000-0000-0000-0000-000000000003"
          getAmendmentInput={() => ({ masterBillContent: {} })}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    fireEvent.click(screen.getByRole('button', { name: /版本与事件/ }));
    expect(await screen.findByText('v1')).toBeInTheDocument();
    expect(
      await screen.findByRole('button', { name: '加载更多（共 2 条）' }),
    ).toBeInTheDocument();

    fireEvent.click(
      await screen.findByRole('button', { name: '加载更多（共 2 条）' }),
    );
    await waitFor(() =>
      expect(historyServiceMocks.listMasterBillVersions).toHaveBeenCalledWith({
        orderId: '00000000-0000-0000-0000-000000000001',
        page: 2,
        pageSize: 200,
      }),
    );
    expect(await screen.findByText('v2')).toBeInTheDocument();
    expect(screen.getByText('v1')).toBeInTheDocument();
    expect(
      screen.queryByRole('button', { name: /加载更多/ }),
    ).not.toBeInTheDocument();

    // 重新打开历史抽屉（loadHistory）时重置回第一页。
    fireEvent.click(screen.getByRole('button', { name: /版本与事件/ }));
    await waitFor(() =>
      expect(
        historyServiceMocks.listMasterBillVersions,
      ).toHaveBeenLastCalledWith({
        orderId: '00000000-0000-0000-0000-000000000001',
        page: 1,
        pageSize: 200,
      }),
    );
    expect(
      await screen.findByRole('button', { name: '加载更多（共 2 条）' }),
    ).toBeInTheDocument();
  });

  it('事件总数超过一页时按原始返回条数判断加载更多并追加下一页', async () => {
    historyServiceMocks.listMasterBillVersions.mockResolvedValue({
      data: [
        {
          id: 'version-1',
          documentType: SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL,
          documentNo: 'COSU123456',
          versionNo: '1',
          sourceEntityVersion: '1',
          content: {},
        },
      ],
      total: 1,
    });
    historyServiceMocks.listDocumentEvents.mockImplementation(
      ({ page }: { page: number }) =>
        Promise.resolve({
          data:
            page === 1
              ? [
                  {
                    id: 'event-1',
                    documentId: '00000000-0000-0000-0000-000000000002',
                    eventType:
                      SeaDocumentEventType.SEA_DOCUMENT_EVENT_TYPE_AMENDMENT,
                    documentNo: 'COSU123456',
                  },
                ]
              : [
                  {
                    id: 'event-2',
                    documentId: 'another-document',
                    eventType:
                      SeaDocumentEventType.SEA_DOCUMENT_EVENT_TYPE_AMENDMENT,
                    documentNo: 'OTHER-DOC',
                  },
                ],
          total: 2,
        }),
    );

    render(
      <App>
        <SeaDocumentHistoryActions
          orderId="00000000-0000-0000-0000-000000000001"
          orderVersion="5"
          documentType={SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL}
          documentId="00000000-0000-0000-0000-000000000002"
          documentNo="COSU123456"
          documentVersion="1"
          currentVersionId="00000000-0000-0000-0000-000000000003"
          getAmendmentInput={() => ({ masterBillContent: {} })}
          onSuccess={vi.fn()}
        />
      </App>,
    );

    fireEvent.click(screen.getByRole('button', { name: /版本与事件/ }));
    expect(await screen.findByText('COSU123456')).toBeInTheDocument();
    expect(
      await screen.findByRole('button', { name: '加载更多（共 2 条）' }),
    ).toBeInTheDocument();

    fireEvent.click(
      screen.getByRole('button', { name: '加载更多（共 2 条）' }),
    );
    await waitFor(() =>
      expect(historyServiceMocks.listDocumentEvents).toHaveBeenCalledWith({
        orderId: '00000000-0000-0000-0000-000000000001',
        page: 2,
        pageSize: 200,
      }),
    );
    // 第二页事件不属于本单证，展示层过滤但计入已加载数量，全部加载后按钮消失。
    await waitFor(() =>
      expect(
        screen.queryByRole('button', { name: /加载更多/ }),
      ).not.toBeInTheDocument(),
    );
    expect(screen.getByText('COSU123456')).toBeInTheDocument();
    expect(screen.queryByText('OTHER-DOC')).not.toBeInTheDocument();
  });
});
