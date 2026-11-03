import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { createRef } from 'react';
import { afterEach, beforeEach, vi } from 'vitest';
import AttachmentDrawer, { type AttachmentDrawerRef } from './AttachmentDrawer';

export { api };

const api = vi.hoisted(() => ({
  list: vi.fn(),
  prepare: vi.fn(),
  register: vi.fn(),
  access: vi.fn(),
  remove: vi.fn(),
}));
vi.mock('@/services/roncin/orderAttachmentService', () => ({
  orderAttachmentServiceListAttachments: api.list,
  orderAttachmentServicePrepareAttachmentUpload: api.prepare,
  orderAttachmentServiceRegisterAttachment: api.register,
  orderAttachmentServiceGetAttachmentAccess: api.access,
  orderAttachmentServiceRemoveAttachmentReference: api.remove,
}));

export const attachment = {
  id: 'attachment-1',
  orderId: 'order-1',
  docType: '提单',
  fileName: '提单.pdf',
  mimeType: 'application/pdf',
  fileSize: '3',
};
export let fetchMock: ReturnType<typeof vi.fn>;
export let createURL = vi.fn(
  (_value: Blob | MediaSource) => 'blob:attachment-preview',
);
export let revokeURL = vi.fn((_url: string) => {});

beforeEach(() => {
  vi.clearAllMocks();
  api.list.mockResolvedValue({ data: [attachment] });
  api.prepare.mockResolvedValue({
    data: {
      uploadUrl: 'https://oss.test/upload',
      uploadToken: 'authorized-upload',
      headers: { 'Content-Type': 'application/pdf' },
      expiresAt: '2099-01-01T00:00:00Z',
    },
  });
  api.register.mockResolvedValue({ success: true, data: attachment });
  api.access.mockResolvedValue({ data: { url: 'https://oss.test/read' } });
  fetchMock = vi.fn().mockResolvedValue({
    ok: true,
    arrayBuffer: async () => new Uint8Array([1, 2, 3]).buffer,
  });
  vi.stubGlobal('fetch', fetchMock);
  createURL = vi.fn((_value: Blob | MediaSource) => 'blob:attachment-preview');
  revokeURL = vi.fn((_url: string) => {});
  vi.stubGlobal(
    'URL',
    class extends URL {
      static createObjectURL(obj: Blob | MediaSource): string {
        return createURL(obj) as string;
      }
      static revokeObjectURL(url: string): void {
        revokeURL(url);
      }
    },
  );
});
afterEach(() => {
  vi.unstubAllGlobals();
});

export async function openDrawer(
  canRegister = true,
  expectedFileName = '提单.pdf',
) {
  const ref = createRef<AttachmentDrawerRef>();
  const view = renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <AttachmentDrawer ref={ref} canRegister={canRegister} />
    </ConfigProvider>,
  );
  await act(async () => ref.current?.open({ id: 'order-1', orderNo: 'SE001' }));
  await screen.findByText(expectedFileName, {}, { timeout: 5000 });
  return { ref, ...view };
}

export async function selectFile(
  container: HTMLElement,
  file = new File(['abc'], '提单.pdf', { type: 'application/pdf' }),
) {
  fireEvent.click(screen.getByRole('button', { name: /上传附件/ }));
  await screen.findByLabelText('文档类型', {}, { timeout: 5000 });
  const input =
    document.querySelector('input[type="file"]') ??
    container.querySelector('input[type="file"]');
  if (!input) throw new Error('没有文件选择器');
  await act(async () => fireEvent.change(input, { target: { files: [file] } }));
  return file;
}

export async function submit() {
  await act(async () =>
    fireEvent.click(screen.getByRole('button', { name: /^上\s*传$/ })),
  );
}
