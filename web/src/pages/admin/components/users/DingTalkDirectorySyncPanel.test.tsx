import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, within } from '@testing-library/react';
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
          {
            userId: 'person-9',
            userName: '测试员工',
            departmentId: 'dept-1',
            departmentName: '操作部',
            message: '身份冲突',
          },
        ],
      },
    });
    renderPanel();
    expect(
      await screen.findByText('同步结束，存在 1 条失败记录'),
    ).toBeInTheDocument();
    expect(screen.getByText(/尚未确认全员授权范围/)).toBeInTheDocument();
    expect(screen.getByText('姓名、钉钉部门')).toBeInTheDocument();
    const failureRow = screen.getByText('身份冲突').closest('tr');
    if (!failureRow) throw new Error('未找到失败记录');
    const row = within(failureRow);
    expect(row.getByText('测试员工').closest('strong')).toBeInTheDocument();
    expect(row.getByText('操作部').closest('strong')).toBeInTheDocument();
    expect(row.getByText('ID：person-9')).toHaveClass(
      'ant-typography-secondary',
    );
    expect(row.getByText('ID：dept-1')).toHaveClass('ant-typography-secondary');
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

it('全企业授权存在失败时突出失败并展示具体缺失字段', async () => {
  api.latest.mockResolvedValue({
    data: {
      taskId: 'task-3',
      status: 'SUCCEEDED',
      fullScope: true,
      total: 13,
      unchanged: 13,
      failed: 14,
      failureTotal: 14,
      failures: [
        {
          userId: 'person-1',
          departmentId: '0',
          message:
            '钉钉人员详情接口未返回已授权字段：头像（字段缺失或为 null）',
        },
      ],
    },
  });
  renderPanel();
  expect(
    await screen.findByText('同步结束，存在 14 条失败记录'),
  ).toBeInTheDocument();
  expect(screen.queryByText('同步完成')).not.toBeInTheDocument();
  expect(screen.getByText('授权范围：全企业')).toBeInTheDocument();
  expect(
    screen.getByText(/人员详情接口未返回已授权字段：头像/),
  ).toBeInTheDocument();
  expect(screen.getByText('姓名未取得')).toBeInTheDocument();
  expect(screen.getByText('部门名称未取得')).toBeInTheDocument();
  expect(screen.getByText('ID：person-1')).toBeInTheDocument();
  expect(screen.queryByText('ID：0')).not.toBeInTheDocument();
  expect(screen.getByText(/与去重人员数不直接相加/)).toBeInTheDocument();
});

it('无邮箱人员同步成功时展示紧凑摘要，不新增缺邮箱警告', async () => {
  api.latest.mockResolvedValue({
    data: {
      taskId: 'task-4',
      status: 'SUCCEEDED',
      fullScope: true,
      total: 27,
      created: 0,
      updated: 27,
      unchanged: 0,
      failed: 0,
      completedAt: '2026-10-09T10:49:08Z',
      availableFields: ['display_name', 'departments', 'avatar_url', 'email'],
    },
  });
  renderPanel();
  expect(await screen.findByText('同步完成')).toBeInTheDocument();
  expect(screen.getByText('授权范围：全企业')).toBeInTheDocument();
  expect(screen.getByText('姓名、钉钉部门、头像、邮箱')).toBeInTheDocument();
  expect(screen.getAllByText('27')).toHaveLength(2);
  expect(screen.queryByRole('alert')).not.toBeInTheDocument();
  expect(screen.getByRole('button', { name: '同步说明' })).toBeInTheDocument();
});

it('部门读取失败明确无人员，不以部门标识冒充部门名称', async () => {
  api.latest.mockResolvedValue({
    data: {
      taskId: 'task-5',
      status: 'SUCCEEDED',
      fullScope: true,
      failed: 1,
      failureTotal: 1,
      failures: [
        {
          departmentId: 'dept-2',
          departmentName: '财务部',
          message: '读取部门人员失败',
        },
      ],
    },
  });
  renderPanel();
  expect(await screen.findByText('读取部门人员失败')).toBeInTheDocument();
  expect(screen.getByText('无人员')).toBeInTheDocument();
  expect(screen.getByText('财务部')).toBeInTheDocument();
  expect(screen.getByText('ID：dept-2')).toBeInTheDocument();
  expect(screen.queryByText('姓名未取得')).not.toBeInTheDocument();
});

it('整体接口失败仍提示服务端原因，不标成成功或邮箱空值', async () => {
  api.latest.mockResolvedValue({
    data: {
      taskId: 'task-6',
      status: 'DEAD_LETTER',
      fullScope: true,
      lastError: '钉钉通讯录接口暂不可用，请重试',
    },
  });
  renderPanel();
  expect(await screen.findByRole('alert')).toHaveTextContent(
    '钉钉通讯录接口暂不可用，请重试',
  );
  expect(screen.queryByText('同步完成')).not.toBeInTheDocument();
  expect(screen.getByRole('alert')).toHaveTextContent('同步已停止');
});
