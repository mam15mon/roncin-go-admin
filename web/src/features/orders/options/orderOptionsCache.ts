import {
  masterDataServiceListAirports,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import { ensureListResponse, unwrapList } from '@/utils/api';

/** 按组织读取订单字典；结果缓存由消费方当前工作区的 QueryClient 管理。 */
export async function getMasterDataOptions(
  organizationId: string,
): Promise<API.MasterDataItem[]> {
  if (!organizationId) throw new Error('缺少当前组织，无法加载订单主数据');
  return unwrapList(
    ensureListResponse(
      await masterDataServiceListOptions(),
      '主数据选项加载失败，请稍后重试',
    ),
  );
}

/** 读取当前组织首批港口，函数自身不持有跨工作区缓存。 */
export async function getCachedPorts(
  organizationId: string,
): Promise<API.Port[]> {
  if (!organizationId) throw new Error('缺少当前组织，无法加载港口主数据');
  return unwrapList(
    ensureListResponse(
      await masterDataServiceListPorts({
        page: 1,
        pageSize: 50,
        enabled: true,
      }),
      '港口主数据加载失败，请稍后重试',
    ),
  );
}

/** 读取当前组织首批机场，函数自身不持有跨工作区缓存。 */
export async function getCachedAirports(
  organizationId: string,
): Promise<API.Airport[]> {
  if (!organizationId) throw new Error('缺少当前组织，无法加载机场主数据');
  return unwrapList(
    ensureListResponse(
      await masterDataServiceListAirports({
        page: 1,
        pageSize: 50,
        enabled: true,
      }),
      '机场主数据加载失败，请稍后重试',
    ),
  );
}

/** 按组织与业务类型读取首批人员，消费方查询键须同时包含这两个身份。 */
export async function getOrderPersonnelOptions(
  organizationId: string,
  businessType: number,
): Promise<API.OrderPersonnelOption[]> {
  if (!organizationId) throw new Error('缺少当前组织，无法加载订单人员选项');
  return unwrapList(
    ensureListResponse(
      await orderServiceListPersonnelOptions({
        businessType,
        page: 1,
        pageSize: 200,
      }),
      '订单人员选项加载失败，请稍后重试',
    ),
  );
}
