import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { act } from '@testing-library/react';
import { App } from 'antd';
import React from 'react';
import { afterEach, describe, expect, it } from 'vitest';
import { appInitialStateContext } from '@/app/initialStateContext';
import {
  componentProps,
  FinanceCommissionsPage,
  resetCommissionPageMocks,
} from './index.fixtures';

describe('提成台账筛选设置接入', () => {
  afterEach(() => {
    resetCommissionPageMocks();
    window.localStorage.clear();
  });

  it('按用户+公司+页面读取偏好并只渲染可见字段', async () => {
    window.localStorage.setItem(
      'roncin:filter-settings:v1:finance:commissions:u1:o1',
      JSON.stringify({
        visible: ['keyword', 'status'],
        hidden: ['commissionMonth'],
      }),
    );
    renderWithClient(
      <App>
        <appInitialStateContext.Provider
          value={
            {
              initialState: {
                currentUser: { id: 'u1', currentOrganization: { id: 'o1' } },
              },
            } as never
          }
        >
          <FinanceCommissionsPage />
        </appInitialStateContext.Provider>
      </App>,
    );
    await act(async () => {});
    const props = componentProps.searchFilter as unknown as {
      visibleNames: string[];
    };
    expect(props.visibleNames).toEqual(['keyword', 'status']);
  });
});
