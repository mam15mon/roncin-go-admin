import { fireEvent, screen, waitFor } from '@testing-library/react';
import { beforeEach, describe, expect, it } from 'vitest';
import {
  getAirSaveState,
  getAirUpdateMock,
  renderAirSaveDetail,
  resetAirSaveState,
  saveAirDetail,
  setHouseStatus,
} from './detail-air-house-save.fixtures';

const state = getAirSaveState();

describe('空运 HAWB 保存后同页改单', () => {
  beforeEach(() => resetAirSaveState());

  it('此前已签发时改回草稿仍保留原因入口，保存后才按新状态收起', async () => {
    await resetAirSaveState('RELEASED');
    renderAirSaveDetail();
    await setHouseStatus('资料待完善');
    expect(screen.getByLabelText('已签发分单修改原因')).toBeVisible();
    fireEvent.change(screen.getByLabelText('已签发分单修改原因'), {
      target: { value: '退回资料待完善' },
    });
    await saveAirDetail();
    await waitFor(() =>
      expect(screen.queryByLabelText('已签发分单修改原因')).toBeNull(),
    );
    expect(getAirUpdateMock().mock.calls[0][1].airExport).toMatchObject({
      house: { status: 'DRAFT', expectedVersion: '2' },
      changeReason: '退回资料待完善',
    });
  });

  it('仅改普通字段允许空原因，HAWB 原版本保持不变', async () => {
    await resetAirSaveState('RELEASED');
    renderAirSaveDetail();
    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: 'updated-reference' },
    });
    await saveAirDetail();
    expect(getAirUpdateMock().mock.calls[0][1]).toMatchObject({
      customerReferenceNo: 'updated-reference',
      airExport: { house: { status: 'RELEASED', expectedVersion: '2' } },
    });
    expect(state.form?.getFieldValue(['airExport', 'house', 'version'])).toBe(
      '2',
    );
    expect(state.form?.getFieldValue('version')).toBe('5');
  });
  it('不借用保存响应中后来更新的共享主单基线', async () => {
    const normalSave = getAirUpdateMock().getMockImplementation();
    getAirUpdateMock().mockImplementation(async (params, payload, options) => {
      const result = await normalSave?.(params, payload, options);
      if (!result?.data?.airExport) throw new Error('测试响应缺少空运资料');
      return {
        ...result,
        data: {
          ...result.data,
          airExport: {
            ...result.data.airExport,
            master: { ...result.data.airExport.master, version: '9' },
          },
        },
      };
    });
    renderAirSaveDetail();
    fireEvent.change(screen.getByLabelText('客户参考号'), {
      target: { value: 'updated-reference' },
    });
    await saveAirDetail();
    expect(state.form?.getFieldValue('airMasterExpectedVersion')).toBe('3');
    expect(state.form?.getFieldValue(['airExport', 'master', 'version'])).toBe(
      '3',
    );
    expect(state.form?.getFieldValue('version')).toBe('4');
  });
});
