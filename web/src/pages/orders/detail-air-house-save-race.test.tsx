import { act, fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import { isTabDirty } from '@/components/layout/tabCloseGuard';
import {
  commitAirSave,
  getAirSaveState,
  getAirUpdateMock,
  renderAirSaveDetail,
  rerenderAirSavePage,
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
    fireEvent.change(screen.getByLabelText('本票单证资料'), {
      target: { value: 'submitted-content' },
    });
    fireEvent.change(screen.getByLabelText('已签发分单修改原因'), {
      target: { value: 'submitted-reason' },
    });
    await saveAirDetail();
    await waitFor(() => expect(getAirUpdateMock()).toHaveBeenCalledOnce());
    fireEvent.change(screen.getByLabelText('本票单证资料'), {
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
    expect(screen.getByLabelText('本票单证资料')).toHaveValue(
      'new-unsaved-content',
    );
    expect(screen.getByLabelText('已签发分单修改原因')).toHaveValue(
      'new-unsaved-reason',
    );
    expect(isTabDirty('/orders/air-export/order-1')).toBe(true);
    expect(state.form?.getFieldValue(['airExport', 'houseContent'])).toBe(
      'submitted-content',
    );
    expect(state.form?.getFieldValue(['airExport', 'houseVersion'])).toBe('3');
  });

  it('保存失败不推进签发基线或清除输入', async () => {
    await resetAirSaveState();
    getAirUpdateMock().mockRejectedValue(new Error('保存失败'));
    renderAirSaveDetail();
    await setHouseStatus('已签发');
    await saveAirDetail();
    expect(screen.queryByLabelText('已签发分单修改原因')).toBeNull();
    expect(state.form?.getFieldValue(['airExport', 'houseStatus'])).toBe(
      'DRAFT',
    );
    expect(state.form?.getFieldValue('airHouseStatus')).toBe('RELEASED');
    expect(state.form?.getFieldValue('version')).toBe('4');
    expect(isTabDirty('/orders/air-export/order-1')).toBe(true);
  });

  it('提交后的更晚服务端版本不能借给旧输入，保留基线并要求刷新', async () => {
    getAirUpdateMock().mockImplementation(async (_params, payload) => {
      const saved = commitAirSave(payload);
      return { success: true, data: { ...saved, version: '6' } };
    });
    renderAirSaveDetail();
    fireEvent.change(screen.getByLabelText('本票单证资料'), {
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
    expect(state.form?.getFieldValue(['airExport', 'houseVersion'])).toBe('2');
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
    fireEvent.change(screen.getByLabelText('本票单证资料'), {
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
        houseContent: 'initial-content',
        houseVersion: '2',
      },
    };
    state.routeId = 'order-2';
    state.order = { ...original, id: 'order-2' };
    rerenderAirSavePage(view);
    state.routeId = 'order-1';
    state.order = original;
    rerenderAirSavePage(view);
    fireEvent.change(screen.getByLabelText('本票单证资料'), {
      target: { value: 'new-A-content' },
    });
    fireEvent.change(screen.getByLabelText('已签发分单修改原因'), {
      target: { value: 'new-A-reason' },
    });
    await act(async () => finishSave({ success: true, data: oldSaved }));
    expect(screen.getByLabelText('本票单证资料')).toHaveValue('new-A-content');
    expect(screen.getByLabelText('已签发分单修改原因')).toHaveValue(
      'new-A-reason',
    );
    expect(state.form?.getFieldValue('version')).toBe('4');
    expect(state.form?.getFieldValue(['airExport', 'houseVersion'])).toBe('2');
    expect(state.loadData).not.toHaveBeenCalled();
    expect(screen.getByRole('button', { name: '保存修改' })).toBeEnabled();
  });
});
