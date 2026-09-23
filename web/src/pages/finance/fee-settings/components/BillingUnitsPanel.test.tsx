import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen } from '@testing-library/react';
import { Form } from 'antd';
import type { ReactNode } from 'react';
import { describe, expect, it, vi } from 'vitest';
import BillingUnitsPanel from './BillingUnitsPanel';

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    isSystemWorkspace: true,
    canCreateFeeSettings: true,
    canUpdateFeeSettings: true,
  }),
}));
vi.mock('@/components/ui', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/components/ui')>()),
  SettingTableTemplate: (props: {
    initialValues: (editing?: API.BillingUnit) => Record<string, unknown>;
    renderFormItems: (editing?: API.BillingUnit) => ReactNode;
  }) => (
    <Form initialValues={props.initialValues(undefined)}>
      {props.renderFormItems(undefined)}
    </Form>
  ),
}));

describe('计费单位数量规则表单', () => {
  it('普通单位默认小数，箱型带入整数，手动选择后不被覆盖', () => {
    const { unmount } = renderWithApp(<BillingUnitsPanel />);
    const decimal = screen.getByRole('radio', { name: '允许小数' });
    const integer = screen.getByRole('radio', { name: '整数' });
    const container = screen.getByRole('switch', { name: '是否为箱型单位' });
    expect(decimal).toBeChecked();
    fireEvent.click(container);
    expect(integer).toBeChecked();
    fireEvent.click(decimal);
    fireEvent.click(container);
    fireEvent.click(container);
    expect(decimal).toBeChecked();
    unmount();

    renderWithApp(<BillingUnitsPanel />);
    expect(screen.getByRole('radio', { name: '允许小数' })).toBeChecked();
  });
});
