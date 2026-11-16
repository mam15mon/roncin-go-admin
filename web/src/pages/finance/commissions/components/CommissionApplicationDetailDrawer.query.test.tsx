import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { beforeEach, expect, it, vi } from 'vitest';
import CommissionApplicationDetailDrawer from './CommissionApplicationDetailDrawer';

const getApplication = vi.hoisted(() => vi.fn());
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceGetCommissionApplication: getApplication,
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({ canOperateOrganization: () => true }),
}));

function drawer(applicationId: string) {
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <CommissionApplicationDetailDrawer
        open
        applicationId={applicationId}
        canManage={false}
        onClose={vi.fn()}
        onApprove={vi.fn()}
        onReject={vi.fn()}
      />
    </ConfigProvider>
  );
}

beforeEach(() => {
  getApplication.mockReset();
});

it.each([
  undefined,
  { success: false, data: {} },
  { success: true, data: {} },
  { success: true, data: { application: {} } },
  {
    success: true,
    data: { application: { id: 'application-a' }, lines: {} },
  },
])('无有效成功详情不展示零笔空明细，可重试恢复', async (failed) => {
  getApplication.mockResolvedValueOnce(failed).mockResolvedValueOnce({
    success: true,
    data: {
      application: {
        id: 'application-a',
        employeeName: '员工甲',
        commissionCount: 0,
      },
    },
  });
  renderWithClient(drawer('application-a'));
  expect(await screen.findByText('申请批次明细加载失败')).toBeInTheDocument();
  expect(screen.queryByText('笔数')).not.toBeInTheDocument();
  fireEvent.click(screen.getByRole('button', { name: /重\s*试/ }));
  expect(await screen.findByText('员工甲')).toBeInTheDocument();
  expect(screen.getByText('笔数')).toBeInTheDocument();
  expect(screen.queryByText('申请批次明细加载失败')).not.toBeInTheDocument();
});

it('换申请时取消旧查询，迟到结果不污染新申请', async () => {
  let resolveOld: ((value: unknown) => void) | undefined;
  let oldSignal: AbortSignal | undefined;
  getApplication.mockImplementation(({ id }, options) => {
    expect(options.skipErrorHandler).toBe(true);
    if (id === 'application-a') {
      oldSignal = options.signal;
      return new Promise((resolve) => {
        resolveOld = resolve;
      });
    }
    return Promise.resolve({
      success: true,
      data: { application: { id, employeeName: '员工乙' } },
    });
  });
  const view = renderWithClient(drawer('application-a'));
  await waitFor(() => expect(oldSignal).toBeDefined());
  view.rerender(drawer('application-b'));
  expect(oldSignal?.aborted).toBe(true);
  expect(await screen.findByText('员工乙')).toBeInTheDocument();
  await act(async () => {
    resolveOld?.({
      success: true,
      data: { application: { id: 'application-a', employeeName: '旧员工甲' } },
    });
  });
  expect(screen.queryByText('旧员工甲')).not.toBeInTheDocument();
  expect(screen.getByText('员工乙')).toBeInTheDocument();
});
