import { renderWithApp } from '@root/tests/renderWithApp';
import { screen, waitFor } from '@testing-library/react';
import { describe, expect, it, vi } from 'vitest';
import FeeTemplatesPanel from './FeeTemplatesPanel';

const mocks = vi.hoisted(() => ({
  list: vi.fn(),
  create: vi.fn(),
  update: vi.fn(),
  props: {} as Record<string, unknown>,
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    isSystemWorkspace: true,
    canCreateFeeSettings: true,
    canUpdateFeeSettings: true,
  }),
}));
vi.mock('@/features/master-data/currencies', () => ({
  getCurrencies: async () => [],
}));
vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListOptions: async () => ({ data: [] }),
}));
vi.mock('@/services/roncin/feeCatalogService', () => ({
  feeCatalogServiceListFeeSettingTemplates: mocks.list,
  feeCatalogServiceCreateFeeSettingTemplate: mocks.create,
  feeCatalogServiceUpdateFeeSettingTemplate: mocks.update,
  feeCatalogServiceListBillingUnits: async () => ({ data: [] }),
}));
vi.mock('@/components/ui', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/components/ui')>()),
  SettingTableTemplate: (props: Record<string, unknown>) => {
    mocks.props = props;
    return <div>目录表格</div>;
  },
}));

describe('系统初始费用目录', () => {
  it('分页读取模板，提交税务默认文本而不引用公司税务 ID', async () => {
    mocks.list.mockResolvedValue({
      data: [{ id: 'template', input: { feeCode: 'DOC', nameZh: '文件费' } }],
      total: 21,
    });
    renderWithApp(<FeeTemplatesPanel />);
    expect(screen.getByText(/后续修改不影响已有公司配置/)).toBeInTheDocument();
    await waitFor(() => expect(mocks.props.canCreate).toBe(true));
    const query = mocks.props.query as (
      params: Record<string, unknown>,
    ) => Promise<{ data: unknown[]; total: number }>;
    expect(await query({ current: 2, pageSize: 20 })).toEqual({
      data: [{ id: 'template', feeCode: 'DOC', nameZh: '文件费' }],
      total: 21,
    });
    expect(mocks.list).toHaveBeenCalledWith({ page: 2, pageSize: 20 });
    const input = {
      feeCode: 'DOC',
      nameZh: '文件费',
      taxableServiceName: '服务费',
      taxableServiceDefaultTaxRate: '6',
    };
    await (
      mocks.props.createItem as (
        input: API.FeeSettingTemplateInput,
      ) => Promise<unknown>
    )(input);
    expect(mocks.create).toHaveBeenCalledWith({ input });
    await (
      mocks.props.updateItem as (
        row: { id: string },
        input: API.FeeSettingTemplateInput,
      ) => Promise<unknown>
    )({ id: 'template' }, input);
    expect(mocks.update).toHaveBeenCalledWith(
      { id: 'template' },
      { id: 'template', input },
    );
  });
});
