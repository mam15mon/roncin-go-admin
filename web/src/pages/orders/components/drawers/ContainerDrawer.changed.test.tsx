import { renderWithApp } from '@root/tests/renderWithApp';
import { act } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import type { SubEntityDrawerTemplateProps } from '@/components/ui/sub-entity-drawer/SubEntityDrawerTemplate';
import {
  orderContainerServiceAddContainer,
  orderContainerServiceRemoveContainer,
  orderContainerServiceUpdateContainer,
} from '@/services/roncin/orderContainerService';
import ContainerDrawer from './ContainerDrawer';

const captured = vi.hoisted(() => ({ props: undefined as unknown }));
vi.mock('@/components/ui/sub-entity-drawer', () => ({
  SubEntityDrawerTemplate: (props: unknown) => {
    captured.props = props;
    return null;
  },
}));
vi.mock('./SeaSharedContainerDrawer', () => ({ default: () => null }));
vi.mock('@/services/roncin/orderContainerService', () => ({
  orderContainerServiceAddContainer: vi.fn(),
  orderContainerServiceUpdateContainer: vi.fn(),
  orderContainerServiceRemoveContainer: vi.fn(),
  orderContainerServiceListContainers: vi.fn(),
}));
type Values = {
  containerNo: string;
  containerSpecId: string;
  packageCount: number;
  grossWeightKg: number;
  volumeCbm: number;
};
const values: Values = {
  containerNo: 'TGHU1234567',
  containerSpecId: 'spec',
  packageCount: 10,
  grossWeightKg: 100,
  volumeCbm: 2,
};
const parent = { id: 'order-1' };
const record = { id: 'box-1', version: '8' };
beforeEach(() => vi.resetAllMocks());
describe('普通实际箱变更通知', () => {
  it.each(['create', 'update', 'remove'] as const)(
    '%s 成功才通知摘要刷新，失败不通知',
    async (operation) => {
      const onChanged = vi.fn();
      renderWithApp(
        <ContainerDrawer
          canCreate
          canUpdate
          canRemove
          containerSpecOptions={[]}
          containerSpecMap={{}}
          onChanged={onChanged}
          hideSharedEntry
        />,
      );
      const props = captured.props as SubEntityDrawerTemplateProps<
        API.OrderContainer,
        API.Order,
        Values
      >;
      const service =
        operation === 'create'
          ? vi.mocked(orderContainerServiceAddContainer)
          : operation === 'update'
            ? vi.mocked(orderContainerServiceUpdateContainer)
            : vi.mocked(orderContainerServiceRemoveContainer);
      const invoke = () =>
        operation === 'create'
          ? props.createItem?.(values, parent)
          : operation === 'update'
            ? props.updateItem?.(record, values, parent)
            : props.removeItem?.(record, parent);
      service.mockResolvedValueOnce(undefined as never);
      await act(async () => {
        await invoke();
      });
      expect(onChanged).not.toHaveBeenCalled();
      service.mockResolvedValueOnce({
        success: true,
        data: { id: 'box-1' },
      } as never);
      await act(async () => {
        await invoke();
      });
      expect(onChanged).toHaveBeenCalledOnce();
    },
  );
});
