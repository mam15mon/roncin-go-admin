import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { isTabDirty } from '@/components/layout/tabCloseGuard';
import {
  commitAirSave,
  getAirSaveState,
  getAirUpdateMock,
  openAirHouseRemarks,
  renderAirSaveDetail,
  resetAirSaveState,
  saveAirDetail,
  setHouseStatus,
} from './detail-air-house-save.fixtures';

const state = getAirSaveState();

describe('空运保存期间的输入与身份隔离', () => {
  beforeEach(() => resetAirSaveState('RELEASED'));

  it('成功仅清已提交原因，保留等待期间的新正文、新原因与未保存保护', async () => {
    let finishSave!: (response: API.UpdateOrderResponse) => void;
    getAirUpdateMock().mockImplementation(
      () =>
        new Promise((resolve) => {
          finishSave = resolve;
        }),
    );
    renderAirSaveDetail();
    await openAirHouseRemarks();
    fireEvent.change(screen.getByLabelText('单证备注'), {
      target: { value: 'submitted-content' },
    });
    fireEvent.change(screen.getByLabelText('已签发分单修改原因'), {
      target: { value: 'submitted-reason' },
    });
    await saveAirDetail();
    await waitFor(() => expect(getAirUpdateMock()).toHaveBeenCalledOnce());
    await openAirHouseRemarks();
    fireEvent.change(screen.getByLabelText('单证备注'), {
      target: { value: 'new-unsaved-content' },
    });
    fireEvent.change(screen.getByLabelText('已签发分单修改原因'), {
      target: { value: 'new-unsaved-reason' },
    });
    await act(async () =>
      finishSave({
        success: true,
        data: commitAirSave(getAirUpdateMock().mock.calls[0][1]),
      }),
    );
    expect(screen.getByLabelText('单证备注')).toHaveValue(
      'new-unsaved-content',
    );
    expect(screen.getByLabelText('已签发分单修改原因')).toHaveValue(
      'new-unsaved-reason',
    );
    expect(isTabDirty('/orders/air-export/order-1')).toBe(true);
    expect(
      state.form?.getFieldValue(['airExport', 'house', 'content', 'remarks']),
    ).toBe('submitted-content');
    expect(state.form?.getFieldValue(['airExport', 'house', 'version'])).toBe(
      '3',
    );
  });

  it('保存失败不推进签发基线或清除输入', async () => {
    await resetAirSaveState();
    getAirUpdateMock().mockRejectedValue(new Error('保存失败'));
    renderAirSaveDetail();
    await setHouseStatus('已签发');
    await saveAirDetail();
    expect(screen.queryByLabelText('已签发分单修改原因')).toBeNull();
    expect(state.form?.getFieldValue(['airExport', 'house', 'status'])).toBe(
      'DRAFT',
    );
    expect(state.form?.getFieldValue('airHouseStatus')).toBe('RELEASED');
    expect(state.form?.getFieldValue('version')).toBe('4');
    expect(isTabDirty('/orders/air-export/order-1')).toBe(true);
  });
});
