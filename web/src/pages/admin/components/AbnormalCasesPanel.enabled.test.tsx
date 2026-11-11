import { render } from '@testing-library/react';
import { beforeEach, expect, it, vi } from 'vitest';
import { AbnormalCasesPanel } from './AbnormalCasesPanel';

const state = vi.hoisted(() => ({
  initialValues: undefined as
    | undefined
    | ((editing?: API.MasterDataItem) => {
        enabled: boolean;
        sortOrder: number;
      }),
}));
vi.mock('@/app/access', () => ({ useAccess: () => ({}) }));
vi.mock('@/components/ui', () => ({
  SettingTableTemplate: (props: {
    initialValues: typeof state.initialValues;
  }) => {
    state.initialValues = props.initialValues;
    return null;
  },
}));
beforeEach(() => {
  state.initialValues = undefined;
});

it.each([undefined, false, true])(
  '异常情况 enabled=%s 编辑值按 ProtoJSON 映射',
  (enabled) => {
    render(<AbnormalCasesPanel />);
    const result = state.initialValues?.({
      id: 'abnormal',
      code: 'WIRE',
      name: '异常情况',
      ...(enabled === undefined ? {} : { enabled }),
    });
    expect(result?.enabled).toBe(enabled === true);
    expect(result?.sortOrder).toBe(0);
    expect(state.initialValues?.()).toMatchObject({
      enabled: true,
      sortOrder: 100,
    });
  },
);
