import { renderWithApp } from '@root/tests/renderWithApp';
import { fireEvent, screen, waitFor, within } from '@testing-library/react';
import { ConfigProvider } from 'antd';
import type { ReactNode } from 'react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import EnterpriseResourcesPage from './index';

const state = vi.hoisted(() => ({ create: vi.fn(), reload: vi.fn() }));

vi.mock('@/app/access', () => ({
  useAccess: () => ({ canCreateEnterpriseResources: true }),
}));
vi.mock('@/components/ui', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/components/ui')>()),
  ResponsiveProTable: (props: {
    actionRef: { current: unknown };
    toolBarRender: () => ReactNode;
  }) => {
    props.actionRef.current = { reload: state.reload };
    return <div>{props.toolBarRender()}</div>;
  },
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
    enterpriseResourceServiceCreateEnterpriseResource: state.create,
  }),
);

beforeEach(() => {
  vi.clearAllMocks();
  state.create.mockResolvedValue({ success: true, data: { id: 'resource-1' } });
});

async function openAddress() {
  renderWithApp(
    <ConfigProvider theme={{ token: { motion: false } }}>
      <EnterpriseResourcesPage />
    </ConfigProvider>,
    {
      routerProps: {
        initialEntries: ['/enterprise-resources/config?tab=addresses'],
      },
    },
  );
  fireEvent.click(screen.getByRole('button', { name: /新建地址/ }));
  return screen.findByRole('dialog');
}

function fillAddress(dialog: HTMLElement) {
  fireEvent.change(within(dialog).getByLabelText('简称/名称'), {
    target: { value: '上海仓库' },
  });
  fireEvent.change(within(dialog).getByLabelText('详细地址'), {
    target: { value: '上海市测试路1号' },
  });
}

describe('企业资源表单提交', () => {
  it.each([
    { tab: 'addresses', name: '地址', errors: ['简称/名称', '详细地址'] },
    {
      tab: 'remarks',
      name: '备注',
      errors: ['简称/名称', '备注类型', '备注内容'],
    },
  ])(
    '$name 空提交只展示字段错误，补齐后仍能保存',
    async ({ tab, name, errors }) => {
      const unhandled = vi.fn();
      window.addEventListener('unhandledrejection', unhandled);
      try {
        renderWithApp(
          <ConfigProvider theme={{ token: { motion: false } }}>
            <EnterpriseResourcesPage />
          </ConfigProvider>,
          {
            routerProps: {
              initialEntries: [`/enterprise-resources/config?tab=${tab}`],
            },
          },
        );
        fireEvent.click(
          screen.getByRole('button', { name: new RegExp(`新建${name}`) }),
        );
        const dialog = await screen.findByRole('dialog');
        fireEvent.click(within(dialog).getByRole('button', { name: /确.*定/ }));
        for (const label of errors) {
          expect(
            await within(dialog).findByText(`请输入${label}`),
          ).toBeInTheDocument();
        }
        expect(state.create).not.toHaveBeenCalled();
        expect(state.reload).not.toHaveBeenCalled();
        expect(unhandled).not.toHaveBeenCalled();
        expect(dialog).toBeInTheDocument();

        if (tab === 'addresses') fillAddress(dialog);
        else {
          fireEvent.change(within(dialog).getByLabelText('简称/名称'), {
            target: { value: '操作提示' },
          });
          fireEvent.mouseDown(within(dialog).getByLabelText('备注类型'));
          fireEvent.click(await screen.findByText('订单备注'));
          fireEvent.change(within(dialog).getByLabelText('备注内容'), {
            target: { value: '请提前确认送货时间' },
          });
        }
        fireEvent.click(within(dialog).getByRole('button', { name: /确.*定/ }));
        await waitFor(() => expect(state.create).toHaveBeenCalledOnce());
        await waitFor(() =>
          expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
        );
        expect(state.create).toHaveBeenCalledWith({
          resource: expect.objectContaining({
            resourceType: tab === 'addresses' ? 1 : 2,
            ...(tab === 'addresses'
              ? {
                  address: expect.objectContaining({
                    addressDetail: '上海市测试路1号',
                  }),
                }
              : { remark: { remarkType: 4, content: '请提前确认送货时间' } }),
          }),
        });
        expect(state.reload).toHaveBeenCalledOnce();
        expect(unhandled).not.toHaveBeenCalled();
      } finally {
        window.removeEventListener('unhandledrejection', unhandled);
      }
    },
  );

  it('保存请求拒绝仍显示失败原因，保留弹窗和输入且允许重试', async () => {
    state.create.mockRejectedValueOnce(
      Object.assign(new Error('Request failed'), {
        response: { data: { message: '当前资源保存失败' } },
      }),
    );
    const dialog = await openAddress();
    fillAddress(dialog);
    fireEvent.click(within(dialog).getByRole('button', { name: /确.*定/ }));
    expect(await screen.findByText('当前资源保存失败')).toBeInTheDocument();
    expect(within(dialog).getByLabelText('简称/名称')).toHaveValue('上海仓库');
    expect(state.reload).not.toHaveBeenCalled();
    fireEvent.click(within(dialog).getByRole('button', { name: /确.*定/ }));
    await waitFor(() =>
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument(),
    );
    expect(state.create).toHaveBeenCalledTimes(2);
  });

  it('请求层已处理失败并返回空响应时，不提示成功或关闭输入', async () => {
    state.create.mockResolvedValueOnce(undefined);
    const dialog = await openAddress();
    fillAddress(dialog);
    fireEvent.click(within(dialog).getByRole('button', { name: /确.*定/ }));
    await waitFor(() => expect(state.create).toHaveBeenCalledOnce());
    expect(dialog).toBeInTheDocument();
    expect(within(dialog).getByLabelText('详细地址')).toHaveValue(
      '上海市测试路1号',
    );
    expect(state.reload).not.toHaveBeenCalled();
    expect(screen.queryByText('资源已创建')).not.toBeInTheDocument();
  });
});
