import { renderWithApp } from '@root/tests/renderWithApp';
import { act, fireEvent, screen } from '@testing-library/react';
import React from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { AuthOrganizationKind } from '@/enums.generated';
import {
  ProfitReportPage,
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

describe('报表工作区、官方采纳与独立导出', () => {
  it('只有报表权限也能查询当前公司，request完成不提前更新完整合计', async () => {
    renderWithApp(<ProfitReportPage />, { user });
    let page:
      | Awaited<ReturnType<NonNullable<typeof reportMocks.template>['request']>>
      | undefined;
    await act(async () => {
      page = await reportMocks.template?.request({ current: 1, pageSize: 1 });
    });
    expect(reportMocks.list).toHaveBeenCalledWith(
      expect.objectContaining({
        organizationId: 'company-a',
        page: 1,
        pageSize: 1,
      }),
      expect.anything(),
    );
    expect(screen.queryByText('完整筛选合计')).not.toBeInTheDocument();
    act(() => {
      if (page)
        reportMocks.template?.onAcceptedSnapshot({
          rows: page.data,
          total: page.total,
          summary: page.summary,
        });
    });
    expect(screen.getByText('完整筛选合计')).toBeInTheDocument();
    expect(screen.getByText('4,240.00')).toBeInTheDocument();
    expect(screen.getByText('2 张订单 · 4 笔有效费用')).toBeInTheDocument();
    await act(async () => {});
  });

  it.each([undefined, AuthOrganizationKind.ORGANIZATION_KIND_SYSTEM])(
    '身份缺失/非公司 %s 不发经营报表请求',
    async (kind) => {
      renderWithApp(<ProfitReportPage />, {
        user: kind
          ? { ...user, currentOrganization: { id: 'system', kind } }
          : undefined,
      });
      let page: { success?: boolean } | undefined;
      await act(async () => {
        page = await reportMocks.template?.request({
          current: 1,
          pageSize: 20,
        });
      });
      expect(page?.success).toBe(false);
      expect(reportMocks.list).not.toHaveBeenCalled();
    },
  );

  it('完整导出读取被采纳筛选且不带分页，金额使用服务端完整结果', async () => {
    const create = vi
      .spyOn(URL, 'createObjectURL')
      .mockReturnValue('blob:report');
    const revoke = vi
      .spyOn(URL, 'revokeObjectURL')
      .mockImplementation(() => {});
    vi.spyOn(HTMLAnchorElement.prototype, 'click').mockImplementation(() => {});
    renderWithApp(<ProfitReportPage />, { user });
    await act(async () => {
      const page = await reportMocks.template?.request({
        current: 3,
        pageSize: 1,
      });
      if (page)
        reportMocks.template?.onAcceptedSnapshot({
          rows: page.data,
          total: page.total,
          summary: page.summary,
        });
    });
    await act(async () => {
      fireEvent.click(screen.getByRole('button', { name: /导出全部筛选结果/ }));
    });
    expect(reportMocks.export).toHaveBeenCalledWith(
      expect.objectContaining({ organizationId: 'company-a' }),
      expect.anything(),
    );
    const query = reportMocks.export.mock.calls[0][0];
    expect(query.page).toBeUndefined();
    expect(query.pageSize).toBeUndefined();
    expect(query.dateFrom).toBe(reportMocks.list.mock.calls[0][0].dateFrom);
    expect(create).toHaveBeenCalledTimes(1);
    expect(revoke).toHaveBeenCalledWith('blob:report');
    create.mockRestore();
    revoke.mockRestore();
  });
});
