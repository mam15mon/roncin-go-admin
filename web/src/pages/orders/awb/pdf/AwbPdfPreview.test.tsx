import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor } from '@testing-library/react';
import { useEffect } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { awbPdfSample } from './AwbPdfDocument.fixtures';
import AwbPdfPreview, { type AwbPdfPreviewProps } from './AwbPdfPreview';

const state = vi.hoisted(() => ({
  pdf: { url: 'about:blank#awb', loading: false, error: null as Error | null },
  generated: vi.fn(),
  cleanup: vi.fn(),
}));

vi.mock('@react-pdf/renderer', () => ({
  Font: { register: vi.fn() },
  usePDF: (options: unknown) => {
    state.generated(options);
    useEffect(() => state.cleanup, []);
    return [state.pdf];
  },
}));
vi.mock('./AwbPdfDocument', () => ({ AwbPdfDocument: () => null }));

const initial: AwbPdfPreviewProps = {
  open: true,
  onClose: vi.fn(),
  model: awbPdfSample,
};
beforeEach(() => {
  vi.clearAllMocks();
  state.pdf = { url: 'about:blank#awb', loading: false, error: null };
});

describe('航空 PDF Blob 生命周期', () => {
  it('预览下载打印共用同URL，关闭卸载回收', async () => {
    const view = renderWithApp(<AwbPdfPreview {...initial} />);
    expect(
      await screen.findByTitle('HAWB SAMPLE-HAWB-001 PDF 预览'),
    ).toHaveAttribute('src', 'about:blank#awb');
    for (const label of [/下载 PDF/, /打开打印/])
      expect(screen.getByRole('link', { name: label })).toHaveAttribute(
        'href',
        'about:blank#awb',
      );
    view.rerender(<AwbPdfPreview {...initial} open={false} />);
    await waitFor(() => expect(state.cleanup).toHaveBeenCalledOnce());
    expect(
      screen.queryByRole('link', { name: /下载 PDF/ }),
    ).not.toBeInTheDocument();
  });
  it('已保存正文、版本、真实主体和上层指纹变化分别销毁旧实例，失败不输出旧URL且可重试', async () => {
    const view = renderWithApp(<AwbPdfPreview {...initial} />);
    await screen.findByTitle('HAWB SAMPLE-HAWB-001 PDF 预览');
    for (const [index, model] of [
      { ...awbPdfSample, version: '8' },
      { ...awbPdfSample, content: { remarks: 'NEW CONTENT' } },
      { ...awbPdfSample, issuerDisplayName: 'OTHER ISSUER' },
    ].entries()) {
      view.rerender(<AwbPdfPreview {...initial} model={model} />);
      expect(state.cleanup).toHaveBeenCalledTimes(index + 1);
    }
    state.pdf = {
      url: 'about:blank#old',
      loading: false,
      error: new Error('失败'),
    };
    view.rerender(
      <AwbPdfPreview {...initial} fingerprint="template-updated" />,
    );
    expect(await screen.findByText('PDF 生成失败，请重试。')).toBeVisible();
    expect(
      screen.queryByRole('link', { name: /打开打印/ }),
    ).not.toBeInTheDocument();
    state.pdf = { url: 'about:blank#retry', loading: false, error: null };
    fireEvent.click(screen.getByRole('button', { name: /重新生成/ }));
    expect(
      await screen.findByTitle('HAWB SAMPLE-HAWB-001 PDF 预览'),
    ).toHaveAttribute('src', 'about:blank#retry');
    expect(state.cleanup).toHaveBeenCalledTimes(5);
  });
  it('在生成中或主体消失时不保留旧下载，不支持字符明示且保留输入模型', async () => {
    state.pdf = { url: 'about:blank#old', loading: true, error: null };
    const view = renderWithApp(<AwbPdfPreview {...initial} />);
    expect(screen.getByRole('button', { name: /下载 PDF/ })).toBeDisabled();
    expect(
      screen.queryByTitle('HAWB SAMPLE-HAWB-001 PDF 预览'),
    ).not.toBeInTheDocument();
    view.rerender(
      <AwbPdfPreview
        {...initial}
        model={{ ...awbPdfSample, issuerDisplayName: '' }}
      />,
    );
    expect(
      await screen.findByText('缺少真实签发主体资料，请补齐后输出。'),
    ).toBeVisible();
    await waitFor(() => expect(state.cleanup).toHaveBeenCalledOnce());
    view.rerender(
      <AwbPdfPreview
        {...initial}
        model={{ ...awbPdfSample, content: { remarks: '😀' } }}
      />,
    );
    expect(
      await screen.findByText(/单证备注含暂不支持打印的字符/),
    ).toBeVisible();
    expect(
      screen.queryByRole('link', { name: /下载 PDF/ }),
    ).not.toBeInTheDocument();
  });
});
