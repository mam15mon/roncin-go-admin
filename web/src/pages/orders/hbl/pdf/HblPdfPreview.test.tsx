import { renderWithApp } from '@root/tests/renderWithApp';
import { screen, waitFor } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { useEffect } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import HblPdfPreview, { type HblPdfPreviewProps } from './HblPdfPreview';

const state = vi.hoisted(() => ({
  // 单元测试使用可被 happy-dom 打开的本地占位地址；实际 Blob 渲染另做浏览器验收。
  pdf: {
    url: 'about:blank#hbl-pdf',
    loading: false,
    error: null as Error | null,
  },
  cleanup: vi.fn(),
  generated: vi.fn(),
}));

vi.mock('@react-pdf/renderer', () => ({
  Font: { register: vi.fn() },
  usePDF: (options: unknown) => {
    state.generated(options);
    useEffect(() => state.cleanup, []);
    return [state.pdf];
  },
}));
vi.mock('./HblPdfDocument', () => ({ HblPdfDocument: () => null }));

const initial: HblPdfPreviewProps = {
  open: true,
  onClose: vi.fn(),
  model: {
    houseNo: 'HBL-TEST-001',
    issuerNameEn: 'SAMPLE EAST LOGISTICS CO., LTD',
    content: { shipperText: '测试发货人' },
  },
  variant: 'draft',
};

function tree(props: HblPdfPreviewProps) {
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <HblPdfPreview {...props} />
    </ConfigProvider>
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  state.pdf = { url: 'about:blank#hbl-pdf', loading: false, error: null };
});

describe('HBL PDF 预览生命周期', () => {
  it('预览、下载和打印使用同一 PDF 地址，关闭会卸载生成实例', async () => {
    const view = renderWithApp(tree(initial));
    const frame = await screen.findByTitle('HBL HBL-TEST-001 PDF 预览');
    expect(frame).toHaveAttribute('src', 'about:blank#hbl-pdf');
    const download = screen.getByRole('link', { name: /下载 PDF/ });
    const print = screen.getByRole('link', { name: /打开打印/ });
    expect(download).toHaveAttribute('href', 'about:blank#hbl-pdf');
    expect(download).toHaveAttribute(
      'download',
      expect.stringContaining('draft'),
    );
    expect(print).toHaveAttribute('href', 'about:blank#hbl-pdf');
    expect(print).toHaveAttribute('target', '_blank');
    view.rerender(tree({ ...initial, open: false }));
    await waitFor(() => expect(state.cleanup).toHaveBeenCalledOnce());
    expect(
      screen.queryByTitle('HBL HBL-TEST-001 PDF 预览'),
    ).not.toBeInTheDocument();
  });

  it('新版本生成中不展示或输出上一次的文件，生成失败也不回退旧文件', async () => {
    const view = renderWithApp(tree(initial));
    await screen.findByTitle('HBL HBL-TEST-001 PDF 预览');
    state.pdf = { url: 'blob:old-draft', loading: true, error: null };
    view.rerender(tree({ ...initial, variant: 'surrendered' }));
    expect(
      screen.queryByTitle('HBL HBL-TEST-001 PDF 预览'),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByRole('link', { name: /下载 PDF/ }),
    ).not.toBeInTheDocument();
    expect(screen.getByRole('button', { name: /下载 PDF/ })).toBeDisabled();
    expect(state.cleanup).toHaveBeenCalledOnce();
    state.pdf = {
      url: 'blob:old-draft',
      loading: false,
      error: new Error('生成失败'),
    };
    view.rerender(tree({ ...initial, variant: 'surrendered' }));
    expect(await screen.findByText('PDF 生成失败，请重试。')).toBeVisible();
    expect(
      screen.queryByRole('link', { name: /打开打印/ }),
    ).not.toBeInTheDocument();
  });

  it('打印不支持的字符时标明字段，保留内容且不生成缺字 PDF', async () => {
    renderWithApp(
      tree({
        ...initial,
        model: {
          houseNo: 'HBL-TEST-001',
          issuerNameEn: initial.model.issuerNameEn,
          content: { shipperText: '联系人😀' },
        },
      }),
    );
    expect(
      await screen.findByText(
        '发货人含暂不支持打印的字符「😀」，请调整后重试。',
      ),
    ).toBeVisible();
    expect(state.generated).not.toHaveBeenCalled();
    expect(
      screen.queryByRole('link', { name: /下载 PDF/ }),
    ).not.toBeInTheDocument();
  });

  it('公司名被清空后卸载旧结果，提示补齐并阻止空抬头文件', async () => {
    const view = renderWithApp(tree(initial));
    await screen.findByTitle('HBL HBL-TEST-001 PDF 预览');
    state.generated.mockClear();
    view.rerender(
      tree({ ...initial, model: { ...initial.model, issuerNameEn: '' } }),
    );
    expect(
      await screen.findByText(
        '签发公司尚未维护英文名称，请在公司档案中补齐后输出。',
      ),
    ).toBeVisible();
    expect(state.generated).not.toHaveBeenCalled();
    expect(state.cleanup).toHaveBeenCalledOnce();
    expect(
      screen.queryByRole('link', { name: /下载 PDF/ }),
    ).not.toBeInTheDocument();
    expect(
      screen.queryByTitle('HBL HBL-TEST-001 PDF 预览'),
    ).not.toBeInTheDocument();
  });
});
