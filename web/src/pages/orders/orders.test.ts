import { beforeEach, describe, expect, it, vi } from 'vitest';
import { MasterDataKind } from '@/enums.generated';
import {
  MASTER_DATA_KINDS,
  ORDER_KIND_CONFIGS,
  PARTNER_ROLES,
  businessTypeOptions,
  fetchOrderMasterData,
  isMasterDataKind,
  parseOrderKind,
  requireSeaServiceTypeOptions,
  seaServiceTypes,
  shipmentModeOptions,
  shipmentTypeOptions,
  tradeDirectionOptions,
} from './common';

const listOptions = vi.hoisted(() => vi.fn());
const listPorts = vi.hoisted(() => vi.fn());
const listAirports = vi.hoisted(() => vi.fn());
const getCurrencies = vi.hoisted(() => vi.fn());

vi.mock('@/services/roncin/masterDataService', () => ({
  masterDataServiceListItems: vi.fn(),
  masterDataServiceListOptions: listOptions,
  masterDataServiceListPorts: listPorts,
  masterDataServiceListAirports: listAirports,
}));

vi.mock('@/utils/options', () => ({
  getCurrencies,
  searchPartnerOptions: vi.fn(),
}));

const numericMasterData = [
  ...seaServiceTypes.map(({ code, name }, index) => ({
    id: `service-${index}`,
    code,
    name: code === 'BOOKING' ? '自定义订舱名称' : name,
    kind: MasterDataKind.MASTER_DATA_KIND_SERVICE_TYPE,
    enabled: true,
  })),
  {
    id: 'cargo-1',
    code: 'GENERAL',
    name: '普货',
    kind: MasterDataKind.MASTER_DATA_KIND_CARGO_CATEGORY,
    enabled: true,
  },
  {
    id: 'container-1',
    code: '40HQ',
    name: '40尺高柜',
    kind: MasterDataKind.MASTER_DATA_KIND_CONTAINER_SPEC,
    enabled: true,
  },
  {
    id: 'region-1',
    code: 'SHA',
    name: '上海',
    kind: MasterDataKind.MASTER_DATA_KIND_REGION,
    enabled: true,
  },
] satisfies API.MasterDataItem[];

