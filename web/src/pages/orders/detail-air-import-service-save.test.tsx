import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import {
  getAirSaveState,
  renderAirSaveDetail,
  resetAirSaveState,
} from './detail-air-house-save.fixtures';

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
    onSaved: (value: {
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
    return <div>本票AI进口服务</div>;
  },
}));
const state = getAirSaveState();
beforeEach(async () => {
  await resetAirSaveState();
  state.routeKind = 'air-import';
  state.order = {
    ...state.order,
    businessType: OrderBusinessType.BUSINESS_TYPE_AI,
    airExport: {
      ...state.order.airExport,
      master: { ...state.order.airExport?.master, businessType: 'AI' },
    },
  };
  service.save = undefined;
});
describe('AI服务独立保存与真实整票表单', () => {
  it.each(['DIRECT', 'HOUSE'])(
    '%s服务成功只推进Order版本并保留未保存经营输入与单证基线',
    async (documentMode) => {
      state.order.airExport = {
        ...state.order.airExport,
        documentMode,
        house:
          documentMode === 'DIRECT' ? undefined : state.order.airExport?.house,
      };
      renderAirSaveDetail();
      expect(screen.getByText('本票AI进口服务')).toBeVisible();
      expect(
        screen
          .getByText('本票AI进口服务')
          .compareDocumentPosition(screen.getByLabelText('客户参考号')) &
          Node.DOCUMENT_POSITION_FOLLOWING,
      ).toBeTruthy();
      fireEvent.change(screen.getByLabelText('客户参考号'), {
        target: { value: '尚未保存客户输入' },
      });
      state.loadData.mockResolvedValueOnce({
        order: { ...state.order, version: '5' },
      });
      await act(async () => service.save?.());
      await waitFor(() =>
        expect(state.form?.getFieldValue('version')).toBe('5'),
      );
      expect(screen.getByLabelText('客户参考号')).toHaveValue(
        '尚未保存客户输入',
      );
      expect(state.form?.getFieldValue('airMasterExpectedVersion')).toBe('3');
      expect(state.form?.getFieldValue('airExport').linkVersion).toBe('1');
    },
  );
  it.each(['order', 'master', 'house'])(
    '服务后重读夹带其他%s写入不推进旧稿基线',
    async (target) => {
      renderAirSaveDetail();
      fireEvent.change(screen.getByLabelText('客户参考号'), {
        target: { value: '保留输入' },
      });
      const latest = structuredClone(state.order);
      latest.version = target === 'order' ? '6' : '5';
      if (target === 'master' && latest.airExport?.master)
        latest.airExport.master.version = '4';
      if (target === 'house' && latest.airExport?.house)
        latest.airExport.house.version = '3';
      state.loadData.mockResolvedValueOnce({ order: latest });
      await act(async () => service.save?.());
      expect(state.form?.getFieldValue('version')).toBe('4');
      expect(screen.getByLabelText('客户参考号')).toHaveValue('保留输入');
      expect(
        await screen.findByText(/未保存的订单输入已保留/),
      ).toBeInTheDocument();
    },
  );
});
