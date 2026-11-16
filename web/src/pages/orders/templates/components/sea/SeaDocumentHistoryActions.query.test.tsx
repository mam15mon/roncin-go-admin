import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { SeaDocumentType } from '@/enums.generated';
import SeaDocumentHistoryActions from './SeaDocumentHistoryActions';

const service = vi.hoisted(() => ({
  preview: vi.fn(),
  versions: vi.fn(),
  events: vi.fn(),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOrder: () => true,
  }),
}));
vi.mock('@/services/roncin/orderAttachmentService', () => ({
  orderAttachmentServiceListAttachments: vi
    .fn()
    .mockResolvedValue({ success: true, data: [] }),
}));
vi.mock('@/services/roncin/seaDocumentService', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/services/roncin/seaDocumentService')
  >()),
  seaDocumentServicePreviewSeaDocumentAmendment: service.preview,
  seaDocumentServiceListSeaMasterBillVersions: service.versions,
  seaDocumentServiceListSeaDocumentEvents: service.events,
}));

const actions = (documentId = 'master-1', version = '2') => (
  <App>
    <SeaDocumentHistoryActions
      orderId="order-1"
      orderVersion="5"
      documentType={SeaDocumentType.SEA_DOCUMENT_TYPE_MASTER_BILL}
      documentId={documentId}
      documentNo="MBL-001"
      documentVersion={version}
      currentVersionId={`version-${version}`}
      getAmendmentInput={() => ({
        masterBillContent: { shipperText: '当前正文' },
      })}
      onSuccess={vi.fn()}
    />
  </App>
);
function openPreview() {
  fireEvent.click(screen.getByRole('button', { name: /单\s*改/ }));
  fireEvent.change(screen.getByLabelText('原因'), {
    target: { value: '客户更正' },
  });
  fireEvent.change(screen.getByLabelText('外部确认方'), {
    target: { value: '船代' },
  });
  fireEvent.change(screen.getByLabelText('确认说明'), {
    target: { value: '邮件确认' },
  });
  fireEvent.click(screen.getByRole('button', { name: /重新预览最终差异/ }));
}
beforeEach(() => vi.clearAllMocks());

describe('单证历史与命令基线隔离', () => {
  it('当前单证版本变化后旧预览立即不可执行，重取预览使用新基线', async () => {
    service.preview.mockResolvedValue({
      success: true,
      data: { executable: true },
    });
    const view = renderWithClient(actions());
    openPreview();
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /确认执行/ })).toBeEnabled(),
    );
    view.rerender(actions('master-1', '3'));
    expect(screen.getByRole('button', { name: /确认执行/ })).toBeDisabled();
    fireEvent.click(screen.getByRole('button', { name: /重新预览最终差异/ }));
    await waitFor(() =>
      expect(screen.getByRole('button', { name: /确认执行/ })).toBeEnabled(),
    );
    expect(service.preview).toHaveBeenLastCalledWith(
      { orderId: 'order-1' },
      expect.objectContaining({
        documentId: 'master-1',
        expectedDocumentVersion: '3',
        expectedCurrentVersionId: 'version-3',
      }),
    );
  });

  it('换单证后旧预览迟到不能启用当前单证的执行', async () => {
    const pending = Promise.withResolvers<{
      success: boolean;
      data: { executable: boolean };
    }>();
    service.preview.mockReturnValue(pending.promise);
    const view = renderWithClient(actions());
    openPreview();
    await waitFor(() => expect(service.preview).toHaveBeenCalledOnce());
    view.rerender(actions('master-2'));
    await act(async () =>
      pending.resolve({ success: true, data: { executable: true } }),
    );
    await waitFor(() =>
      expect(
        screen.getByRole('button', { name: /重新预览最终差异/ }),
      ).toBeEnabled(),
    );
    expect(screen.getByRole('button', { name: /确认执行/ })).toBeDisabled();
    expect(screen.queryByText('预览通过，可以执行')).not.toBeInTheDocument();
  });

  it('关闭历史取消版本与事件请求，重开从第一页重新读取', async () => {
    const oldVersions = Promise.withResolvers<{
      data: API.SeaDocumentVersion[];
    }>();
    const oldEvents = Promise.withResolvers<{ data: API.SeaDocumentEvent[] }>();
    service.versions
      .mockReturnValueOnce(oldVersions.promise)
      .mockResolvedValue({ success: true, data: [], total: 0 });
    service.events
      .mockReturnValueOnce(oldEvents.promise)
      .mockResolvedValue({ success: true, data: [], total: 0 });
    renderWithClient(actions());
    fireEvent.click(screen.getByRole('button', { name: /版本与事件/ }));
    await waitFor(() => expect(service.versions).toHaveBeenCalledOnce());
    const versionSignal = service.versions.mock.calls[0][1]
      .signal as AbortSignal;
    const eventSignal = service.events.mock.calls[0][1].signal as AbortSignal;
    fireEvent.click(screen.getByRole('button', { name: /Close/i }));
    await waitFor(() => expect(versionSignal.aborted).toBe(true));
    expect(eventSignal.aborted).toBe(true);
    fireEvent.click(screen.getByRole('button', { name: /版本与事件/ }));
    await waitFor(() => expect(service.versions).toHaveBeenCalledTimes(2));
    expect(service.versions).toHaveBeenLastCalledWith(
      { orderId: 'order-1', page: 1, pageSize: 200 },
      expect.objectContaining({ signal: expect.anything() }),
    );
    await act(async () => {
      oldVersions.resolve({ data: [] });
      oldEvents.resolve({ data: [] });
    });
  });
});
