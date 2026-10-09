import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen } from '@testing-library/react';
import React from 'react';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

const api = vi.hoisted(() => ({
  latest: vi.fn(),
  get: vi.fn(),
  start: vi.fn(),
  reload: vi.fn(),
}));
vi.mock('@/services/roncin/adminService', () => ({
  adminServiceGetLatestDingTalkDirectorySync: api.latest,
  adminServiceGetDingTalkDirectorySync: api.get,
  adminServiceStartDingTalkDirectorySync: api.start,
}));

import DingTalkDirectorySyncPanel from './DingTalkDirectorySyncPanel';

function renderPanel() {
  return renderWithClient(
    <DingTalkDirectorySyncPanel
      actionRef={{ current: { reload: api.reload } as any }}
    />,
  );
}
beforeEach(() => {
  Object.values(api).forEach((mock) => {
    mock.mockReset();
  });
});
afterEach(() => vi.useRealTimers());
describe('钉钉通讯录同步面板', () => {
  it('部分授权及失败摘要如实显示，并按分页请求失败详情', async () => {
    api.latest.mockResolvedValue({
      data: {
        taskId: 'task-1',
        status: 'SUCCEEDED',
        fullScope: false,
        scopeSummary: '仅授权操作部',
        total: 9,
        created: 8,
        failed: 1,
        availableFields: ['display_name', 'departments'],
        failureTotal: 21,
        failures: [
          { userId: 'person-9', departmentId: 'dept-1', message: '身份冲突' },
        ],
      },
    });
    renderPanel();
    expect(await screen.findByText('同步完成')).toBeInTheDocument();
    expect(screen.getByText(/尚未确认全员授权范围/)).toBeInTheDocument();
    expect(screen.getByText('姓名、钉钉部门')).toBeInTheDocument();
    expect(screen.getByText('身份冲突')).toBeInTheDocument();
    fireEvent.click(screen.getByTitle('2'));
    await act(async () => {});
    expect(api.latest).toHaveBeenLastCalledWith({ page: 2, pageSize: 20 });
  });
  it('运行任务自动刷新，进入终态后停止轮询并刷新人员列表', async () => {
    vi.useFakeTimers();
    api.latest
      .mockResolvedValueOnce({
        data: { taskId: 'task-2', status: 'RUNNING', fullScope: true },
      })
      .mockResolvedValue({
        data: {
          taskId: 'task-2',
          status: 'SUCCEEDED',
          fullScope: true,
          created: 2,
        },
      });
    renderPanel();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(1);
    });
    expect(api.latest).toHaveBeenCalledTimes(1);
    await act(async () => {
      await vi.advanceTimersByTimeAsync(2100);
    });
    expect(api.latest).toHaveBeenCalledTimes(2);
    expect(screen.getByText('同步完成')).toBeInTheDocument();
    expect(api.reload).toHaveBeenCalledOnce();
    await act(async () => {
      await vi.advanceTimersByTimeAsync(10000);
    });
    expect(api.latest).toHaveBeenCalledTimes(2);
  });
});
