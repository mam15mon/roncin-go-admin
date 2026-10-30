import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import {
  getFormDraft,
  getFormDraftKey,
  saveFormDraft,
} from '@/components/layout/formDraft';
import {
  changeDischarge,
  editDestination,
  renderDestination,
} from './SeaFinalDestinationField.fixtures';

const path = '/orders/sea-export/new?sourceOrderId=source-a';
const key = getFormDraftKey('sea-destination', path, 'user-1:org-1');
afterEach(() => sessionStorage.clear());

describe('最终目的港草稿意图', () => {
  it('恢复默认状态草稿后仍随本票卸货港变化', async () => {
    saveFormDraft(key, {
      dischargeLocationId: 'singapore',
      destinationLocationId: 'singapore',
      seaDestinationDefault: {
        mode: 'discharge',
        value: 'singapore',
        label: '新加坡',
      },
    });
    const { handles } = renderDestination({ draftPathname: path });
    await screen.findByText('新加坡（同卸货港）');
    await changeDischarge('bangkok');
    expect(handles.form?.getFieldValue('destinationLocationId')).toBe(
      'bangkok',
    );
    await waitFor(() =>
      expect(getFormDraft(key)).toMatchObject({
        destinationLocationId: 'bangkok',
        seaDestinationDefault: { mode: 'discharge', value: 'bangkok' },
      }),
    );
  });

  it.each(['jakarta', null])(
    '恢复人工目的港%s不因默认卸货港被覆盖',
    async (destination) => {
      saveFormDraft(key, {
        dischargeLocationId: 'singapore',
        destinationLocationId: destination,
        seaDestinationDefault: { mode: 'manual', value: destination },
      });
      const { handles } = renderDestination({
        draftPathname: path,
        initialValues: { dischargeLocationId: 'bangkok' },
      });
      await changeDischarge('bangkok');
      expect(handles.form?.getFieldValue('destinationLocationId')).toBe(
        destination,
      );
      expect(
        handles.form?.getFieldValue('seaDestinationDefault'),
      ).toMatchObject({ mode: 'manual' });
    },
  );

  it('恢复显式null且没有默认元数据的草稿仍保留清空', async () => {
    saveFormDraft(key, {
      dischargeLocationId: 'singapore',
      destinationLocationId: null,
    });
    const { handles } = renderDestination({ draftPathname: path });
    await changeDischarge('bangkok');
    expect(handles.form?.getFieldValue('destinationLocationId')).toBeNull();
    expect(handles.form?.getFieldValue('seaDestinationDefault')).toMatchObject({
      mode: 'manual',
    });
  });

  it('只点击更改而不改值也保存人工意图，恢复后不跟随', async () => {
    const first = renderDestination({
      draftPathname: path,
      initialValues: { dischargeLocationId: 'singapore' },
    });
    await editDestination();
    await waitFor(() =>
      expect(getFormDraft(key)).toMatchObject({
        destinationLocationId: 'singapore',
        seaDestinationDefault: { mode: 'manual' },
      }),
    );
    first.unmount();
    const second = renderDestination({ draftPathname: path });
    await changeDischarge('bangkok');
    expect(second.handles.form?.getFieldValue('destinationLocationId')).toBe(
      'singapore',
    );
  });

  it('来源A与B草稿独立，恢复默认后可显式重置创建意图', async () => {
    saveFormDraft(key, {
      dischargeLocationId: 'singapore',
      destinationLocationId: 'jakarta',
      seaDestinationDefault: { mode: 'manual' },
    });
    const first = renderDestination({ draftPathname: path });
    expect(first.handles.form?.getFieldValue('destinationLocationId')).toBe(
      'jakarta',
    );
    first.unmount();
    const second = renderDestination({
      draftPathname: '/orders/sea-export/new?sourceOrderId=source-b',
      initialValues: { dischargeLocationId: 'bangkok' },
    });
    await screen.findByText('曼谷（同卸货港）');
    expect(second.handles.form?.getFieldValue('destinationLocationId')).toBe(
      'bangkok',
    );
    second.unmount();
    const back = renderDestination({ draftPathname: path });
    expect(back.handles.form?.getFieldValue('destinationLocationId')).toBe(
      'jakarta',
    );
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: '恢复与卸货港一致' })),
    );
    expect(back.handles.form?.getFieldValue('destinationLocationId')).toBe(
      'singapore',
    );
    await act(async () => back.handles.actions?.resetTo({}));
    await changeDischarge('bangkok');
    expect(back.handles.form?.getFieldValue('destinationLocationId')).toBe(
      'bangkok',
    );
  });
});
