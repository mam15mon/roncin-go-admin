import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen } from '@testing-library/react';
import type { MessageInstance } from 'antd/es/message/interface';
import { expect, it, vi } from 'vitest';
import { resourceTabs } from '../resourceConstants';
import ResourceImportModal from './ResourceImportModal';

it.each([
  {
    canOverwrite: false,
    conflictCount: 0,
    overwriteAllowed: false,
    enabled: true,
  },
  {
    canOverwrite: false,
    conflictCount: 1,
    overwriteAllowed: true,
    enabled: false,
  },
  {
    canOverwrite: true,
    conflictCount: 1,
    overwriteAllowed: true,
    enabled: true,
  },
  {
    canOverwrite: true,
    conflictCount: 2,
    overwriteAllowed: false,
    enabled: false,
  },
])(
  '覆盖权限 $canOverwrite、冲突 $conflictCount、服务端允许 $overwriteAllowed',
  ({ canOverwrite, conflictCount, overwriteAllowed, enabled }) => {
    const commit = vi.fn();
    const active = resourceTabs.find((tab) => tab.type === 5);
    if (!active) throw new Error('缺少发货人资源页签');
    renderWithApp(
      <ResourceImportModal
        active={active}
        open
        setOpen={vi.fn()}
        files={[]}
        setFiles={vi.fn()}
        rows={[{ shortName: '测试企业', resourceType: 5 }]}
        setRows={vi.fn()}
        preview={{
          success: true,
          validCount: 1,
          conflictCount,
          overwriteAllowed,
        }}
        setPreview={vi.fn()}
        canOverwrite={canOverwrite}
        loading={false}
        onPreview={vi.fn()}
        onCommit={commit}
        onDownloadTemplate={vi.fn()}
        message={{ error: vi.fn() } as unknown as MessageInstance}
      />,
    );
    const button = screen.getByRole('button', {
      name: conflictCount ? `确认覆盖 ${conflictCount} 条` : '确认导入',
    });
    if (enabled) expect(button).toBeEnabled();
    else expect(button).toBeDisabled();
    fireEvent.click(button);
    expect(commit).toHaveBeenCalledTimes(enabled ? 1 : 0);
    if (!canOverwrite && conflictCount) {
      expect(
        screen.getByText(
          '覆盖已有资源需要编辑资源权限，请联系有权限的人员处理。',
        ),
      ).toBeInTheDocument();
      expect(
        screen.queryByText(/存在一行匹配多个资源的歧义/),
      ).not.toBeInTheDocument();
    }
  },
);
