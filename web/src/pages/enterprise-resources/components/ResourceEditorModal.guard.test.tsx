import {
  act,
  fireEvent,
  render,
  screen,
  waitFor,
} from '@testing-library/react';
import { App, Form, type UploadFile } from 'antd';
import type { MessageInstance } from 'antd/es/message/interface';
import { useState } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { getAnyDirtyTabKey } from '@/components/layout/tabCloseGuard';
import { resourceTabs } from '../resourceConstants';
import ResourceEditorModal from './ResourceEditorModal';

vi.mock('../resourceConstants', async (importOriginal) => ({
  ...(await importOriginal<typeof import('../resourceConstants')>()),
  imageChecksum: vi.fn().mockResolvedValue('checksum'),
}));
vi.mock('@/services/roncin/enterpriseResourceService', () => ({
  enterpriseResourceServicePrepareEnterpriseResourceImageUpload: vi.fn(
    () => new Promise(() => {}),
  ),
}));

const message = { error: vi.fn() } as unknown as MessageInstance;
function Harness({ open = true }: { open?: boolean }) {
  const [form] = Form.useForm();
  const [imageFiles, setImageFiles] = useState<UploadFile[]>([]);
  const active = resourceTabs.find((tab) => tab.type === 3);
  if (!active) throw new Error('缺少图片类型');
  return (
    <App>
      <ResourceEditorModal
        active={active}
        open={open}
        setOpen={vi.fn()}
        saving={false}
        form={form}
        onSave={vi.fn()}
        imageFiles={imageFiles}
        setImageFiles={setImageFiles}
        capabilities={{ imageMaxFileSize: '10485760' }}
        partnerOptions={[]}
        searchPartners={vi.fn()}
        assigneeOptions={[]}
        searchAssignees={vi.fn()}
        provinceOptions={[]}
        cityOptions={[]}
        districtOptions={[]}
        tagGroups={[]}
        message={message}
      />
    </App>
  );
}

describe('企业资源未保存输入保护', () => {
  beforeEach(() =>
    window.history.replaceState({}, '', '/enterprise-resources/config'),
  );

  it('初始 Form 默认值不标脏，图片外部 state 的用户上传标脏，关闭清理', async () => {
    const view = render(<Harness />);
    expect(getAnyDirtyTabKey()).toBeUndefined();
    const fileInput = document.querySelector('input[type="file"]');
    if (!fileInput) throw new Error('缺少图片上传入口');
    await act(async () => {
      fireEvent.change(fileInput, {
        target: {
          files: [new File(['image'], 'test.png', { type: 'image/png' })],
        },
      });
    });
    await waitFor(() =>
      expect(getAnyDirtyTabKey()).toBe('/enterprise-resources/config'),
    );
    view.rerender(<Harness open={false} />);
    expect(getAnyDirtyTabKey()).toBeUndefined();
  });

  it('实际名称输入标脏，卸载移除本表单守卫', () => {
    const view = render(<Harness />);
    fireEvent.change(screen.getByLabelText('简称/名称'), {
      target: { value: '待保存图片资料' },
    });
    expect(getAnyDirtyTabKey()).toBe('/enterprise-resources/config');
    view.unmount();
    expect(getAnyDirtyTabKey()).toBeUndefined();
  });
});
