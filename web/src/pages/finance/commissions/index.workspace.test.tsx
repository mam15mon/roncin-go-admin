import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { QueryClientProvider } from '@tanstack/react-query';
import { act, screen } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { beforeEach, expect, it } from 'vitest';
import {
  accessState,
  componentProps,
  FinanceCommissionsPage,
  resetCommissionPageMocks,
  serviceMocks,
  workspace,
} from './index.fixtures';

beforeEach(resetCommissionPageMocks);

it.each(['missing-user', 'missing-company', 'non-company'])(
  '%s 不加载提成或提供导出入口',
  async (scenario) => {
    if (scenario === 'missing-user') workspace.userId = undefined;
    if (scenario === 'missing-company') workspace.organizationId = undefined;
    if (scenario === 'non-company') workspace.canOperate = false;
    accessState.canExportFinanceCommissions = true;
    renderWithClient(
      <App>
        <FinanceCommissionsPage />
      </App>,
    );
    await act(async () => {
      expect(
        await componentProps.proTable?.request({ current: 1, pageSize: 20 }),
      ).toMatchObject({ success: false });
    });
    expect(serviceMocks.listCommissions).not.toHaveBeenCalled();
    expect(
      screen.queryByRole('button', { name: /导出提成/ }),
    ).not.toBeInTheDocument();
  },
);

it('旧筛选公司不能覆盖认证公司，重置和工作区切换保持唯一目标', async () => {
  serviceMocks.listCommissions.mockResolvedValue({ data: [], total: 0 });
  const { rerender, queryClient } = renderWithClient(
    <App>
      <FinanceCommissionsPage />
    </App>,
  );
  await act(async () => {
    componentProps.searchFilter?.onSearch({
      keyword: '业务条件',
      organizationId: 'other-company',
    });
    await componentProps.proTable?.request({ current: 2, pageSize: 40 });
  });
  expect(serviceMocks.listCommissions).toHaveBeenLastCalledWith(
    expect.objectContaining({ keyword: '业务条件', organizationId: 'org-a' }),
  );
  await act(async () => {
    componentProps.searchFilter?.onReset();
    await componentProps.proTable?.request({ current: 1, pageSize: 40 });
  });
  expect(serviceMocks.listCommissions).toHaveBeenLastCalledWith(
    expect.objectContaining({ organizationId: 'org-a' }),
  );
  workspace.organizationId = 'org-b';
  rerender(
    <QueryClientProvider client={queryClient}>
      <App>
        <FinanceCommissionsPage key="org-b" />
      </App>
    </QueryClientProvider>,
  );
  await act(async () => {
    await componentProps.proTable?.request({ current: 1, pageSize: 40 });
  });
  expect(serviceMocks.listCommissions).toHaveBeenLastCalledWith(
    expect.objectContaining({ organizationId: 'org-b' }),
  );
});
