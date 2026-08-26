import type { ReactNode } from 'react';

export type OrderDetailKind =
  | 'sea-export'
  | 'sea-import'
  | 'air-export'
  | 'air-import'
  | 'rail'
  | 'truck'
  | 'customs';

/** 订单详情聚合数据接口 */
export interface OrderDetailData {
  // 基础信息
  id?: string;
  orderNo?: string;
  businessTypeTitle?: string;
  status?: string;
  isLocked?: boolean;
  canModify?: boolean;
  createdAt?: string;
  updatedAt?: string;

  // 客户与商务
  customerName?: string;
  customerId?: string;
  customerReferenceNo?: string;
  internalReferenceNo?: string;
  tradeTermName?: string;
  paymentTermName?: string;
  bookingAgentName?: string;
  carrierName?: string;
  contractNo?: string;
  serviceTypeNames?: string[];
  cargoValueWithCurrency?: string;
  insurancePremiumWithCurrency?: string;

  // 航程与物流节点
  originName?: string;
  originCode?: string;
  destinationName?: string;
  destinationCode?: string;
  dischargeName?: string;
  transitName?: string;
  vesselVoyage?: string;
  etd?: string;
  eta?: string;
  loadingTerms?: string;
  siCutoff?: string;
  docCutoff?: string;
  customsCutoff?: string;
  vgmCutoff?: string;

  // 货物与计量
  totalPackages?: number;
  packageUnit?: string;
  grossWeightKg?: number;
  volumeCbm?: number;

  // 备注信息
  bookingNotes?: string;
  allocationNotes?: string;
  operationNotes?: string;
  notes?: string;

  // 子集合数据
  shippingDocuments?: Array<{
    id?: string;
    masterNo?: string;
    houseNo?: string;
    masterDocumentType?: string;
    masterReleaseMethod?: string;
    releaseType?: string;
    status?: string;
    createdAt?: string;
  }>;

  containers?: Array<{
    id?: string;
    containerNo?: string;
    sealNo?: string;
    containerSpecName?: string;
    grossWeightKg?: number;
    volumeCbm?: number;
    note?: string;
  }>;

  cargoItems?: Array<{
    id?: string;
    cargoName?: string;
    packageCount?: number;
    grossWeightKg?: number;
    volumeCbm?: number;
    netWeightKg?: number;
    note?: string;
  }>;

  milestones?: Array<{
    id?: string;
    type?: string;
    occurredAt?: string;
    confirmedAt?: string;
    note?: string;
  }>;

  attachments?: Array<{
    id?: string;
    docType?: string;
    fileName?: string;
    fileSize?: string | number;
    mimeType?: string;
    createdAt?: string;
  }>;

  personnel?: Array<{
    id?: string;
    roleName?: string;
    userId?: string;
    userName?: string;
  }>;
}

export interface OrderDetailTemplateProps {
  /** 订单品类标识（如 'sea-export'） */
  orderKind?: OrderDetailKind;
  /** 业务类型标题，如 "海运出口订单" */
  title?: string;
  /** 订单聚合数据对象 */
  data: OrderDetailData;
  /** 加载中状态 */
  loading?: boolean;
  /** 返回列表回调 */
  onBack?: () => void;
  /** 刷新数据回调 */
  onRefresh?: () => void;
  /** 点击编辑订单回调（有权限且草稿/允许修改时） */
  onEdit?: () => void;
  /** 打开费用录入面板 */
  onOpenFees?: () => void;
  /** 打开里程碑履约面板 */
  onOpenMilestones?: () => void;
  /** 打开放货凭证 (POD) 面板 */
  onOpenReleasePod?: () => void;
  /** 打开异常标记与登记面板 */
  onOpenAbnormal?: () => void;
  /** 顶部右侧自定义操作按钮组插槽 */
  extraActions?: ReactNode[];
  /** 自定义扩展区块插槽 */
  children?: ReactNode;
}