describe('orders common and config', () => {
  beforeEach(() => {
    vi.clearAllMocks();
    listOptions.mockResolvedValue({ data: numericMasterData });
    listPorts.mockResolvedValue({ data: [] });
    listAirports.mockResolvedValue({ data: [] });
    getCurrencies.mockResolvedValue([]);
  });

  it('正确解析业务类型路径到配置', () => {
    expect(parseOrderKind('sea-export')).toEqual(ORDER_KIND_CONFIGS['sea-export']);
    expect(parseOrderKind('/orders/sea-export')?.businessType).toBe(1);
    expect(parseOrderKind('/orders/sea-export/new')?.category).toBe('sea');

    expect(parseOrderKind('sea-import')).toBeUndefined();
    expect(parseOrderKind('air-export')).toBeUndefined();
    expect(parseOrderKind('air-import')).toBeUndefined();

    expect(parseOrderKind('unknown-kind')).toBeUndefined();
    expect(parseOrderKind('')).toBeUndefined();
  });

  it('验证业务类型与贸易方向配置', () => {
    expect(ORDER_KIND_CONFIGS['sea-export'].tradeDirection).toBe(1);

    expect(businessTypeOptions).toEqual([
      { label: '海运出口', value: 1, color: 'blue' },
    ]);
    expect(tradeDirectionOptions).toHaveLength(2);
    expect(PARTNER_ROLES).not.toHaveProperty('BOOKING_AGENT');
    expect(PARTNER_ROLES.FOREIGN_AGENT).toBe(3);
    expect(shipmentModeOptions).toEqual([
      { label: '集运', value: 1 },
      { label: '跨境', value: 2 },
    ]);
    expect(shipmentTypeOptions).toEqual([
      { label: '整箱', value: 1 },
      { label: '拼箱', value: 2 },
      { label: '散杂', value: 3 },
    ]);
    expect(seaServiceTypes[0]).toEqual({ code: 'BOOKING', name: '订舱' });
    expect(seaServiceTypes).toHaveLength(19);
    expect(seaServiceTypes).toEqual(
      expect.arrayContaining([
        { code: 'PALLET_CHARTER', name: '包板' },
        { code: 'DOCUMENT_EXCHANGE', name: '换单' },
        { code: 'INSPECTION', name: '报检' },
      ]),
    );
  });

  it('按业务编码解析海运服务类型', () => {
    const options = seaServiceTypes.map(({ code, name }, index) => ({
      code,
      label: code === 'BOOKING' ? '自定义订舱名称' : name,
      value: `id-${index}`,
    }));

    expect(requireSeaServiceTypeOptions(options)[0]).toEqual({
      code: 'BOOKING',
      label: '自定义订舱名称',
      value: 'id-0',
    });
    expect(() => requireSeaServiceTypeOptions(options.slice(1))).toThrow(
      '缺少海运服务类型主数据：订舱（BOOKING）',
    );
  });

  it('按 REST 数字枚举识别订单主数据类型', () => {
    expect(MASTER_DATA_KINDS).toEqual({
      REGION: MasterDataKind.MASTER_DATA_KIND_REGION,
      CONTAINER_SPEC: MasterDataKind.MASTER_DATA_KIND_CONTAINER_SPEC,
      SERVICE_TYPE: MasterDataKind.MASTER_DATA_KIND_SERVICE_TYPE,
      CARGO_CATEGORY: MasterDataKind.MASTER_DATA_KIND_CARGO_CATEGORY,
    });
    expect(
      isMasterDataKind(
        MasterDataKind.MASTER_DATA_KIND_REGION,
        MASTER_DATA_KINDS.REGION,
      ),
    ).toBe(true);
    expect(
      isMasterDataKind(
        MasterDataKind.MASTER_DATA_KIND_CONTAINER_SPEC,
        MASTER_DATA_KINDS.CONTAINER_SPEC,
      ),
    ).toBe(true);
    expect(
      isMasterDataKind(
        MasterDataKind.MASTER_DATA_KIND_SERVICE_TYPE,
        MASTER_DATA_KINDS.SERVICE_TYPE,
      ),
    ).toBe(true);
    expect(
      isMasterDataKind(
        MasterDataKind.MASTER_DATA_KIND_CARGO_CATEGORY,
        MASTER_DATA_KINDS.CARGO_CATEGORY,
      ),
    ).toBe(true);
    expect(
      isMasterDataKind(
        MasterDataKind.MASTER_DATA_KIND_CARGO_CATEGORY,
        MASTER_DATA_KINDS.SERVICE_TYPE,
      ),
    ).toBe(false);
  });

  it('使用真实数字枚举构建完整订单主数据候选', async () => {
    const result = await fetchOrderMasterData();

    expect(result.serviceTypeOptions).toHaveLength(19);
    expect(requireSeaServiceTypeOptions(result.serviceTypeOptions)[0]).toEqual({
      code: 'BOOKING',
      label: '自定义订舱名称',
      value: 'service-0',
    });
    expect(result.cargoCategoryOptions).toEqual([
      { code: 'GENERAL', label: '普货', value: 'cargo-1' },
    ]);
    expect(result.seaLocationOptions).toEqual([
      { label: '上海 (SHA)', value: 'region-1' },
    ]);
    expect(
      result.masterOptions.filter((item) =>
        isMasterDataKind(item.kind, MASTER_DATA_KINDS.CONTAINER_SPEC),
      ),
    ).toEqual([
      expect.objectContaining({ id: 'container-1', code: '40HQ' }),
    ]);
  });

  it('真实数字响应缺少 BOOKING 时保留明确报错', async () => {
    listOptions.mockResolvedValue({
      data: numericMasterData.filter((item) => item.code !== 'BOOKING'),
    });

    const result = await fetchOrderMasterData();

    expect(() =>
      requireSeaServiceTypeOptions(result.serviceTypeOptions),
    ).toThrow('缺少海运服务类型主数据：订舱（BOOKING）');
  });
});
