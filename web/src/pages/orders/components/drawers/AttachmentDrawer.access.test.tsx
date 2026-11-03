import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import {
  api,
  attachment,
  fetchMock,
  openDrawer,
  revokeURL,
  selectFile,
  submit,
} from './AttachmentDrawer.fixtures';

describe('订单附件访问与生命周期', () => {
  it('预览按受控 MIME 读取且关闭时释放 Blob URL，无管理权限也可查看', async () => {
    api.list.mockResolvedValueOnce({
      data: [{ ...attachment, mimeType: 'image/png' }],
    });
    await openDrawer(false);
    expect(
      screen.queryByRole('button', { name: /上传附件/ }),
    ).not.toBeInTheDocument();
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /^预\s*览$/ }));
    });
    const image = await screen.findByRole(
      'img',
      { name: '提单.pdf' },
      { timeout: 5000 },
    );
    expect(image).toHaveAttribute('src', 'blob:attachment-preview');
    expect(fetchMock).toHaveBeenCalledWith(
      'https://oss.test/read',
      expect.objectContaining({ credentials: 'omit' }),
    );
    const closeButtons = screen.getAllByRole('button', { name: '关闭' });
    fireEvent.click(closeButtons.at(-1) as HTMLElement);
    await waitFor(() =>
      expect(revokeURL).toHaveBeenCalledWith('blob:attachment-preview'),
    );
  });

  it('换单后忽略迟到的准备上传结果，不上传或登记到新订单', async () => {
    const { container, ref } = await openDrawer();
    let finish: (value: unknown) => void = () => {};
    api.prepare.mockImplementationOnce(
      () =>
        new Promise((resolve) => {
          finish = resolve;
        }),
    );
    await selectFile(container);
    await submit();
    await waitFor(() => expect(api.prepare).toHaveBeenCalled());
    await act(async () =>
      ref.current?.open({ id: 'order-2', orderNo: 'SE002' }),
    );
    await act(async () =>
      finish({
        data: {
          uploadUrl: 'https://oss.test/late',
          uploadToken: 'old-order-token',
        },
      }),
    );
    expect(fetchMock).not.toHaveBeenCalled();
    expect(api.register).not.toHaveBeenCalled();
    expect(screen.queryByText('上传订单附件')).not.toBeInTheDocument();
  });

  it('HTML/SVG 没有预览按钮，下载仍走权限链接', async () => {
    api.list.mockResolvedValue({
      data: [{ ...attachment, mimeType: 'image/svg+xml', fileName: '图.svg' }],
    });
    await openDrawer(false, '图.svg');
    expect(
      screen.queryByRole('button', { name: /^预\s*览$/ }),
    ).not.toBeInTheDocument();
    const click = vi
      .spyOn(HTMLAnchorElement.prototype, 'click')
      .mockImplementation(() => {});
    fireEvent.click(screen.getByRole('button', { name: /^下\s*载$/ }));
    await waitFor(() => expect(click).toHaveBeenCalled());
    expect(api.access.mock.calls[0][0]).toEqual({
      orderId: 'order-1',
      id: 'attachment-1',
      download: true,
    });
    click.mockRestore();
  });
});
