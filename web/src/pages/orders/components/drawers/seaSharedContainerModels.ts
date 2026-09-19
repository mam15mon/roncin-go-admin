import Decimal from 'decimal.js';
import { SeaSharedContainerStatus } from '@/enums.generated';

// 候选订单服务端分页大小（分页单位是订单，不是货物行）；共享箱列表一次取分页上限
export const CANDIDATE_PAGE_SIZE = 20;
export const CONTAINER_PAGE_SIZE = 200;
export const STATUS_CONFIRMED =
  SeaSharedContainerStatus.SEA_SHARED_CONTAINER_STATUS_CONFIRMED;

// 本地草稿分配项：携带完整身份与期望版本，翻页或跨页编辑后仍可正确提交
export type DraftAllocation = {
  orderId: string;
  houseBillId: string;
  cargoItemId: string;
  expectedOrderVersion: string;
  expectedLinkVersion: string;
  expectedHouseBillVersion: string;
  expectedCargoItemVersion: string;
  packageCount: number;
  grossWeightKg: string;
  volumeCbm: string;
};

export type CargoAllocationItem = {
  key: string;
  draft: DraftAllocation;
  orderNo: string;
  houseNo: string;
  cargoName: string;
  totalPackageCount: number;
  totalGrossWeightKg: string;
  totalVolumeCbm: string;
};

export type AllocationSummary = {
  totalPackages: number;
  totalWeight: string;
  totalVolume: string;
  allocPackages: number;
  allocWeight: string;
  allocVolume: string;
  diffPackages: number;
  diffWeight: string;
  diffVolume: string;
  isBalanced: boolean;
};

export const isZeroQuantity = (pkg: number, weight: string, volume: string) =>
  pkg <= 0 &&
  new Decimal(weight || 0).lte(0) &&
  new Decimal(volume || 0).lte(0);

// 展开当前候选页订单与货物行；草稿值从 drafts 读取（含不在本页展示的历史编辑）
export const buildFlatCargoList = (
  candidates: API.SeaSharedContainerCandidateOrder[],
  drafts: Record<string, DraftAllocation>,
): CargoAllocationItem[] => {
  const list: CargoAllocationItem[] = [];
  for (const order of candidates) {
    if (!order.cargoItems) continue;
    for (const cargo of order.cargoItems) {
      if (!order.orderId || !cargo.id || !order.houseBillId) continue;
      const key = `${order.orderId}:${cargo.id}`;
      const draft =
        drafts[key] ||
        ({
          orderId: order.orderId,
          houseBillId: order.houseBillId,
          cargoItemId: cargo.id,
          expectedOrderVersion: String(order.orderVersion || 1),
          expectedLinkVersion: String(order.linkVersion || 1),
          expectedHouseBillVersion: String(order.houseBillVersion || 1),
          expectedCargoItemVersion: String(cargo.version || 1),
          packageCount: 0,
          grossWeightKg: '0.000',
          volumeCbm: '0.000000',
        } satisfies DraftAllocation);
      list.push({
        key,
        draft,
        orderNo: order.orderNo || '',
        houseNo: order.houseNo || '',
        cargoName: cargo.cargoName || '未命名货物',
        totalPackageCount: cargo.packageCount || 0,
        totalGrossWeightKg: cargo.grossWeightKg || '0.000',
        totalVolumeCbm: cargo.volumeCbm || '0.000000',
      });
    }
  }
  return list;
};

// 本地实时汇总已分配件重尺与剩余
export const buildAllocationSummary = (
  selectedContainer: API.SeaSharedContainer | null,
  drafts: Record<string, DraftAllocation>,
): AllocationSummary => {
  if (!selectedContainer) {
    return {
      totalPackages: 0,
      totalWeight: '0.000',
      totalVolume: '0.000000',
      allocPackages: 0,
      allocWeight: '0.000',
      allocVolume: '0.000000',
      diffPackages: 0,
      diffWeight: '0.000',
      diffVolume: '0.000000',
      isBalanced: false,
    };
  }

  let allocPackages = 0;
  let allocWeight = new Decimal(0);
  let allocVolume = new Decimal(0);

  for (const item of Object.values(drafts)) {
    allocPackages += Number(item.packageCount || 0);
    allocWeight = allocWeight.plus(new Decimal(item.grossWeightKg || 0));
    allocVolume = allocVolume.plus(new Decimal(item.volumeCbm || 0));
  }

  const totalPackages = selectedContainer.packageCount || 0;
  const totalWeight = new Decimal(selectedContainer.grossWeightKg || 0);
  const totalVolume = new Decimal(selectedContainer.volumeCbm || 0);

  const diffPackages = totalPackages - allocPackages;
  const diffWeight = totalWeight.minus(allocWeight);
  const diffVolume = totalVolume.minus(allocVolume);

  const isBalanced =
    diffPackages === 0 && diffWeight.isZero() && diffVolume.isZero();

  return {
    totalPackages,
    totalWeight: totalWeight.toFixed(3),
    totalVolume: totalVolume.toFixed(6),
    allocPackages,
    allocWeight: allocWeight.toFixed(3),
    allocVolume: allocVolume.toFixed(6),
    diffPackages,
    diffWeight: diffWeight.toFixed(3),
    diffVolume: diffVolume.toFixed(6),
    isBalanced,
  };
};

// 构建提交 payload：以本地全部草稿为准（跨页新录入同样保留），零值项剔除
export const buildAllocationInputs = (
  drafts: Record<string, DraftAllocation>,
): API.SeaSharedContainerAllocationInput[] | undefined => {
  const inputs: API.SeaSharedContainerAllocationInput[] = [];
  for (const draft of Object.values(drafts)) {
    if (
      isZeroQuantity(draft.packageCount, draft.grossWeightKg, draft.volumeCbm)
    ) {
      continue;
    }
    inputs.push({
      orderId: draft.orderId,
      houseBillId: draft.houseBillId,
      cargoItemId: draft.cargoItemId,
      packageCount: draft.packageCount,
      grossWeightKg: draft.grossWeightKg,
      volumeCbm: draft.volumeCbm,
      expectedOrderVersion: draft.expectedOrderVersion,
      expectedLinkVersion: draft.expectedLinkVersion,
      expectedHouseBillVersion: draft.expectedHouseBillVersion,
      expectedCargoItemVersion: draft.expectedCargoItemVersion,
    });
  }
  return inputs;
};
