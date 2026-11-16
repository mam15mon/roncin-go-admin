import { beforeEach, describe, expect, it, vi } from 'vitest';
import { OrderBusinessType } from '@/enums.generated';
import { serializeCsv } from '@/utils/csv';
import { formatDate } from '@/utils/format';
import {
  fetchOrderListExport,
  orderExportColumns,
  orderExportFileName,
} from './list-export';
import { buildOrderListRequestFilters } from './list-query';
import { getOrderKindDefinition } from './order-kinds/registry';

const mocks = vi.hoisted(() => ({
  export: vi.fn(),
  list: vi.fn(),
  masterOptions: vi.fn(),
}));
vi.mock('@/services/roncin/orderService', () => ({
  orderServiceExportOrders: mocks.export,
  orderServiceListOrders: mocks.list,
}));
vi.mock('@/features/orders/options', () => ({
  getMasterDataOptions: mocks.masterOptions,
}));

const generatedAt = '2026-11-16T16:30:45Z';
const context = {
  organizationId: 'company',
  ports: [],
  airports: [],
  customerMap: {},
};
function requireDefinition(kind: string) {
  const found = getOrderKindDefinition(kind);
  if (!found) throw new Error('测试kind未注册');
  return found;
}
const definition = requireDefinition('air-import');
const order = {
  id: 'id-1',
  orderNo: 'AI-001',
  businessType: OrderBusinessType.BUSINESS_TYPE_AI,
  organizationId: 'company',
};
beforeEach(() => vi.resetAllMocks());

