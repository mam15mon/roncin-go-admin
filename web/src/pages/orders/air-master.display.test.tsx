import { describe, expect, it } from 'vitest';
import {
  formatAirMasterHistoryAction,
  formatPackagesByUnit,
} from './air-master';

describe('航空主单展示文案', () => {
  it('历史动作按服务端写入值显示中文，未知值原样显示', () => {
    expect(
      ['CREATE_LINK', 'END_LINK', 'RESTORE_LINK', 'UPDATE'].map(
        formatAirMasterHistoryAction,
      ),
    ).toEqual(['建票关联', '终止关联', '恢复关联', '修改共同资料']);
    expect(formatAirMasterHistoryAction('SOMETHING_NEW')).toBe('SOMETHING_NEW');
    expect(formatAirMasterHistoryAction(undefined)).toBe('-');
  });

  it('件数单位缺省时只显示件数，零件数不被吞掉', () => {
    // ProtoJSON 省略空串单位与零值件数
    expect(formatPackagesByUnit([{ packages: 30 }, { unit: 'CTNS' }])).toBe(
      '30 / 0 CTNS',
    );
    expect(formatPackagesByUnit([])).toBe('-');
    expect(formatPackagesByUnit(undefined)).toBe('-');
  });
});
