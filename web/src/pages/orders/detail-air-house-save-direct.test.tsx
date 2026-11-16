import { fireEvent, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { isTabDirty } from '@/components/layout/tabCloseGuard';
import {
  getAirSaveState,
  getAirUpdateMock,
  renderAirSaveDetail,
  resetAirSaveState,
  saveAirDetail,
} from './detail-air-house-save.fixtures';

const state = getAirSaveState();

describe('航空直单整票保存', () => {
  beforeEach(async () => {
    await resetAirSaveState();
    state.order = {
      ...state.order,
      airExport: {
        ...state.order.airExport,
        documentMode: 'DIRECT',
        house: undefined,
      },
    };
  });

  it('没有HAWB字段仍可连续保存，订单基线推进且不创建分单或改动关联版本', async () => {
    renderAirSaveDetail();
    expect(screen.getByText('直单：本票使用主运单，无 HAWB')).toBeVisible();
    expect(screen.queryByLabelText('本票分运单号（HAWB）')).toBeNull();
    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: '直单修改' },
    });
    await saveAirDetail();
    expect(getAirUpdateMock().mock.calls[0][1]).toMatchObject({
      expectedVersion: '4',
      airExport: { documentMode: 'DIRECT', expectedLinkVersion: '1' },
    });
    expect(
      getAirUpdateMock().mock.calls[0][1].airExport?.house,
    ).toBeUndefined();
    expect(state.form?.getFieldValue('version')).toBe('5');
    expect(state.form?.getFieldValue(['airExport', 'linkVersion'])).toBe('1');
    expect(isTabDirty('/orders/air-export/order-1')).toBe(false);
    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: '再次修改' },
    });
    await saveAirDetail();
    expect(getAirUpdateMock().mock.calls[1][1].expectedVersion).toBe('5');
    expect(state.form?.getFieldValue('version')).toBe('6');
  });
});
