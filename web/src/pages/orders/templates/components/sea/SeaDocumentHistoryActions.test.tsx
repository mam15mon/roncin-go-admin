import { fireEvent, render, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import {
  SeaDocumentType,
  SeaHouseBillIssuerSource,
  SeaHouseBillStatus,
} from '@/enums.generated';
import * as service from '@/services/roncin/seaDocumentService';
import SeaDocumentHistoryActions from './SeaDocumentHistoryActions';

vi.mock('@umijs/max', async (importOriginal) => {
  const actual = await importOriginal<typeof import('@umijs/max')>();
  return {
    ...actual,
    useAccess: () => ({ canOrder: () => true }),
  };
});

describe('SeaDocumentHistoryActions', () => {
  beforeEach(() => {
    vi.restoreAllMocks();
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
    expect(screen.getByRole('button', { name: /Switch B\/L/ })).toBeDisabled();
  });
});
