import { renderWithClient } from '@root/tests/queryClientTestUtils';
import { App } from 'antd';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { settlementServiceListNettings } from '@/services/roncin/settlementService';
import FinanceNettingsPage from './index';

type LedgerProps = {
  topBar?: unknown;
  columns: { title?: unknown }[];
  request: (params: {
    current: number;
    pageSize: number;
    keyword?: string;
    organizationId?: string;
  }) => Promise<unknown>;
};
const state = vi.hoisted(() => ({
  organizationId: 'workspace-A' as string | undefined,
  company: true,
  ledger: undefined as LedgerProps | undefined,
}));
vi.mock('@/app/AppProvider', () => ({
  useInitialState: () => ({
    initialState: {
      currentUser: {
        id: 'user',
        currentOrganization: { id: state.organizationId },
      },
    },
  }),
}));
vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: (id?: string) =>
      state.company && id === state.organizationId,
  }),
}));
vi.mock('@/components/ui', () => ({
  ResponsiveTable: () => null,
  FinanceLedgerTemplate: (props: LedgerProps) => {
    state.ledger = props;
    return null;
  },
}));
vi.mock('@/services/roncin/settlementService', () => ({
  settlementServiceListNettings: vi
    .fn()
    .mockResolvedValue({ data: [], total: '0', matchedSummary: {} }),
}));

beforeEach(() => {
  vi.clearAllMocks();
  state.company = true;
  state.organizationId = 'workspace-A';
});
describe('nettings列表当前公司', () => {
  it('请求固定当前公司，分页搜索传来的外公司值不能覆盖；公司展示列保留', async () => {
    renderWithClient(
      <App>
        <FinanceNettingsPage />
      </App>,
    );
    expect(state.ledger?.topBar).toBeUndefined();
    expect(
      state.ledger?.columns.some((column) => column.title === '所属公司'),
    ).toBe(true);
    await state.ledger?.request({
      current: 2,
      pageSize: 20,
      keyword: '条件',
      organizationId: 'other',
    });
    expect(settlementServiceListNettings).toHaveBeenLastCalledWith(
      expect.objectContaining({
        organizationId: 'workspace-A',
        page: 2,
        pageSize: 20,
        keyword: '条件',
      }),
      { skipErrorHandler: true },
    );
  });
  it.each(['未就绪', '系统工作台'])('%s不请求经营数据', async (scenario) => {
    if (scenario === '未就绪') state.organizationId = undefined;
    else state.company = false;
    renderWithClient(
      <App>
        <FinanceNettingsPage />
      </App>,
    );
    await expect(
      state.ledger?.request({ current: 1, pageSize: 20 }),
    ).resolves.toMatchObject({ success: false });
    expect(settlementServiceListNettings).not.toHaveBeenCalled();
  });
  it('工作区重建后分页使用新公司', async () => {
    const first = renderWithClient(
      <App>
        <FinanceNettingsPage />
      </App>,
    );
    await state.ledger?.request({ current: 1, pageSize: 20 });
    first.unmount();
    state.organizationId = 'workspace-B';
    renderWithClient(
      <App>
        <FinanceNettingsPage />
      </App>,
    );
    await state.ledger?.request({ current: 3, pageSize: 40 });
    expect(settlementServiceListNettings).toHaveBeenLastCalledWith(
      expect.objectContaining({ organizationId: 'workspace-B', page: 3 }),
      { skipErrorHandler: true },
    );
  });
});
