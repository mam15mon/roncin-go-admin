import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import React from 'react';
import { Link, Route, Routes } from 'react-router';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { useInitialState } from '@/app/AppProvider';
import { OrganizationWorkspace } from '@/components/layout/OrganizationWorkspace';
import Workbench from './index';

const mocks = vi.hoisted(() => ({
  overview: vi.fn(),
  fees: vi.fn(),
  abnormal: vi.fn(),
}));
vi.mock('@/components/layout/TagsView', () => ({ TagsView: () => null }));
vi.mock('@/services/roncin/workbenchService', async (importOriginal) => ({
  ...(await importOriginal<
    typeof import('@/services/roncin/workbenchService')
  >()),
  workbenchServiceGetWorkbenchOverview: mocks.overview,
  workbenchServiceListMyUnbilledFees: mocks.fees,
  workbenchServiceListMyOpenAbnormalCases: mocks.abnormal,
}));

const user: API.CurrentUser = {
  id: 'employee-a',
  currentOrganization: { id: 'company-a', name: '公司甲', kind: 2 },
  permissionCapabilities: ['read', 'fee.read', 'abnormal_case.read'].map(
    (operation) => ({
      key: `business.order.se.${operation}`,
      dataScope: 'organization',
    }),
  ),
};

function deferred<T>() {
  let resolve!: (value: T) => void;
  const promise = new Promise<T>((done) => {
    resolve = done;
  });
  return { promise, resolve };
}

function TestWorkspace() {
  const { setInitialState } = useInitialState();
  return (
    <ConfigProvider theme={{ token: { motion: false } }}>
      <button
        type="button"
        onClick={() =>
          setInitialState({
            currentUser: {
              ...user,
              currentOrganization: { id: 'company-b', name: '公司乙', kind: 2 },
            },
          })
        }
      >
        切换公司
      </button>
      <Link to="/orders/sea-export/order-a/fees">模拟费用办理</Link>
      <OrganizationWorkspace>
        <Routes>
          <Route path="/welcome" element={<Workbench />} />
          <Route
            path="/orders/sea-export/order-a/fees"
            element={<Link to="/welcome">返回工作台</Link>}
          />
        </Routes>
      </OrganizationWorkspace>
    </ConfigProvider>
  );
}

function renderWorkspace() {
  return renderWithApp(<TestWorkspace />, {
    user,
    routerProps: { initialEntries: ['/welcome'] },
  });
}

beforeEach(() => {
  vi.clearAllMocks();
  mocks.overview.mockResolvedValue({
    success: true,
    data: { todos: { unbilledFeeCount: 1, openAbnormalCount: 1 } },
  });
  mocks.fees.mockResolvedValue({
    success: true,
    total: '1',
    data: [{ feeId: 'f-a', orderNo: '甲订单费用', feeName: '运费' }],
  });
  mocks.abnormal.mockResolvedValue({
    success: true,
    total: '1',
    data: [{ id: 'a-a', abnormalCaseName: '单证缺失' }],
  });
});

describe('工作台待办刷新与身份隔离', () => {
  it('摘要打开对应分类，关闭重开重取，返回工作台更新已经处理完的数量', async () => {
    renderWorkspace();
    fireEvent.click(
      (await screen.findAllByRole('button', { name: '查看明细' }))[0],
    );
    expect(await screen.findByText('甲订单费用')).toBeInTheDocument();
    expect(mocks.fees).toHaveBeenCalledTimes(1);
    await waitFor(() => expect(mocks.overview).toHaveBeenCalledTimes(2));
    fireEvent.click(screen.getByRole('button', { name: '关闭' }));
    fireEvent.click(screen.getAllByRole('button', { name: '查看明细' })[1]);
    expect(await screen.findByText('单证缺失')).toBeInTheDocument();
    expect(screen.getByRole('tab', { name: '进行中异常' })).toHaveAttribute(
      'aria-selected',
      'true',
    );
    fireEvent.click(screen.getByRole('button', { name: '关闭' }));
    mocks.fees.mockResolvedValue({ success: true, total: '0', data: [] });
    fireEvent.click(screen.getAllByRole('button', { name: '查看明细' })[0]);
    expect(await screen.findByText('当前没有此类待办')).toBeInTheDocument();
    expect(mocks.fees).toHaveBeenCalledTimes(2);
    fireEvent.click(screen.getByRole('button', { name: '关闭' }));
    fireEvent.click(screen.getByRole('link', { name: '模拟费用办理' }));
    expect(
      await screen.findByRole('link', { name: '返回工作台' }),
    ).toBeInTheDocument();
    mocks.overview.mockResolvedValue({ success: true, data: { todos: {} } });
    const beforeReturn = mocks.overview.mock.calls.length;
    fireEvent.click(screen.getByRole('link', { name: '返回工作台' }));
    await waitFor(() =>
      expect(mocks.overview).toHaveBeenCalledTimes(beforeReturn + 1),
    );
    await waitFor(() =>
      expect(screen.queryByTestId('todos-card')).not.toBeInTheDocument(),
    );
  });

  it('切换公司首帧销毁旧明细，旧列表迟到也不会覆盖新公司', async () => {
    const oldFees = deferred<API.ListMyUnbilledFeesResponse>();
    mocks.fees.mockReturnValueOnce(oldFees.promise);
    renderWorkspace();
    fireEvent.click(
      (await screen.findAllByRole('button', { name: '查看明细' }))[0],
    );
    await waitFor(() => expect(mocks.fees).toHaveBeenCalledTimes(1));
    const nextOverview = deferred<API.GetWorkbenchOverviewResponse>();
    mocks.overview.mockReturnValueOnce(nextOverview.promise);
    fireEvent.click(screen.getByRole('button', { name: '切换公司' }));
    expect(screen.queryByRole('dialog')).not.toBeInTheDocument();
    expect(screen.queryByTestId('todos-card')).not.toBeInTheDocument();
    await act(async () => {
      nextOverview.resolve({
        success: true,
        data: {
          hasCommissionEligibility: false,
          todos: { unbilledFeeCount: 1 },
        },
      });
    });
    mocks.fees.mockResolvedValue({
      success: true,
      total: '1',
      data: [{ feeId: 'f-b', orderNo: '乙订单费用' }],
    });
    fireEvent.click(
      (await screen.findAllByRole('button', { name: '查看明细' }))[0],
    );
    expect(await screen.findByText('乙订单费用')).toBeInTheDocument();
    await act(async () => {
      oldFees.resolve({
        success: true,
        total: '1',
        data: [{ feeId: 'f-a', orderNo: '旧公司的迟到费用' }],
      });
    });
    expect(screen.getByText('乙订单费用')).toBeInTheDocument();
    expect(screen.queryByText('旧公司的迟到费用')).not.toBeInTheDocument();
  });
});
