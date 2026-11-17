import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import {
  ProfitReportPage,
  profitResponse,
  reportMocks,
  resetReportMocks,
} from './ReportShell.fixtures';

const user: API.CurrentUser = {
  id: 'u',
  currentOrganization: {
    id: 'company-a',
    name: '公司甲',
    kind: AuthOrganizationKind.ORGANIZATION_KIND_COMPANY,
  },
  permissionCapabilities: [
    { key: 'system.finance.report.read', dataScope: 'organization' },
  ],
};

beforeEach(() => {
  resetReportMocks();
  localStorage.clear();
});
async function renderAccepted() {
  const view = renderWithApp(<ProfitReportPage />, { user });
  await act(async () => {
    const page = await reportMocks.template?.request({
      current: 1,
      pageSize: 1,
    });
    if (page)
      reportMocks.template?.onAcceptedSnapshot({
        rows: page.data,
        total: page.total,
        summary: page.summary,
      });
  });
  return view;
}

describe('报表导出迟到响应', () => {
  it('导出在途卸载后不下载、不显示迟到成功', async () => {
    let resolve: ((value: API.ListProfitReportResponse) => void) | undefined;
    reportMocks.export.mockImplementation(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    const download = vi.spyOn(URL, 'createObjectURL');
    const view = await renderAccepted();
    fireEvent.click(screen.getByRole('button', { name: /导出全部筛选结果/ }));
    view.unmount();
    await act(async () => {
      resolve?.(profitResponse);
    });
    expect(download).not.toHaveBeenCalled();
    download.mockRestore();
  });
  it('切换已提交筛选后，旧导出和旧列表结果均不更新当前页面', async () => {
    let resolve: ((value: API.ListProfitReportResponse) => void) | undefined;
    reportMocks.export.mockImplementation(
      () =>
        new Promise((r) => {
          resolve = r;
        }),
    );
    const download = vi.spyOn(URL, 'createObjectURL');
    await renderAccepted();
    const old = await reportMocks.template?.request({
      current: 2,
      pageSize: 1,
    });
    fireEvent.click(screen.getByRole('button', { name: /导出全部筛选结果/ }));
    fireEvent.change(screen.getByLabelText('综合搜索'), {
      target: { value: '新客户' },
    });
    fireEvent.click(screen.getByRole('button', { name: /查 询|查询/ }));
    await act(async () => {
      resolve?.(profitResponse);
      if (old)
        reportMocks.template?.onAcceptedSnapshot({
          rows: old.data,
          total: old.total,
          summary: old.summary,
        });
    });
    expect(download).not.toHaveBeenCalled();
    expect(screen.queryByText('完整筛选合计')).not.toBeInTheDocument();
    download.mockRestore();
  });
});