describe('订单完整筛选专用导出', () => {
  it.each(['sea-export', 'sea-import', 'air-export', 'air-import'])(
    '%s超过200行只请求一次专用接口，筛选排序与列表同入口',
    async (kind) => {
      const current = requireDefinition(kind);
      const filters = {
        numberKeyword: '中文,客户',
        isLocked: 'unlocked' as const,
        shareStatus: 'unshared' as const,
        sorterField: 'eta',
        sorterOrder: 'ascend' as const,
        etaRange: ['2026-11-15', '2026-11-17'] as [string, string],
      };
      mocks.export.mockResolvedValue({
        success: true,
        data: Array.from({ length: 251 }, (_, i) => ({
          ...order,
          businessType: current.businessType,
          id: `id-${i}`,
          orderNo: `${kind}-${i}`,
        })),
        total: 251,
        generatedAt,
      });
      const result = await fetchOrderListExport(
        { filters, total: 999 },
        current,
        context,
      );
      expect(result.rows).toHaveLength(251);
      expect(mocks.export).toHaveBeenCalledExactlyOnceWith(
        buildOrderListRequestFilters(filters, current),
      );
      expect(mocks.export.mock.calls[0][0]).not.toHaveProperty('page');
      expect(mocks.export.mock.calls[0][0]).not.toHaveProperty('pageSize');
      expect(mocks.list).not.toHaveBeenCalled();
    },
  );

  it('ProtoJSON省略零总数/空data可成功采纳空结果，失败不能当空', async () => {
    mocks.export.mockResolvedValue({ success: true, generatedAt });
    expect(
      (
        await fetchOrderListExport(
          { filters: {}, total: 0 },
          definition,
          context,
        )
      ).rows,
    ).toEqual([]);
    mocks.export.mockResolvedValue({ success: false, generatedAt });
    await expect(
      fetchOrderListExport({ filters: {}, total: 0 }, definition, context),
    ).rejects.toThrow('导出失败');
  });

  it.each([
    { total: 2, data: [order], generatedAt },
    { total: 1, data: [{ ...order, id: undefined }], generatedAt },
    { total: 1, data: [{ ...order, orderNo: undefined }], generatedAt },
    { total: 1, data: [{ ...order, organizationId: 'other' }], generatedAt },
    {
      total: 1,
      data: [{ ...order, businessType: OrderBusinessType.BUSINESS_TYPE_AE }],
      generatedAt,
    },
    { total: 1, data: [order], generatedAt: undefined },
  ])('拒绝不完整/错身份及缺真实字段的响应：%j', async (response) => {
    mocks.export.mockResolvedValue({ success: true, ...response });
    await expect(
      fetchOrderListExport({ filters: {}, total: 1 }, definition, context),
    ).rejects.toThrow();
  });

  it('公司未就绪不发送请求', async () => {
    await expect(
      fetchOrderListExport({ filters: {}, total: 0 }, definition, {
        ...context,
        organizationId: undefined,
      }),
    ).rejects.toThrow('公司尚未就绪');
    expect(mocks.export).not.toHaveBeenCalled();
  });

  it.each([undefined, false])(
    '省略或失败的成功标志%s不能导出封套内旧资料',
    async (success) => {
      mocks.export.mockResolvedValue({
        success,
        data: [order],
        total: 1,
        generatedAt,
      });
      await expect(
        fetchOrderListExport({ filters: {}, total: 1 }, definition, context),
      ).rejects.toThrow('导出失败');
    },
  );

  it.each(['sea-export', 'sea-import'])(
    '%s箱计划CSV只消费完整导出响应中的代码/名称投影',
    async (kind) => {
      const current = requireDefinition(kind);
      mocks.masterOptions.mockRejectedValue(new Error('不可另查可变字典'));
      mocks.export.mockResolvedValue({
        success: true,
        data: [
          {
            ...order,
            businessType: current.businessType,
            containerRequests: [
              {
                containerSpecId: 'old-code',
                quantity: 3,
                containerSpecCode: 'EXPORT-OLD',
                containerSpecName: '原箱型名称',
              },
              {
                containerSpecId: 'only-name',
                quantity: 1,
                containerSpecName: '历史停用特种箱',
              },
            ],
          },
        ],
        total: 1,
        generatedAt,
      });
      const { rows } = await fetchOrderListExport(
        { filters: {}, total: 1 },
        current,
        context,
      );
      const columns = orderExportColumns({
        containers: false,
        attachments: false,
      });
      const csv = serializeCsv(rows, columns);
      expect(csv).toContain('3×EXPORT-OLD, 1×历史停用特种箱');
      expect(csv).not.toContain('原箱型名称');
      expect(mocks.masterOptions).not.toHaveBeenCalled();
      expect(mocks.list).not.toHaveBeenCalled();
      expect(mocks.export).toHaveBeenCalledOnce();
    },
  );

  it('CSV区分省略optional与零，金额两位/数量保精度，中文和多行转义及本地时间正确', async () => {
    mocks.export.mockResolvedValue({
      success: true,
      data: [
        {
          ...order,
          cargoValue: '15362.50000000',
          cargoCurrency: 'USD',
          totalPackages: 0,
          totalGrossWeightKg: 12.345,
          totalVolumeCbm: 0,
          notes: '=公式, "中文"\n次行',
          eta: generatedAt,
          airExport: {
            documentMode: 'DIRECT',
            master: {
              content: { masterNo: '99912345675' },
              airlineName: '航司中文',
            },
          },
        },
      ],
      total: 1,
      generatedAt,
    });
    const { rows } = await fetchOrderListExport(
      { filters: {}, total: 1 },
      definition,
      context,
    );
    const columns = orderExportColumns({ containers: true, attachments: true });
    const value = (header: string) =>
      columns.find((column) => column.header === header)?.value(rows[0]);
    expect(value('货值')).toBe('15362.50');
    expect(value('件数')).toBe('0');
    expect(value('毛重(KG)')).toBe('12.345');
    expect(value('体积(m³)')).toBe('0');
    expect(value('附件数')).toBe('');
    expect(value('ETA')).toBe(formatDate(generatedAt, 'minute'));
    expect(value('主分单模式')).toBe('直单');
    expect(value('航司/船公司')).toBe('航司中文');
    const csv = serializeCsv(rows, columns);
    if (!csv) throw new Error('导出不应为空');
    expect(csv).toContain('"\'=公式, ""中文""\n次行"');
    expect(csv.startsWith('\uFEFF')).toBe(true);
    expect(csv).toContain('\r\n');
    expect(csv).not.toContain('15,362.50');
    expect(csv).not.toContain(generatedAt);
    expect(orderExportFileName('空运进口订单', generatedAt)).toBe(
      `空运进口订单_${formatDate(generatedAt).replaceAll(':', '-')}.csv`,
    );
  });

  it('无子资源权限不导出实际箱号/附件列，不把optional未出现补成零', () => {
    const columns = orderExportColumns({
      containers: false,
      attachments: false,
    });
    expect(
      columns.some((column) => ['实际箱号', '附件数'].includes(column.header)),
    ).toBe(false);
    const amount = columns.find((column) => column.header === '货值');
    if (!amount) throw new Error('货值列缺失');
    expect(amount.value({ id: 'a', orderNo: 'AI-1' })).toBe('');
    expect(amount.value({ id: 'a', orderNo: 'AI-1', cargoValue: '0' })).toBe(
      '0.00',
    );
  });
});
