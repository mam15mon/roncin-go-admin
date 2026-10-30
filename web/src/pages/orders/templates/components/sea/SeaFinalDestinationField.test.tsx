import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { afterEach, describe, expect, it } from 'vitest';
import { buildSeaExportCreatePayload } from '../../../order-kinds/sea-export/form-adapter';
import {
  changeDischarge,
  chooseDestination,
  editDestination,
  renderDestination,
} from './SeaFinalDestinationField.fixtures';

afterEach(() => sessionStorage.clear());

describe('新建海运最终目的港默认值', () => {
  it('首票只选择卸货港，最终港默认同港并显式进入创建载荷', async () => {
    const { handles } = renderDestination();
    await changeDischarge('singapore');
    expect(handles.form?.getFieldValue('destinationLocationId')).toBe(
      'singapore',
    );
    expect(await screen.findByText('新加坡（同卸货港）')).toBeVisible();
    expect(screen.queryByRole('combobox', { name: '最终目的港' })).toBeNull();
    const payload = buildSeaExportCreatePayload({
      customerId: 'customer',
      paymentTerm: 1,
      ...handles.form?.getFieldsValue(true),
    });
    expect(payload).toMatchObject({
      destinationLocationId: 'singapore',
      dischargeLocationId: 'singapore',
    });
    expect(payload).not.toHaveProperty('seaDestinationDefault');
  });

  it('默认状态更换卸货港持续跟随，清卸货港也清默认值', async () => {
    const { handles } = renderDestination();
    await changeDischarge('singapore');
    await changeDischarge('bangkok');
    expect(handles.form?.getFieldValue('destinationLocationId')).toBe(
      'bangkok',
    );
    await changeDischarge('');
    expect(handles.form?.getFieldValue('destinationLocationId')).toBe('');
  });

  it('本票改为不同最终港后，共同段变化不覆盖它', async () => {
    const { handles } = renderDestination({
      initialValues: { dischargeLocationId: 'singapore' },
    });
    await editDestination();
    await chooseDestination('雅加达');
    await changeDischarge('bangkok');
    expect(handles.form?.getFieldValue('destinationLocationId')).toBe(
      'jakarta',
    );
    expect(handles.form?.getFieldValue('seaDestinationDefault')).toMatchObject({
      mode: 'manual',
    });
  });

  it('选择同港也是人工意图，只有明确恢复默认才再次跟随', async () => {
    const { handles } = renderDestination({
      initialValues: { dischargeLocationId: 'singapore' },
    });
    await editDestination();
    await chooseDestination('新加坡');
    await changeDischarge('bangkok');
    expect(handles.form?.getFieldValue('destinationLocationId')).toBe(
      'singapore',
    );
    await act(async () =>
      fireEvent.click(screen.getByRole('button', { name: '恢复与卸货港一致' })),
    );
    expect(handles.form?.getFieldValue('destinationLocationId')).toBe(
      'bangkok',
    );
    await changeDischarge('singapore');
    expect(handles.form?.getFieldValue('destinationLocationId')).toBe(
      'singapore',
    );
  });

  it('人工清空最终港不会被卸货港的下一次变化填回', async () => {
    const { handles, container } = renderDestination({
      initialValues: { dischargeLocationId: 'singapore' },
    });
    await editDestination();
    const clear = container.querySelector('.ant-select-clear');
    if (!clear) throw new Error('最终目的港清空操作未渲染');
    await act(async () => fireEvent.click(clear));
    await changeDischarge('bangkok');
    expect(
      handles.form?.getFieldValue('destinationLocationId'),
    ).toBeUndefined();
    expect(handles.form?.getFieldValue('seaDestinationDefault')).toMatchObject({
      mode: 'manual',
    });
  });

  it.each(['singapore', 'jakarta'])(
    '明确预填%s即使同港也不变成持续绑定',
    async (preset) => {
      const { handles } = renderDestination({
        initialValues: {
          dischargeLocationId: 'singapore',
          destinationLocationId: preset,
        },
      });
      await waitFor(() =>
        expect(
          handles.form?.getFieldValue('seaDestinationDefault'),
        ).toMatchObject({ mode: 'manual' }),
      );
      await changeDischarge('bangkok');
      expect(handles.form?.getFieldValue('destinationLocationId')).toBe(preset);
    },
  );

  it.each([undefined, 'jakarta'])(
    '已有详情原值%s不因挂载或共同段变化自动改写',
    async (destination) => {
      const { handles } = renderDestination({
        isDetail: true,
        initialValues: {
          destinationLocationId: destination,
          dischargeLocationId: 'singapore',
        },
      });
      await changeDischarge('bangkok');
      expect(handles.form?.getFieldValue('destinationLocationId')).toBe(
        destination,
      );
      expect(
        handles.form?.getFieldValue('seaDestinationDefault'),
      ).toBeUndefined();
    },
  );
});
