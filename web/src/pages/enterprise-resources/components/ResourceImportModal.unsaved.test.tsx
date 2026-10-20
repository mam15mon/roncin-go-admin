import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, type UploadFile } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';
import { useState } from 'react';
import { beforeEach, expect, it, vi } from 'vitest';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import { resourceTabs } from '../resourceConstants';
import ResourceImportModal from './ResourceImportModal';

vi.mock('../resourceConstants', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../resourceConstants')>()),
  parseImportFile: vi.fn(() => [{ shortName: '导入资源' }]),
}));
function Harness({ open = true }: { open?: boolean }) {
  const [files, setFiles] = useState<UploadFile[]>([]);
  const [rows, setRows] = useState<API.EnterpriseResourceInput[]>([]);
  const [preview, setPreview] =
    useState<API.PreviewEnterpriseResourceImportResponse>();
  return (
    <App>
      <ResourceImportModal
        active={resourceTabs[0]}
        open={open}
        setOpen={vi.fn()}
        files={files}
        setFiles={setFiles}
        rows={rows}
        setRows={setRows}
        preview={preview}
        setPreview={setPreview}
        loading={false}
        onPreview={vi.fn()}
        onCommit={vi.fn()}
        onDownloadTemplate={vi.fn()}
        message={{ error: vi.fn() } as unknown as MessageInstance}
      />
    </App>
  );
}
beforeEach(() =>
  window.history.replaceState({}, '', '/enterprise-resources/config'),
);
it('CSV真实载入待导入行后保护，关闭清理且初始化不脏', async () => {
  const view = render(<Harness />);
  expect(getAnyDirtyTabKey()).toBeUndefined();
  const input = document.querySelector('input[type="file"]');
  if (!input) throw new Error('缺少CSV文件输入');
  const file = new File(['data'], 'resources.csv');
  Object.defineProperty(file, 'text', { value: async () => '简称\n导入资源' });
  await act(async () => {
    fireEvent.change(input, { target: { files: [file] } });
  });
  await screen.findByText('已读取 1 条数据，请先校验');
  expect(getAnyDirtyTabKey()).toBe('/enterprise-resources/config');
  view.rerender(<Harness open={false} />);
  await waitFor(() => expect(getAnyDirtyTabKey()).toBeUndefined());
});
