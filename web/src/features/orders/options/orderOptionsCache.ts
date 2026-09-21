import {
  masterDataServiceListAirports,
  masterDataServiceListOptions,
  masterDataServiceListPorts,
} from '@/services/roncin/masterDataService';
import { orderServiceListPersonnelOptions } from '@/services/roncin/orderService';
import { ensureListResponse, unwrapList } from '@/utils/api';

const masterOptionsCache = new Map<string, Promise<API.MasterDataItem[]>>();
const portsCache = new Map<string, Promise<API.Port[]>>();
const airportsCache = new Map<string, Promise<API.Airport[]>>();
const personnelOptionsCache = new Map<
  string,
  Promise<API.OrderPersonnelOption[]>
>();

/**
 * 按组织获取主数据字典，组织隔离并并发共享在途请求。
 */
export function getMasterDataOptions(
  organizationId: string,
): Promise<API.MasterDataItem[]> {
  if (!organizationId) {
    return Promise.reject(new Error('缺少当前组织，无法加载订单主数据'));
  }
  const req = masterOptionsCache.get(organizationId);
  if (!req) {
    let createdReq: Promise<API.MasterDataItem[]>;
    createdReq = masterDataServiceListOptions()
      .then((response) =>
        unwrapList(
          ensureListResponse(response, '主数据选项加载失败，请稍后重试'),
        ),
      )
      .catch((err) => {
        if (masterOptionsCache.get(organizationId) === createdReq) {
          masterOptionsCache.delete(organizationId);
        }
        throw err;
      });
    masterOptionsCache.set(organizationId, createdReq);
    return createdReq;
  }
  return req;
}

/**
 * 按组织获取首批港口数据。
 */
export function getCachedPorts(organizationId: string): Promise<API.Port[]> {
  if (!organizationId) {
    return Promise.reject(new Error('缺少当前组织，无法加载港口主数据'));
  }
  const req = portsCache.get(organizationId);
  if (!req) {
    let createdReq: Promise<API.Port[]>;
    createdReq = masterDataServiceListPorts({
      page: 1,
      pageSize: 50,
      enabled: true,
    })
      .then((response) =>
        unwrapList(
          ensureListResponse(response, '港口主数据加载失败，请稍后重试'),
        ),
      )
      .catch((err) => {
        if (portsCache.get(organizationId) === createdReq) {
          portsCache.delete(organizationId);
        }
        throw err;
      });
    portsCache.set(organizationId, createdReq);
    return createdReq;
  }
  return req;
}

/**
 * 按组织获取首批机场数据。
 */
export function getCachedAirports(
  organizationId: string,
): Promise<API.Airport[]> {
  if (!organizationId) {
    return Promise.reject(new Error('缺少当前组织，无法加载机场主数据'));
  }
  const req = airportsCache.get(organizationId);
  if (!req) {
    let createdReq: Promise<API.Airport[]>;
    createdReq = masterDataServiceListAirports({
      page: 1,
      pageSize: 50,
      enabled: true,
    })
      .then((response) =>
        unwrapList(
          ensureListResponse(response, '机场主数据加载失败，请稍后重试'),
        ),
      )
      .catch((err) => {
        if (airportsCache.get(organizationId) === createdReq) {
          airportsCache.delete(organizationId);
        }
        throw err;
      });
    airportsCache.set(organizationId, createdReq);
    return createdReq;
  }
  return req;
}

/**
 * 按组织和业务类别获取人员候选项（默认获取前 200 条）。
 */
export function getOrderPersonnelOptions(
  organizationId: string,
  businessType: number,
): Promise<API.OrderPersonnelOption[]> {
  if (!organizationId) {
    return Promise.reject(new Error('缺少当前组织，无法加载订单人员选项'));
  }
  const key = `${organizationId}:${businessType}`;
  const req = personnelOptionsCache.get(key);
  if (!req) {
    let createdReq: Promise<API.OrderPersonnelOption[]>;
    createdReq = orderServiceListPersonnelOptions({
      businessType,
      page: 1,
      pageSize: 200,
    })
      .then((response) =>
        unwrapList(
          ensureListResponse(response, '订单人员选项加载失败，请稍后重试'),
        ),
      )
      .catch((err) => {
        if (personnelOptionsCache.get(key) === createdReq) {
          personnelOptionsCache.delete(key);
        }
        throw err;
      });
    personnelOptionsCache.set(key, createdReq);
    return createdReq;
  }
  return req;
}

/**
 * 清除订单主数据缓存。
 *
 * @param organizationId 如果传入，则仅失效该组织的缓存；若不传，则清空所有组织缓存（如退出登录或全局刷新）。
 */
export function clearOrderMasterDataCache(organizationId?: string): void {
  if (!organizationId) {
    masterOptionsCache.clear();
    portsCache.clear();
    airportsCache.clear();
    personnelOptionsCache.clear();
    return;
  }

  masterOptionsCache.delete(organizationId);
  portsCache.delete(organizationId);
  airportsCache.delete(organizationId);
  for (const key of personnelOptionsCache.keys()) {
    if (key.startsWith(`${organizationId}:`)) {
      personnelOptionsCache.delete(key);
    }
  }
}
