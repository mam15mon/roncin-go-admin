import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import {
  bundle,
  getPortSaveState,
  renderDetail,
  resetPortSaveState,
} from './detail-port-save.fixtures';

const service = vi.hoisted(() => ({
  save: undefined as (() => Promise<void>) | undefined,
}));
vi.mock('./import-handling/ImportHandlingPanel', () => ({
  default: ({
    order,
    expectedVersion,
    onSaved,
  }: {
    order: API.Order;
    expectedVersion?: string;
    onSaved: (result: {
      orderId: string;
      previousVersion: string;
      saved: API.OrderImportHandling;
    }) => Promise<void>;
  }) => {
    service.save = () =>
      onSaved({
        orderId: order.id as string,
        previousVersion: expectedVersion as string,
        saved: { orderId: order.id, orderVersion: '5' },
      });
    return <div>本票进口服务</div>;
  },
}));
const state = getPortSaveState();
beforeEach(() => {
  resetPortSaveState();
  state.routeKind = 'sea-import';
  state.order = {
    ...state.order,
    businessType: OrderBusinessType.BUSINESS_TYPE_SI,
    seaMasterBill: {
      ...state.order.seaMasterBill,
      transportExecutionId: 'voyage',
      transportExecutionVersion: '2',
    },
  };
  state.documents = { ...state.documents, orderId: 'order-1' };
  service.save = undefined;
});
describe('SI独立服务保存与整票编辑输入', () => {
  it('本人服务保存仅推进确切Order版本，不清未保存业务字段和主单内容基线', async () => {
    renderDetail();
    expect(screen.getByText('本票进口服务')).toBeVisible();
    expect(
      screen
        .getByText('本票进口服务')
        .compareDocumentPosition(screen.getByLabelText('客户业务编号')) &
        Node.DOCUMENT_POSITION_FOLLOWING,
    ).toBeTruthy();
    fireEvent.change(screen.getByLabelText('客户业务编号'), {
      target: { value: '尚未保存的客户编号' },
    });
    state.loadData.mockImplementationOnce(async () => ({
      ...bundle(),
      order: { ...state.order, version: '5' },
    }));
    await act(async () => service.save?.());
    await waitFor(() => expect(state.form?.getFieldValue('version')).toBe('5'));
    expect(screen.getByLabelText('客户业务编号')).toHaveValue(
      '尚未保存的客户编号',
    );
    expect(state.form?.getFieldValue('seaMasterBillContentVersion')).toBe('4');
  });
  it.each(['order', 'voyage'])(
    '服务成功后重读夹带其他%s修改时保留旧基线与全部输入',
    async (changed) => {
      renderDetail();
      fireEvent.change(screen.getByLabelText('客户业务编号'), {
        target: { value: '仍需保留' },
      });
      state.loadData.mockImplementationOnce(async () => ({
        ...bundle(),
        order: {
          ...state.order,
          version: changed === 'order' ? '6' : '5',
          seaMasterBill: {
            ...state.order.seaMasterBill,
            transportExecutionVersion: changed === 'voyage' ? '3' : '2',
          },
        },
      }));
      await act(async () => service.save?.());
      expect(state.form?.getFieldValue('version')).toBe('4');
      expect(screen.getByLabelText('客户业务编号')).toHaveValue('仍需保留');
      expect(
        await screen.findByText(/未保存的订单输入已保留/),
      ).toBeInTheDocument();
    },
  );
});
