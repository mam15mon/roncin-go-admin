import { describe, expect, it } from 'vitest';
import { mapPersistedMasterDataItem } from './masterDataMapper';

const item: API.MasterDataItem = {
  id: 'agent-1',
  code: 'AGENT-1',
  name: '船代一',
  source: 'seed',
};

describe('主数据 ProtoJSON 映射', () => {
  it('省略零值字段时还原排序 0 和停用状态', () => {
    expect(mapPersistedMasterDataItem(item)).toMatchObject({
      sortOrder: 0,
      enabled: false,
    });
  });

  it('保留显式排序和启用状态', () => {
    expect(
      mapPersistedMasterDataItem({ ...item, sortOrder: 10, enabled: true }),
    ).toMatchObject({ sortOrder: 10, enabled: true });
  });

  it.each(['id', 'code', 'name', 'source'] as const)(
    '缺少必填字段 %s 时仍报错',
    (field) => {
      expect(() =>
        mapPersistedMasterDataItem({ ...item, [field]: undefined }),
      ).toThrow('主数据响应缺少必填字段');
    },
  );
});
