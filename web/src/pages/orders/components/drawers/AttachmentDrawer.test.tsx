import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import {
  api,
  fetchMock,
  openDrawer,
  selectFile,
  submit,
} from './AttachmentDrawer.fixtures';

describe('订单附件上传', () => {
  it('直接选文件上传，不要求手填技术字段，成功后刷新', async () => {
    const { container } = await openDrawer();
    const file = await selectFile(container);
    expect(
      screen.queryByLabelText('对象存储键 (Object Key)'),
    ).not.toBeInTheDocument();
    await submit();
    await waitFor(() => expect(api.register).toHaveBeenCalled());
    expect(api.prepare.mock.calls[0][1]).toMatchObject({
      docType: '提单',
      fileName: '提单.pdf',
      fileSize: '3',
      checksum:
        'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad',
    });
    expect(fetchMock).toHaveBeenCalledWith(
      'https://oss.test/upload',
      expect.objectContaining({
        method: 'PUT',
        credentials: 'omit',
        body: file,
      }),
    );
    expect(api.register.mock.calls[0][1]).toEqual({
      orderId: 'order-1',
      uploadToken: 'authorized-upload',
    });
    await waitFor(() => expect(api.list.mock.calls.length).toBeGreaterThan(1));
  });

  it('上传失败保留文件与表单，重试可成功', async () => {
    const { container } = await openDrawer();
    fetchMock.mockRejectedValueOnce(new Error('不可显示的签名 URL'));
    await selectFile(container);
    await submit();
    expect(
      await screen.findByText('上传失败，文件和填写内容已保留，请重试。'),
    ).toBeInTheDocument();
    expect(screen.getByLabelText('文档类型')).toHaveValue('提单');
    expect(api.register).not.toHaveBeenCalled();
    await submit();
    await waitFor(() => expect(api.register).toHaveBeenCalledTimes(1));
  });

  it('文件上传成功但登记失败时重用授权，不重复上传', async () => {
    const { container } = await openDrawer();
    api.register.mockResolvedValueOnce(undefined);
    await selectFile(container);
    await submit();
    expect(
      await screen.findByText('保存附件失败，文件和填写内容已保留，请重试。'),
    ).toBeInTheDocument();
    await submit();
    await waitFor(() => expect(api.register).toHaveBeenCalledTimes(2));
    expect(api.prepare).toHaveBeenCalledTimes(1);
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });
});
