import { fireEvent, render, screen } from '@testing-library/react';
import { App, ConfigProvider, Form } from 'antd';
import { expect, it, vi } from 'vitest';
import { OrderClosureStatus, OrderTerminationStatus } from '@/enums.generated';
import { buildOrderStatusSection } from './OrderStatusSection';

it('内容表单禁用时，合法生命周期动作仍可操作，显式阻断的动作保持禁用', async () => {
  const complete = vi.fn();
  const cancel = vi.fn();
  const close = vi.fn();
  const renderSection = (closureDisabled: boolean) => {
    const section = buildOrderStatusSection(
      {
        terminationStatus:
          OrderTerminationStatus.ORDER_TERMINATION_STATUS_TERMINATING,
        closureStatus: OrderClosureStatus.ORDER_CLOSURE_STATUS_OPEN,
      },
      {
        completeTermination: { onClick: complete },
        cancelTermination: { onClick: cancel, disabled: true },
        close: { onClick: close, disabled: closureDisabled },
      },
    );
    return (
      <ConfigProvider theme={{ token: { motion: false } }}>
        <App>
          <Form disabled>
            {section.extra}
            {section.content}
          </Form>
        </App>
      </ConfigProvider>
    );
  };
  const { rerender } = render(renderSection(false));
  expect(screen.getByRole('button', { name: '完结订单' })).toBeEnabled();
  fireEvent.click(screen.getByRole('button', { name: '完结订单' }));
  expect(close).toHaveBeenCalledOnce();
  expect(screen.getByRole('button', { name: '选择退关操作' })).toBeEnabled();
  fireEvent.click(screen.getByRole('button', { name: '选择退关操作' }));
  fireEvent.click(await screen.findByRole('menuitem', { name: '完成退关' }));
  expect(complete).toHaveBeenCalledOnce();
  expect(cancel).not.toHaveBeenCalled();
  rerender(renderSection(true));
  expect(screen.getByRole('button', { name: '完结订单' })).toBeDisabled();
});
