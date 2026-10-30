import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it, vi } from 'vitest';
import * as orderService from '@/services/roncin/orderService';
import {
  candidateResponse,
  masterField,
  renderVoyageForm,
  voyage,
} from './SeaMasterBillFields.test-utils';

afterEach(() => vi.restoreAllMocks());

describe('共同航程与本票默认最终港', () => {
  it.each([false, true])(
    '已有主单默认最终港来自所选共同段（来源开单：%s）',
    async (fromSource) => {
      vi.spyOn(
        orderService,
        'orderServiceMatchSeaMasterBillCandidate',
      ).mockResolvedValue(candidateResponse());
      const { form } = renderVoyageForm(
        {},
        {
          source: fromSource
            ? {
                orderId: 'order-a',
                masterBillId: 'mbl-1',
                masterNo: 'MBL123',
                shippingLineId: 'carrier-1',
                transportExecutionId: 'te-1',
              }
            : undefined,
        },
      );
      await waitFor(() =>
        expect(form.getFieldValue('destinationLocationId')).toBe('discharge'),
      );
      expect(await screen.findByText('洛杉矶（同卸货港）')).toBeVisible();
      expect(form.getFieldValue('seaDestinationDefault')).toMatchObject({
        mode: 'discharge',
      });
    },
  );

  it('采用另一共同段后默认跟随，但进入人工状态后保留当前值并隔离迟到候选', async () => {
    const late =
      Promise.withResolvers<
        Awaited<
          ReturnType<
            typeof orderService.orderServiceMatchSeaMasterBillCandidate
          >
        >
      >();
    const match = vi
      .spyOn(orderService, 'orderServiceMatchSeaMasterBillCandidate')
      .mockResolvedValueOnce(candidateResponse())
      .mockResolvedValueOnce(
        candidateResponse({
          transportExecutions: [
            {
              ...voyage,
              dischargeLocationId: 'new-port',
              dischargeLocationName: '新加坡',
            },
          ],
        }),
      )
      .mockReturnValueOnce(late.promise)
      .mockResolvedValueOnce({ matched: false });
    const { form } = renderVoyageForm();
    await screen.findByText('洛杉矶（同卸货港）');
    await act(async () => form.setFieldValue(masterField, 'NEXT123'));
    await waitFor(() => expect(match).toHaveBeenCalledTimes(2));
    const adoptButton = (await screen.findByText('采用主单航程')).closest(
      'button',
    );
    await waitFor(() => expect(adoptButton).toBeVisible());
    expect(adoptButton).toBeEnabled();
    fireEvent.click(adoptButton as HTMLButtonElement);
    await screen.findByText('新加坡（同卸货港）');
    expect(form.getFieldValue('destinationLocationId')).toBe('new-port');
    fireEvent.click(screen.getByRole('button', { name: '更改最终目的港' }));
    await act(async () => form.setFieldValue(masterField, 'LATE123'));
    await waitFor(() => expect(match).toHaveBeenCalledTimes(3));
    await act(async () => form.setFieldValue(masterField, 'NONE123'));
    await waitFor(() => expect(match).toHaveBeenCalledTimes(4));
    await act(async () => late.resolve(candidateResponse()));
    expect(form.getFieldValue('destinationLocationId')).toBe('new-port');
    expect(form.getFieldValue('seaDestinationDefault')).toMatchObject({
      mode: 'manual',
    });
    expect(form.getFieldValue('seaMasterBillCandidateId')).toBeUndefined();
  });
});
