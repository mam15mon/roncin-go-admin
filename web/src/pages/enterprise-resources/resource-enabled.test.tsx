import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import EnterpriseResourcesPage from './index';

const state = vi.hoisted(() => ({
  list: vi.fn(),
  replace: vi.fn(),
  update: vi.fn(),
  remove: vi.fn(),
}));
vi.mock('@/router/history', () => ({
  history: {
    replace: state.replace,
    location: { pathname: '/enterprise-resources/config' },
  },
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canCreateEnterpriseResources: true,
    canUpdateEnterpriseResources: true,
    canDeleteEnterpriseResources: true,
  }),
}));
vi.mock('./useRegionOptions', () => ({
  useRegionOptions: () => ({
    provinceOptions: [],
    cityOptions: [],
    districtOptions: [],
  }),
}));
vi.mock(
  '@/services/roncin/enterpriseResourceService',
  async (importOriginal) => ({
    ...(await importOriginal<
      typeof import('@/services/roncin/enterpriseResourceService')
    >()),
    enterpriseResourceServiceGetEnterpriseResourceCapabilities: vi
      .fn()
      .mockResolvedValue({}),
    enterpriseResourceServiceListEnterpriseResources: state.list,
    enterpriseResourceServiceSearchEnterpriseResourcePartnerOptions: vi
      .fn()
      .mockResolvedValue({ data: [] }),
    enterpriseResourceServiceUpdateEnterpriseResource: state.update,
    enterpriseResourceServiceDeleteEnterpriseResource: state.remove,
  }),
);

const resources: Record<number, API.EnterpriseResource> = {
  1: {
    id: 'address-1',
    resourceType: 1,
    shortName: 'CODEX地址验收1109',
    address: { countryCode: 'CN', addressDetail: '上海测试路1号' },
  },
  2: {
    id: 'remark-1',
    resourceType: 2,
    shortName: '本类备注',
    enabled: true,
    remark: { remarkType: 4, content: '备注正文' },
  },
  6: {
    id: 'consignee-1',
    resourceType: 6,
    shortName: '本类收货人',
    enabled: true,
  },
};
const result = (type: number) => ({
  success: true,
  data: [resources[type]],
  total: 1,
});
function setup(tab = 'addresses') {
  return renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <EnterpriseResourcesPage />
    </ConfigProvider>,
    {
      routerProps: {
        initialEntries: [`/enterprise-resources/config?tab=${tab}`],
      },
    },
  );
}
beforeEach(() => {
  vi.clearAllMocks();
  state.list.mockImplementation(
    async ({ resourceType }: { resourceType: number }) => result(resourceType),
  );
});

describe('企业资源停用状态往返', () => {
  it.each([undefined, false, true])(
    '编辑状态 %s 按 wire 语义往返，不能误启用',
    async (enabled) => {
      state.list.mockResolvedValue({
        success: true,
        data: [
          {
            ...resources[1],
            ...(enabled === undefined ? {} : { enabled }),
          },
        ],
        total: 1,
      });
      state.update.mockResolvedValue({
        success: true,
        data: { id: 'address-1' },
      });
      setup();
      await screen.findByText('CODEX地址验收1109');
      fireEvent.click(screen.getByRole('button', { name: /编辑/ }));
      const dialog = await screen.findByRole('dialog');
      expect(within(dialog).getByRole('switch')).toHaveAttribute(
        'aria-checked',
        String(Boolean(enabled)),
      );
      fireEvent.change(within(dialog).getByLabelText('简称/名称'), {
        target: { value: '更新地址名称' },
      });
      fireEvent.click(within(dialog).getByRole('button', { name: /确.*定/ }));
      await waitFor(() =>
        expect(state.update).toHaveBeenCalledWith(
          { id: 'address-1' },
          expect.objectContaining({
            resource: expect.objectContaining({ enabled: Boolean(enabled) }),
          }),
        ),
      );
    },
  );
});
