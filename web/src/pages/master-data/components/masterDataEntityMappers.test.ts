import { describe, expect, it } from 'vitest';
import { mapAirline } from './AirlinesPanel';
import { mapAirport } from './AirportsPanel';
import { mapPort } from './PortsPanel';
import { mapShippingLine } from './ShippingLinesPanel';

const persisted = {
  id: 'record-1',
  source: 'seed',
  countryCode: 'CN',
  nameEn: 'EXAMPLE',
};
const port: API.Port = { ...persisted, unLocode: 'CNSHG' };
const airport: API.Airport = { ...persisted, iataCode: 'PVG' };
const airline: API.Airline = { ...persisted, iataCode: 'CA' };
const shippingLine: API.ShippingLine = {
  ...persisted,
  scacCode: 'COSU',
  nameZh: '中远',
};

// 各 mapper 消费真实 JSON 缺 key 的形态，不用字段齐全的 fixture 掩盖省略规则。
const cases = [
  {
    name: '港口',
    map: (patch: Partial<API.Port>) => mapPort({ ...port, ...patch }),
  },
  {
    name: '机场',
    map: (patch: Partial<API.Airport>) => mapAirport({ ...airport, ...patch }),
  },
  {
    name: '航司',
    map: (patch: Partial<API.Airline>) => mapAirline({ ...airline, ...patch }),
  },
  {
    name: '船司',
    map: (patch: Partial<API.ShippingLine>) =>
      mapShippingLine({ ...shippingLine, ...patch }),
  },
];

describe.each(cases)('$name 响应映射', ({ name, map }) => {
  it('省略和显式零值均保留记录，并保留非零排序与启用状态', () => {
    expect(map({})).toMatchObject({
      id: 'record-1',
      enabled: false,
      sortOrder: 0,
    });
    expect(map({ enabled: false, sortOrder: 0 })).toEqual(map({}));
    expect(map({ enabled: true, sortOrder: 25 })).toMatchObject({
      enabled: true,
      sortOrder: 25,
    });
  });

  it.each(['id', 'source', 'nameEn', 'countryCode'] as const)(
    '缺少业务必填 %s 仍拒绝',
    (field) => {
      expect(() => map({ [field]: undefined })).toThrow(
        `${name}响应缺少必填字段`,
      );
    },
  );
});

describe('专有字段语义', () => {
  it('港口机场无中文名仍正常展示，空运输方式列表不报错', () => {
    expect(mapPort(port)).toMatchObject({ name: '', modes: [] });
    expect(mapAirport(airport)).toMatchObject({ name: '', cityName: '' });
  });

  it('港口只接受正式运输类型，非标准代码显式失败', () => {
    expect(
      mapPort({ ...port, transportModes: ['SEA', 'RAIL', 'ROAD'] }).modes,
    ).toEqual(['SEA', 'RAIL', 'ROAD']);
    expect(() => mapPort({ ...port, transportModes: ['maritime'] })).toThrow(
      '港口响应包含无效运输类型',
    );
  });

  it('航司可选前缀保持缺失，纯货机省略按 false 映射', () => {
    expect(mapAirline(airline)).toMatchObject({
      name: 'EXAMPLE',
      isCargoOnly: false,
    });
    expect(mapAirline(airline).awbPrefix).toBeUndefined();
    expect(
      mapAirline({ ...airline, cargoOnly: true, awbPrefix: '999' }),
    ).toMatchObject({ isCargoOnly: true, awbPrefix: '999' });
  });

  it('船司必填中文名、各实体业务代码不可缺省', () => {
    expect(() =>
      mapShippingLine({ ...shippingLine, nameZh: undefined }),
    ).toThrow();
    expect(() => mapPort({ ...port, unLocode: undefined })).toThrow();
    expect(() => mapAirport({ ...airport, iataCode: undefined })).toThrow();
    expect(() => mapAirline({ ...airline, iataCode: undefined })).toThrow();
    expect(() =>
      mapShippingLine({ ...shippingLine, scacCode: undefined }),
    ).toThrow();
  });
});
