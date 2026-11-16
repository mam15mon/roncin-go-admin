import { render, screen } from '@testing-library/react';
import { App } from 'antd';
import { expect, it, vi } from 'vitest';
import { OrderBusinessType, OrderFlowStatus } from '@/enums.generated';
import { buildOrderStatusSection } from './OrderStatusSection';

it.each([
  OrderBusinessType.BUSINESS_TYPE_SI,
  OrderBusinessType.BUSINESS_TYPE_AI,
])(
  '%s仅显示派生进口摘要，不开放出口订舱/配舱/放单的手动流转',
  (businessType) => {
    const section = buildOrderStatusSection(
      {
        businessType,
        flowStatus: OrderFlowStatus.ORDER_FLOW_STATUS_IMPORT_IN_PROGRESS,
        allowedTargetFlowStatuses: [OrderFlowStatus.ORDER_FLOW_STATUS_BOOKED],
      },
      { transitionFlow: vi.fn() },
    );
    render(
      <App>
        {section.extra}
        {section.content}
      </App>,
    );
    expect(screen.getByText('当前：进行中')).toBeVisible();
    expect(screen.getByText(/进口操作进度由本票承接服务/)).toBeVisible();
    expect(screen.queryByRole('button', { name: /流转到/ })).toBeNull();
    expect(screen.queryByText('已订舱')).toBeNull();
  },
);
