import { renderWithApp } from '@root/tests/renderWithApp';
import { screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderPersonnelRole } from '@/enums.generated';
import OrderDetailPage from './detail';
import type { OrderDetailFeaturesProps } from './order-kinds/types';
import type { TemplateProps } from './templates/types';

const state = vi.hoisted(() => ({
  kind: 'sea-export',
  canSearchPersonnel: true,
  order: {} as API.Order,
  personnel: [] as API.OrderPersonnel[],
  personnelOptions: [] as API.OrderPersonnelOption[],
}));

vi.mock('react-router', async (importOriginal) => ({
  ...(await importOriginal<typeof import('react-router')>()),
  useParams: () => ({ kind: state.kind, id: state.order.id }),
}));

vi.mock('@/app/access', () => ({
  useAccess: () => ({
    canOperateOrganization: () => true,
    canOrder: (_businessType: number, operation: string) =>
      operation === 'create' ? state.canSearchPersonnel : true,
  }),
}));

vi.mock('./use-order-detail-data', () => ({
  useOrderDetailData: () => ({
    loading: false,
    error: null,
    order: state.order,
    seaDocuments: { documentStructure: 2, linkVersion: '1' },
    shippingDocs: [],
    personnel: state.personnel,
    cargoCategoryOptions: [],
    locationOptions: [],
    searchLocations: async () => [],
    currencyOptions: [],
    containerSpecOptions: [],
    personnelOptions: state.personnelOptions,
    draftScope: 'viewer:org-1',
    loadData: vi.fn(),
  }),
}));

vi.mock('./use-order-lock-state', async (importOriginal) => ({
  ...(await importOriginal<typeof import('./use-order-lock-state')>()),
  useOrderLockState: () => ({
    state: { isLocked: false },
    loading: false,
    error: null,
    refresh: vi.fn(),
  }),
}));

vi.mock('./order-kinds/sea-export/SeaExportDetailFeatures', () => ({
  default: ({ children }: OrderDetailFeaturesProps) => children({}),
}));
vi.mock('./order-kinds/air-export/AirExportDetailFeatures', () => ({
  default: ({ children }: OrderDetailFeaturesProps) => children({}),
}));
vi.mock('./components/detail/OrderDetailHeader', () => ({
  default: () => null,
}));

// 保留真实详情页回填、品类适配器和人员选择框，仅省略无关业务分节。
vi.mock('./templates', async () => {
  const { buildSeaPersonnelSection } = await import(
    './templates/components/sea/SeaPersonnelSection'
  );
  const buildSections = (props: TemplateProps) => [
    buildSeaPersonnelSection(props),
  ];
  return {
    getSeaTemplateSections: buildSections,
    getAirTemplateSections: buildSections,
  };
});

function personnelField(label: string) {
  return screen.getByText(label).closest('.ant-form-item');
}

describe('订单详情创建人员展示', () => {
  beforeEach(() => {
    sessionStorage.clear();
    state.kind = 'sea-export';
    state.canSearchPersonnel = true;
    state.order = {
      id: 'order-1',
      orderNo: 'SE2026110800002',
      organizationId: 'org-1',
      version: '1',
      createdAt: '2026-11-08T00:00:00Z',
      creatorName: '张创建',
    };
    state.personnel = [
      // 创建人与当前登录人、操作人不同，且不在可编辑人员候选中。
      {
        userId: 'operator-id',
        role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_OPERATOR,
      },
      {
        userId: 'creator-internal-id',
        role: OrderPersonnelRole.ORDER_PERSONNEL_ROLE_CREATOR,
      },
    ];
    state.personnelOptions = [{ userId: 'operator-id', displayName: '李操作' }];
  });

  it.each(['sea-export', 'air-export'])(
    '%s 显示订单创建人姓名并保持不可编辑',
    async (kind) => {
      state.kind = kind;
      renderWithApp(<OrderDetailPage />);

      await waitFor(() =>
        expect(personnelField('创建人员')).toHaveTextContent('张创建'),
      );
      expect(personnelField('创建人员')).not.toHaveTextContent(
        /creator-internal-id|·/,
      );
      expect(screen.getByLabelText('创建人员')).toBeDisabled();
      expect(personnelField('操作人员')).toHaveTextContent('李操作');
      expect(personnelField('操作人员')).not.toHaveTextContent('·');
    },
  );

  it('只有人员读取权限、候选为空时仍显示创建人姓名', async () => {
    state.canSearchPersonnel = false;
    state.personnelOptions = [];
    renderWithApp(<OrderDetailPage />);

    await waitFor(() =>
      expect(personnelField('创建人员')).toHaveTextContent('张创建'),
    );
    expect(screen.getByLabelText('创建人员')).toBeDisabled();
  });
});
