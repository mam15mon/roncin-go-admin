import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { isTabDirty } from '@/components/layout/tabCloseGuard';
import {
  getAirSaveState,
  getAirUpdateMock,
  openAirHouseRemarks,
  renderAirSaveDetail,
  resetAirSaveState,
  saveAirDetail,
  setHouseStatus,
} from './detail-air-house-save.fixtures';

const state = getAirSaveState();

describe('空运首次签发与同页改单', () => {
  beforeEach(() => resetAirSaveState());
  it('首次签发后立即出现原因框，同页改单提交原因与本次保存基线', async () => {
    renderAirSaveDetail();
    expect(screen.queryByLabelText('已签发分单修改原因')).toBeNull();
    await setHouseStatus('已签发');
    await saveAirDetail();
    expect(await screen.findByLabelText('已签发分单修改原因')).toBeVisible();
    expect(isTabDirty('/orders/air-export/order-1')).toBe(false);
    await openAirHouseRemarks();
    fireEvent.change(screen.getByLabelText('单证备注'), {
      target: { value: 'amended-content' },
    });
    fireEvent.change(screen.getByLabelText('已签发分单修改原因'), {
      target: { value: '客户确认修改' },
    });
    await saveAirDetail();
    await waitFor(() => expect(getAirUpdateMock()).toHaveBeenCalledTimes(2));
    expect(getAirUpdateMock().mock.calls[1][1]).toMatchObject({
      expectedVersion: '5',
      airExport: {
        house: {
          status: 'RELEASED',
          content: { remarks: 'amended-content' },
          expectedVersion: '3',
        },
        changeReason: '客户确认修改',
        expectedLinkVersion: '2',
      },
    });
    expect(state.form?.getFieldValue(['airExport', 'house', 'version'])).toBe(
      '4',
    );
    expect(screen.getByLabelText('已签发分单修改原因')).toHaveValue('');
  });
});
