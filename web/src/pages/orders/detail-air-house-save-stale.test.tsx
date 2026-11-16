import { act, fireEvent, screen } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { isTabDirty } from '@/components/layout/tabCloseGuard';
import {
  commitAirSave,
  getAirSaveState,
  getAirUpdateMock,
  openAirHouseRemarks,
  renderAirSaveDetail,
  rerenderAirSavePage,
  resetAirSaveState,
  saveAirDetail,
} from './detail-air-house-save.fixtures';

const state = getAirSaveState();

describe('空运迟到保存快照与身份隔离', () => {
  beforeEach(() => resetAirSaveState('RELEASED'));

  it('提交后的更晚服务端版本不能借给旧输入，保留基线并要求刷新', async () => {
    getAirUpdateMock().mockImplementation(async (_params, payload) => {
      const saved = commitAirSave(payload);
      return { success: true, data: { ...saved, version: '6' } };
    });
    renderAirSaveDetail();
    await openAirHouseRemarks();
    fireEvent.change(screen.getByLabelText('单证备注'), {
      target: { value: 'submitted-content' },
    });
    fireEvent.change(screen.getByLabelText('已签发分单修改原因'), {
      target: { value: 'submitted-reason' },
    });
    await saveAirDetail();
    expect(
      await screen.findByText(
        '订单已保存，但已发生后续变动，请刷新后继续修改。',
      ),
    ).toBeInTheDocument();
    expect(state.form?.getFieldValue('version')).toBe('4');
    expect(state.form?.getFieldValue(['airExport', 'house', 'version'])).toBe(
      '2',
    );
    expect(screen.getByLabelText('已签发分单修改原因')).toHaveValue(
      'submitted-reason',
    );
    expect(isTabDirty('/orders/air-export/order-1')).toBe(true);
  });

  it('A→B→A 后旧保存响应不能推进新 A 表单基线或清输入', async () => {
    let finishSave!: (response: API.UpdateOrderResponse) => void;
    getAirUpdateMock().mockImplementation(
      () =>
        new Promise((resolve) => {
          finishSave = resolve;
        }),
    );
    const view = renderAirSaveDetail();
    await openAirHouseRemarks();
    fireEvent.change(screen.getByLabelText('单证备注'), {
      target: { value: 'old-submitted-content' },
    });
    fireEvent.change(screen.getByLabelText('已签发分单修改原因'), {
      target: { value: 'old-reason' },
    });
    await saveAirDetail();
    const oldSaved = commitAirSave(getAirUpdateMock().mock.calls[0][1]);
    const original = {
      ...state.order,
      version: '4',
      airExport: {
        ...state.order.airExport,
        house: {
          ...state.order.airExport?.house,
          content: { remarks: 'initial-content' },
          version: '2',
        },
        linkVersion: '1',
      },
    };
    state.routeId = 'order-2';
    state.order = { ...original, id: 'order-2' };
    rerenderAirSavePage(view);
    state.routeId = 'order-1';
    state.order = original;
    rerenderAirSavePage(view);
    await openAirHouseRemarks();
    fireEvent.change(screen.getByLabelText('单证备注'), {
      target: { value: 'new-A-content' },
    });
    fireEvent.change(screen.getByLabelText('已签发分单修改原因'), {
      target: { value: 'new-A-reason' },
    });
    await act(async () => finishSave({ success: true, data: oldSaved }));
    expect(screen.getByLabelText('单证备注')).toHaveValue('new-A-content');
    expect(screen.getByLabelText('已签发分单修改原因')).toHaveValue(
      'new-A-reason',
    );
    expect(state.form?.getFieldValue('version')).toBe('4');
    expect(state.form?.getFieldValue(['airExport', 'house', 'version'])).toBe(
      '2',
    );
    expect(state.loadData).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: '保存修改' })).toBeEnabled();
  });
});
